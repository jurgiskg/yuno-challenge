// Package acquirer holds the Acquirer interface, the mock acquirers' rules and
// the simulated issuer checks. The failover engine that routes requests across
// acquirers lives in the processor package.
package acquirer

import (
	"context"
	"errors"
	"math/rand/v2"
	"slices"
	"time"

	"yuno-challenge/authorization"
	"yuno-challenge/shared/country"
)

// Acquirer is implemented by every mock acquirer (see the acq* subpackages).
type Acquirer interface {
	// Name identifies the acquirer in routing config and attempt logs.
	Name() string
	Authorize(ctx context.Context, req authorization.Request) AuthorizationResponse
}

// Rules define which transactions a mock acquirer approves. Set either
// SuccessRate alone, or any combination of AcceptedCountries and AcceptedBINPrefixes.
type Rules struct {
	// AcceptedCountries lists the country codes the acquirer approves; others get
	// POLICY_DECLINE. Empty accepts every country.
	AcceptedCountries []country.Code
	// AcceptedBINPrefixes lists the card number prefixes the acquirer approves;
	// others get SUSPECTED_FRAUD. Empty accepts every card.
	AcceptedBINPrefixes []string
	// SuccessRate, when non-zero, is the share (0-1] of requests approved at
	// random; the rest get GENERIC_DECLINE.
	SuccessRate float64
}

func (r Rules) Validate() error {
	if r.SuccessRate < 0 || r.SuccessRate > 1 {
		return errors.New("success rate must be between 0 and 1")
	}
	if r.SuccessRate > 0 && (len(r.AcceptedCountries) > 0 || len(r.AcceptedBINPrefixes) > 0) {
		return errors.New("success rate cannot be combined with accepted countries or BIN prefixes")
	}
	return nil
}

// Check returns the decline reason if the rules reject the request, or false if they accept it.
func (r Rules) Check(req authorization.Request) (authorization.DeclineReason, bool) {
	switch {
	case r.SuccessRate > 0 && rand.Float64() >= r.SuccessRate:
		return authorization.ReasonGenericDecline, true
	case len(r.AcceptedCountries) > 0 && !slices.Contains(r.AcceptedCountries, req.Country):
		return authorization.ReasonPolicyDecline, true
	case len(r.AcceptedBINPrefixes) > 0 && !req.Card.HasAnyPrefix(r.AcceptedBINPrefixes):
		return authorization.ReasonSuspectedFraud, true
	}
	return "", false
}

type AuthorizationResponse struct {
	Approved bool
	// DeclineReason is empty when Approved is true.
	DeclineReason authorization.DeclineReason
}

func Approved() AuthorizationResponse {
	return AuthorizationResponse{Approved: true}
}

func Declined(reason authorization.DeclineReason) AuthorizationResponse {
	return AuthorizationResponse{DeclineReason: reason}
}

// Test cards the simulated issuer always hard-declines, whichever acquirer routes them.
const (
	CardStolen            = "4000000000000119"
	CardInsufficientFunds = "4000000000009995"
)

// IssuerDecline simulates the issuing bank's checks, which are the same whichever
// acquirer routes the transaction. It only returns non-retriable reasons, and
// returns false if the issuer would approve.
func IssuerDecline(card authorization.Card, now time.Time) (authorization.DeclineReason, bool) {
	switch {
	case len(card.Number) < 12 || (len(card.CVV) != 3 && len(card.CVV) != 4):
		return authorization.ReasonInvalidCard, true
	case !now.Before(time.Date(card.ExpiryYear, time.Month(card.ExpiryMonth)+1, 1, 0, 0, 0, 0, time.UTC)):
		return authorization.ReasonExpiredCard, true
	case card.Number == CardStolen:
		return authorization.ReasonStolenCard, true
	case card.Number == CardInsufficientFunds:
		return authorization.ReasonInsufficientFunds, true
	}
	return "", false
}
