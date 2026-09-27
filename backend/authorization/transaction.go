package authorization

import (
	"time"

	"yuno-challenge/country"
)

// Attempt is one call to one acquirer within a transaction's attempt chain.
type Attempt struct {
	Acquirer  string
	StartedAt time.Time
	Duration  time.Duration
	Approved  bool
	// DeclineReason is empty when Approved is true.
	DeclineReason DeclineReason
}

// Transaction is the outcome of processing one authorization request across
// the acquirers, with the full attempt chain.
type Transaction struct {
	ID         string
	CreatedAt  time.Time
	MerchantID string
	// BIN and Last4 identify the card for analytics without storing the full number.
	BIN      string
	Last4    string
	Amount   int64
	Currency string
	Country  country.Code
	// Attempts is in the order the acquirers were tried.
	Attempts []Attempt
	Approved bool
	// Acquirer is the acquirer that approved the transaction, empty if declined.
	Acquirer string
	// DeclineReason is the last acquirer's decline reason, empty if approved.
	DeclineReason DeclineReason
}
