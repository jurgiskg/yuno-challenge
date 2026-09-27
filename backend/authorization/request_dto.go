package authorization

import (
	"fmt"
	"maps"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"yuno-challenge/shared/country"
	"yuno-challenge/shared/currency"
	"yuno-challenge/shared/sharedgin"

	"github.com/shopspring/decimal"
)

const (
	ErrorCodeValidationError = sharedgin.ErrorCodeValidation
	ErrorCodeInvalidCard     = "invalid-card"
	ErrorCodeInvalidExpiry   = "invalid-expiry"
)

// countryCurrency is the local currency SolarBazaar charges in for each country
// it operates in. Requests from any other country are rejected.
var countryCurrency = map[country.Code]currency.Code{
	country.MX: currency.MXN,
	country.CO: currency.COP,
	country.BR: currency.BRL,
	country.CL: currency.CLP,
}

func validateCountry(c country.Code) error {
	if !c.Valid() {
		return fmt.Errorf("invalid country: %s. Country must be an ISO 3166-1 alpha-2 code", c)
	}
	if _, ok := countryCurrency[c]; !ok {
		return fmt.Errorf("unsupported country: %s. Country must be one of MX, CO, BR, CL", c)
	}
	return nil
}

func validateCurrency(c currency.Code) error {
	if !c.Valid() {
		return fmt.Errorf("invalid currency: %s. Currency must be an ISO 4217 code", c)
	}
	if !slices.Contains(slices.Collect(maps.Values(countryCurrency)), c) {
		return fmt.Errorf("unsupported currency: %s. Currency must be one of MXN, COP, BRL, CLP", c)
	}
	return nil
}

type CardDetails struct {
	// Card number (PAN), digits only. The first 6-8 digits are the BIN.
	Number string `json:"number" example:"4532015112830366"`
	// Name printed on the card.
	HolderName string `json:"holderName" example:"Maria Lopez"`
	// Card expiry in format YYYYMM.
	Expiry string `json:"expiry" example:"202812"`
	// Card verification value, 3 or 4 digits.
	CVV string `json:"cvv" example:"123"`
} // @name CardDetails

type CreateAuthorizationRequest struct {
	// Merchant ID the transaction is processed for.
	MerchantID string `json:"merchantId" example:"solarbazaar"`
	// Amount in major units of the currency. Accepts a JSON string or number, e.g. "14500.00" MXN or "850000" CLP.
	Amount decimal.Decimal `json:"amount" swaggertype:"string" example:"14500.00"`
	// Transaction currency. Must be the local currency of country.
	Currency currency.Code `json:"currency" swaggertype:"string" example:"MXN"`
	// ISO 3166-1 alpha-2 country of the transaction.
	Country country.Code `json:"country" swaggertype:"string" example:"MX"`
	// Customer card details.
	Card CardDetails `json:"card"`
} // @name CreateAuthorizationRequest

func (r CreateAuthorizationRequest) Validate() (string, error) {
	if strings.TrimSpace(r.MerchantID) == "" {
		return ErrorCodeValidationError, fmt.Errorf("merchantId is required")
	}
	if r.Amount.LessThanOrEqual(decimal.Zero) {
		return ErrorCodeValidationError, fmt.Errorf("amount must be greater than 0")
	}
	if err := validateCurrency(r.Currency); err != nil {
		return ErrorCodeValidationError, err
	}
	if decimals := r.Currency.Decimals(); !r.Amount.Equal(r.Amount.Truncate(decimals)) {
		return ErrorCodeValidationError, fmt.Errorf("amount %s has more than %d decimal places allowed for %s", r.Amount, decimals, r.Currency)
	}
	if err := validateCountry(r.Country); err != nil {
		return ErrorCodeValidationError, err
	}
	if expected := countryCurrency[r.Country]; r.Currency != expected {
		return ErrorCodeValidationError, fmt.Errorf("currency %s does not match country %s, expected %s", r.Currency, r.Country, expected)
	}
	return r.Card.Validate()
}

// ToRequest converts the request to the acquirers' format, with the amount in
// minor units. Call it only after Validate has passed.
func (r CreateAuthorizationRequest) ToRequest() Request {
	year, _ := strconv.Atoi(r.Card.Expiry[:4])
	month, _ := strconv.Atoi(r.Card.Expiry[4:6])
	return Request{
		MerchantID: r.MerchantID,
		Card: Card{
			Number:      r.Card.Number,
			HolderName:  r.Card.HolderName,
			ExpiryMonth: month,
			ExpiryYear:  year,
			CVV:         r.Card.CVV,
		},
		Amount:   r.Amount.Shift(r.Currency.Decimals()).IntPart(),
		Currency: r.Currency,
		Country:  r.Country,
	}
}

var (
	cardNumberRegex = regexp.MustCompile(`^\d{12,19}$`)
	cvvRegex        = regexp.MustCompile(`^\d{3,4}$`)
	expiryRegex     = regexp.MustCompile(`^\d{4}\d{2}$`)
)

// Validate checks the card details are well-formed. It deliberately does not
// run a Luhn check or reject past expiry dates: INVALID_CARD and EXPIRED_CARD
// are acquirer decline reasons, so those cards must reach the acquirers.
func (c CardDetails) Validate() (string, error) {
	// Never echo the card number or CVV back in error messages.
	if !cardNumberRegex.MatchString(c.Number) {
		return ErrorCodeInvalidCard, fmt.Errorf("card.number must be 12 to 19 digits")
	}
	if strings.TrimSpace(c.HolderName) == "" {
		return ErrorCodeInvalidCard, fmt.Errorf("card.holderName is required")
	}
	if !cvvRegex.MatchString(c.CVV) {
		return ErrorCodeInvalidCard, fmt.Errorf("card.cvv must be 3 or 4 digits")
	}
	if err := validateExpiry(c.Expiry); err != nil {
		return ErrorCodeInvalidExpiry, err
	}
	return "", nil
}

func validateExpiry(expiry string) error {
	if !expiryRegex.MatchString(expiry) {
		return fmt.Errorf("invalid card.expiry: %s. Expiry must be in format YYYYMM", expiry)
	}
	expiryMonth, _ := strconv.Atoi(expiry[4:6])
	if expiryMonth < 1 || expiryMonth > 12 {
		return fmt.Errorf("invalid card.expiry: %s. Expiry month must be between 01 and 12", expiry)
	}
	return nil
}
