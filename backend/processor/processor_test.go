package processor

import (
	"context"
	"slices"
	"testing"
	"time"

	"yuno-challenge/acquirer"
	"yuno-challenge/authorization"
	"yuno-challenge/shared/country"
	"yuno-challenge/shared/currency"

	"go.uber.org/zap"
)

// stubAcquirer returns resp after delay, or blocks until the context is done if
// delay is longer than the processor's attempt timeout.
type stubAcquirer struct {
	name  string
	resp  acquirer.AuthorizationResponse
	delay time.Duration
	calls int
}

func (s *stubAcquirer) Name() string { return s.name }

func (s *stubAcquirer) Authorize(ctx context.Context, _ authorization.Request) acquirer.AuthorizationResponse {
	s.calls++
	select {
	case <-time.After(s.delay):
		return s.resp
	case <-ctx.Done():
		return acquirer.Declined(authorization.ReasonTimeout)
	}
}

func newTestProcessor(t *testing.T, acquirers ...*stubAcquirer) (*Processor, *authorization.Store) {
	t.Helper()
	return newTestProcessorWithRanking(t, true, acquirers...)
}

func newTestProcessorWithRanking(t *testing.T, dynamicRanking bool, acquirers ...*stubAcquirer) (*Processor, *authorization.Store) {
	t.Helper()
	list := make([]acquirer.Acquirer, len(acquirers))
	for i, a := range acquirers {
		list[i] = a
	}
	store := authorization.NewStore()
	p, err := New(list, dynamicRanking, store, zap.NewNop().Sugar())
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return p, store
}

var testRequest = authorization.Request{
	MerchantID: "solarbazaar",
	Card:       authorization.Card{Number: "4532015112830366", HolderName: "Maria Lopez", ExpiryMonth: 12, ExpiryYear: 2028, CVV: "123"},
	Amount:     1450000,
	Currency:   currency.MXN,
	Country:    country.MX,
}

func attemptSummary(txn authorization.Transaction) []string {
	out := make([]string, len(txn.Attempts))
	for i, a := range txn.Attempts {
		out[i] = a.Acquirer + ":" + string(a.DeclineReason)
	}
	return out
}

func TestProcessor_Process(t *testing.T) {
	approved := acquirer.Approved()
	tests := []struct {
		name         string
		responses    [3]acquirer.AuthorizationResponse
		wantApproved bool
		wantAcquirer string
		wantReason   authorization.DeclineReason
		wantAttempts []string
	}{
		{
			name:         "primary approves",
			responses:    [3]acquirer.AuthorizationResponse{approved, approved, approved},
			wantApproved: true,
			wantAcquirer: "One",
			wantAttempts: []string{"One:"},
		},
		{
			name:         "retriable decline fails over to secondary",
			responses:    [3]acquirer.AuthorizationResponse{acquirer.Declined(authorization.ReasonPolicyDecline), approved, approved},
			wantApproved: true,
			wantAcquirer: "Two",
			wantAttempts: []string{"One:POLICY_DECLINE", "Two:"},
		},
		{
			name:         "fails over to tertiary",
			responses:    [3]acquirer.AuthorizationResponse{acquirer.Declined(authorization.ReasonSuspectedFraud), acquirer.Declined(authorization.ReasonGenericDecline), approved},
			wantApproved: true,
			wantAcquirer: "Three",
			wantAttempts: []string{"One:SUSPECTED_FRAUD", "Two:GENERIC_DECLINE", "Three:"},
		},
		{
			name:         "hard decline from primary stops",
			responses:    [3]acquirer.AuthorizationResponse{acquirer.Declined(authorization.ReasonStolenCard), approved, approved},
			wantReason:   authorization.ReasonStolenCard,
			wantAttempts: []string{"One:STOLEN_CARD"},
		},
		{
			name:         "hard decline from secondary stops",
			responses:    [3]acquirer.AuthorizationResponse{acquirer.Declined(authorization.ReasonPolicyDecline), acquirer.Declined(authorization.ReasonInsufficientFunds), approved},
			wantReason:   authorization.ReasonInsufficientFunds,
			wantAttempts: []string{"One:POLICY_DECLINE", "Two:INSUFFICIENT_FUNDS"},
		},
		{
			name:         "every acquirer declines",
			responses:    [3]acquirer.AuthorizationResponse{acquirer.Declined(authorization.ReasonPolicyDecline), acquirer.Declined(authorization.ReasonSuspectedFraud), acquirer.Declined(authorization.ReasonGenericDecline)},
			wantReason:   authorization.ReasonGenericDecline,
			wantAttempts: []string{"One:POLICY_DECLINE", "Two:SUSPECTED_FRAUD", "Three:GENERIC_DECLINE"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, store := newTestProcessor(t,
				&stubAcquirer{name: "One", resp: tt.responses[0]},
				&stubAcquirer{name: "Two", resp: tt.responses[1]},
				&stubAcquirer{name: "Three", resp: tt.responses[2]},
			)

			txn := p.Process(context.Background(), testRequest)

			if txn.Approved != tt.wantApproved || txn.Acquirer != tt.wantAcquirer || txn.DeclineReason != tt.wantReason {
				t.Errorf("got approved=%v acquirer=%q reason=%q, want approved=%v acquirer=%q reason=%q",
					txn.Approved, txn.Acquirer, txn.DeclineReason, tt.wantApproved, tt.wantAcquirer, tt.wantReason)
			}
			if got := attemptSummary(txn); !slices.Equal(got, tt.wantAttempts) {
				t.Errorf("attempts = %v, want %v", got, tt.wantAttempts)
			}
			if txn.ID == "" || txn.BIN != "453201" || txn.Last4 != "0366" {
				t.Errorf("got id=%q bin=%q last4=%q", txn.ID, txn.BIN, txn.Last4)
			}
			if stored := store.List(); len(stored) != 1 || stored[0].ID != txn.ID {
				t.Errorf("store holds %d transactions, want the processed one", len(stored))
			}
		})
	}
}

func TestProcessor_AttemptTimeoutFailsOver(t *testing.T) {
	slow := &stubAcquirer{name: "Slow", resp: acquirer.Approved(), delay: time.Second}
	fast := &stubAcquirer{name: "Fast", resp: acquirer.Approved()}
	p, _ := newTestProcessor(t, slow, fast)
	p.attemptTimeout = 10 * time.Millisecond

	txn := p.Process(context.Background(), testRequest)

	if !txn.Approved || txn.Acquirer != "Fast" {
		t.Fatalf("got approved=%v acquirer=%q, want approved by Fast", txn.Approved, txn.Acquirer)
	}
	if got, want := attemptSummary(txn), []string{"Slow:TIMEOUT", "Fast:"}; !slices.Equal(got, want) {
		t.Fatalf("attempts = %v, want %v", got, want)
	}
	if d := txn.Attempts[0].Duration; d < 10*time.Millisecond || d > 500*time.Millisecond {
		t.Fatalf("timed out attempt took %s, want about the attempt timeout", d)
	}
}

func TestProcessor_CanceledContextStops(t *testing.T) {
	one := &stubAcquirer{name: "One", resp: acquirer.Approved()}
	p, _ := newTestProcessor(t, one)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	txn := p.Process(ctx, testRequest)

	if txn.Approved || len(txn.Attempts) != 0 || one.calls != 0 {
		t.Fatalf("got approved=%v attempts=%d calls=%d, want no attempts", txn.Approved, len(txn.Attempts), one.calls)
	}
}

func TestProcessor_RankingReordersOnRetriableDeclines(t *testing.T) {
	one := &stubAcquirer{name: "One", resp: acquirer.Declined(authorization.ReasonPolicyDecline)}
	two := &stubAcquirer{name: "Two", resp: acquirer.Approved()}
	p, _ := newTestProcessor(t, one, two)

	p.Process(context.Background(), testRequest)
	txn := p.Process(context.Background(), testRequest)

	if got, want := attemptSummary(txn), []string{"Two:"}; !slices.Equal(got, want) {
		t.Fatalf("second transaction attempts = %v, want %v (One demoted)", got, want)
	}
}

func TestProcessor_HardDeclinesDontAffectRanking(t *testing.T) {
	one := &stubAcquirer{name: "One", resp: acquirer.Declined(authorization.ReasonExpiredCard)}
	two := &stubAcquirer{name: "Two", resp: acquirer.Approved()}
	p, _ := newTestProcessor(t, one, two)

	for range 5 {
		p.Process(context.Background(), testRequest)
	}

	if got, want := p.ranking.Order(), []string{"One", "Two"}; !slices.Equal(got, want) {
		t.Fatalf("order = %v, want %v", got, want)
	}
}

func TestProcessor_FixedOrderIgnoresOutcomes(t *testing.T) {
	one := &stubAcquirer{name: "One", resp: acquirer.Declined(authorization.ReasonPolicyDecline)}
	two := &stubAcquirer{name: "Two", resp: acquirer.Approved()}
	p, _ := newTestProcessorWithRanking(t, false, one, two)

	for range 5 {
		txn := p.Process(context.Background(), testRequest)
		if got, want := attemptSummary(txn), []string{"One:POLICY_DECLINE", "Two:"}; !slices.Equal(got, want) {
			t.Fatalf("attempts = %v, want %v (configured order kept)", got, want)
		}
	}
}
