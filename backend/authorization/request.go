package authorization

import (
	"strings"

	"yuno-challenge/shared/country"
	"yuno-challenge/shared/currency"
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
	Amount   int64
	Currency currency.Code
	Country  country.Code
}
