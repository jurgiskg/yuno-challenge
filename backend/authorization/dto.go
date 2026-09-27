package authorization

import (
	"time"

	"yuno-challenge/country"
)

type Status string

const (
	StatusApproved Status = "APPROVED"
	StatusDeclined Status = "DECLINED"
)

func StatusOf(approved bool) Status {
	if approved {
		return StatusApproved
	}
	return StatusDeclined
}

type AttemptResponse struct {
	// Acquirer that was tried.
	Acquirer string `json:"acquirer" example:"AcquirerOne"`
	// When the attempt started.
	StartedAt time.Time `json:"startedAt" example:"2026-09-27T12:00:00Z"`
	// How long the acquirer took to respond, in milliseconds.
	DurationMs float64 `json:"durationMs" example:"1.25"`
	Status     Status  `json:"status" example:"DECLINED"`
	// Acquirer's decline reason, omitted when approved.
	DeclineReason DeclineReason `json:"declineReason,omitempty" swaggertype:"string" example:"POLICY_DECLINE"`
} // @name AttemptResponse

func NewAttemptResponses(attempts []Attempt) []AttemptResponse {
	out := make([]AttemptResponse, len(attempts))
	for i, a := range attempts {
		out[i] = AttemptResponse{
			Acquirer:      a.Acquirer,
			StartedAt:     a.StartedAt,
			DurationMs:    float64(a.Duration.Microseconds()) / 1000,
			Status:        StatusOf(a.Approved),
			DeclineReason: a.DeclineReason,
		}
	}
	return out
}

// AuthorizationResponse is the result of POST /authorizations.
type AuthorizationResponse struct {
	// Transaction ID.
	ID     string `json:"id" example:"txn_3f9a1c2b4d5e6f70"`
	Status Status `json:"status" example:"APPROVED"`
	// Acquirer that approved the transaction, omitted when declined.
	Acquirer string `json:"acquirer,omitempty" example:"AcquirerTwo"`
	// Final decline reason, omitted when approved.
	DeclineReason DeclineReason `json:"declineReason,omitempty" swaggertype:"string" example:"STOLEN_CARD"`
	// Every acquirer attempt, in the order they were tried.
	Attempts []AttemptResponse `json:"attempts"`
} // @name AuthorizationResponse

func NewAuthorizationResponse(txn Transaction) AuthorizationResponse {
	return AuthorizationResponse{
		ID:            txn.ID,
		Status:        StatusOf(txn.Approved),
		Acquirer:      txn.Acquirer,
		DeclineReason: txn.DeclineReason,
		Attempts:      NewAttemptResponses(txn.Attempts),
	}
}

// TransactionResponse is one entry of the authorization log.
type TransactionResponse struct {
	ID         string    `json:"id" example:"txn_3f9a1c2b4d5e6f70"`
	CreatedAt  time.Time `json:"createdAt" example:"2026-09-27T12:00:00Z"`
	MerchantID string    `json:"merchantId" example:"solarbazaar"`
	// First 6 digits of the card number.
	BIN string `json:"bin" example:"453201"`
	// Last 4 digits of the card number.
	Last4 string `json:"last4" example:"0366"`
	// Amount in minor units of the currency (e.g. centavos).
	AmountMinor int64        `json:"amountMinor" example:"1450000"`
	Currency    string       `json:"currency" example:"MXN"`
	Country     country.Code `json:"country" swaggertype:"string" example:"MX"`
	// Acquirer order as ranked when the transaction arrived, best first.
	RoutingOrder []string `json:"routingOrder" example:"AcquirerOne,AcquirerTwo,AcquirerThree"`
	Status       Status   `json:"status" example:"APPROVED"`
	// Acquirer that approved the transaction, omitted when declined.
	Acquirer string `json:"acquirer,omitempty" example:"AcquirerTwo"`
	// Final decline reason, omitted when approved.
	DeclineReason DeclineReason `json:"declineReason,omitempty" swaggertype:"string" example:"STOLEN_CARD"`
	// Every acquirer attempt, in the order they were tried.
	Attempts []AttemptResponse `json:"attempts"`
} // @name TransactionResponse

func NewTransactionResponse(txn Transaction) TransactionResponse {
	return TransactionResponse{
		ID:            txn.ID,
		CreatedAt:     txn.CreatedAt,
		MerchantID:    txn.MerchantID,
		BIN:           txn.BIN,
		Last4:         txn.Last4,
		AmountMinor:   txn.Amount,
		Currency:      txn.Currency,
		Country:       txn.Country,
		RoutingOrder:  txn.RoutingOrder,
		Status:        StatusOf(txn.Approved),
		Acquirer:      txn.Acquirer,
		DeclineReason: txn.DeclineReason,
		Attempts:      NewAttemptResponses(txn.Attempts),
	}
}

type ListAuthorizationsResponse struct {
	// Every processed transaction, oldest first.
	Transactions []TransactionResponse `json:"transactions"`
} // @name ListAuthorizationsResponse
