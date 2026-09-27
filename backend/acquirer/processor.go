package acquirer

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"yuno-challenge/acquirer/ranking"
	"yuno-challenge/authorization"

	"go.uber.org/zap"
)

// DefaultAttemptTimeout bounds a single acquirer call; slower calls are treated
// as a retriable TIMEOUT decline so the transaction fails over.
const DefaultAttemptTimeout = 2 * time.Second

// Processor authorizes transactions by trying acquirers in ranking order until
// one approves, failing over only on retriable declines.
type Processor struct {
	acquirers      map[string]Acquirer
	ranking        *ranking.Ranking
	store          *authorization.Store
	attemptTimeout time.Duration
	logger         *zap.SugaredLogger
	now            func() time.Time
}

// NewProcessor returns a processor over the given acquirers. Their order is the
// configured routing order: it is used as-is until the ranking has recorded
// enough outcomes to reorder them.
func NewProcessor(acquirers []Acquirer, store *authorization.Store, logger *zap.SugaredLogger) (*Processor, error) {
	names := make([]string, len(acquirers))
	byName := make(map[string]Acquirer, len(acquirers))
	for i, a := range acquirers {
		names[i] = a.Name()
		byName[a.Name()] = a
	}
	rank, err := ranking.New(names)
	if err != nil {
		return nil, fmt.Errorf("failed to create acquirer ranking: %w", err)
	}
	if store == nil {
		return nil, errors.New("store is required")
	}
	return &Processor{
		acquirers:      byName,
		ranking:        rank,
		store:          store,
		attemptTimeout: DefaultAttemptTimeout,
		logger:         logger,
		now:            time.Now,
	}, nil
}

// Process authorizes req, trying acquirers best-ranked first. It stops at the
// first approval, the first non-retriable decline, or when every acquirer has
// declined. The transaction is saved to the store before it is returned.
func (p *Processor) Process(ctx context.Context, req authorization.Request) authorization.Transaction {
	order := p.ranking.Order()
	txn := authorization.Transaction{
		ID:         newTransactionID(),
		CreatedAt:  p.now(),
		MerchantID: req.MerchantID,
		BIN:        prefix(req.Card.Number, 6),
		Last4:      suffix(req.Card.Number, 4),
		Amount:     req.Amount,
		Currency:   req.Currency,
		Country:    req.Country,
		// Order is a fresh slice per call, so the transaction can keep it.
		RoutingOrder: order,
		Attempts:     make([]authorization.Attempt, 0, len(order)),
	}

	for _, name := range order {
		// The caller gave up (e.g. client disconnected); don't start more attempts.
		if ctx.Err() != nil {
			txn.DeclineReason = authorization.ReasonTimeout
			break
		}

		attempt := p.attempt(ctx, p.acquirers[name], req)
		txn.Attempts = append(txn.Attempts, attempt)

		if attempt.Approved {
			txn.Approved = true
			txn.Acquirer = name
			txn.DeclineReason = ""
			p.record(name, true)
			break
		}

		txn.DeclineReason = attempt.DeclineReason
		if !attempt.DeclineReason.Retriable() {
			// Issuer-side decline: every acquirer would decline, and it isn't this
			// acquirer's fault, so don't count it against its ranking.
			break
		}
		p.record(name, false)
	}

	p.store.Save(txn)
	p.log(txn, order)
	return txn
}

// attempt calls the acquirer, treating a call that outlives the attempt
// timeout as a TIMEOUT decline.
func (p *Processor) attempt(ctx context.Context, acq Acquirer, req authorization.Request) authorization.Attempt {
	ctx, cancel := context.WithTimeout(ctx, p.attemptTimeout)
	defer cancel()

	started := p.now()
	// Buffered so the goroutine can finish and exit even after we stop waiting.
	result := make(chan AuthorizationResponse, 1)
	go func() { result <- acq.Authorize(ctx, req) }()

	var resp AuthorizationResponse
	select {
	case resp = <-result:
	case <-ctx.Done():
		resp = Declined(authorization.ReasonTimeout)
	}

	return authorization.Attempt{
		Acquirer:      acq.Name(),
		StartedAt:     started,
		Duration:      p.now().Sub(started),
		Approved:      resp.Approved,
		DeclineReason: resp.DeclineReason,
	}
}

func (p *Processor) record(name string, approved bool) {
	if err := p.ranking.Record(name, approved); err != nil {
		p.logger.Errorw("Failed to record acquirer outcome", "acquirer", name, "error", err)
	}
}

func (p *Processor) log(txn authorization.Transaction, order []string) {
	chain := make([]string, len(txn.Attempts))
	for i, a := range txn.Attempts {
		outcome := "APPROVED"
		if !a.Approved {
			outcome = string(a.DeclineReason)
		}
		chain[i] = fmt.Sprintf("%s:%s(%s)", a.Acquirer, outcome, a.Duration)
	}
	fields := []any{
		"transactionId", txn.ID,
		"merchantId", txn.MerchantID,
		"country", txn.Country,
		"bin", txn.BIN,
		"routingOrder", order,
		"attempts", len(txn.Attempts),
		"chain", chain,
	}
	if txn.Approved {
		p.logger.Infow("Transaction approved by "+txn.Acquirer, fields...)
		return
	}
	p.logger.Infow(fmt.Sprintf("Transaction declined after %d attempts", len(txn.Attempts)),
		append(fields, "declineReason", txn.DeclineReason)...)
}

func newTransactionID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b) // crypto/rand.Read never returns an error
	return "txn_" + hex.EncodeToString(b)
}

func prefix(s string, n int) string {
	return s[:min(n, len(s))]
}

func suffix(s string, n int) string {
	return s[max(len(s)-n, 0):]
}
