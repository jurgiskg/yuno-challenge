package authorization

import (
	"strings"

	"yuno-challenge/country"
)

// Card is the customer's card as sent to the acquirers.
type Card struct {
	Number      string
	HolderName  string
	ExpiryMonth int
	ExpiryYear  int
	CVV         string
}

// HasAnyPrefix reports whether the card's number starts with any of the given BIN prefixes.
func (c Card) HasAnyPrefix(prefixes []string) bool {
	for _, p := range prefixes {
		if strings.HasPrefix(c.Number, p) {
			return true
		}
	}
	return false
}

// Request is an authorization request as sent to the acquirers.
type Request struct {
	MerchantID string
	Card       Card
	// Amount is in minor units (e.g. centavos).
	Amount int64
	// Currency is an ISO 4217 code: MXN, COP, BRL or CLP.
	Currency string
	Country  country.Code
}
