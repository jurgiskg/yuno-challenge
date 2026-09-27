package merchant

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"yuno-challenge/sharedgin"

	"github.com/shopspring/decimal"
)

const (
	ErrorCodeValidationError = sharedgin.ErrorCodeValidation
	ErrorCodeInvalidCard     = "invalid-card"
	ErrorCodeInvalidExpiry   = "invalid-expiry"
)

type (
	Country  string
	Currency string
)

const (
	CountryMexico   Country = "MX"
	CountryColombia Country = "CO"
	CountryBrazil   Country = "BR"
	CountryChile    Country = "CL"

	CurrencyMXN Currency = "MXN"
	CurrencyCOP Currency = "COP"
	CurrencyBRL Currency = "BRL"
	CurrencyCLP Currency = "CLP"
)

// countryCurrency is the local currency SolarBazaar charges in for each country.
var countryCurrency = map[Country]Currency{
	CountryMexico:   CurrencyMXN,
	CountryColombia: CurrencyCOP,
	CountryBrazil:   CurrencyBRL,
	CountryChile:    CurrencyCLP,
}

func (c Country) Validate() error {
	switch c {
	case CountryMexico, CountryColombia, CountryBrazil, CountryChile:
		return nil
	default:
		return fmt.Errorf("invalid country: %s. Country must be one of MX, CO, BR, CL", c)
	}
}

// Decimals is the number of minor unit digits of the currency (ISO 4217).
func (c Currency) Decimals() int32 {
	if c == CurrencyCLP {
		return 0
	}
	return 2
}

func (c Currency) Validate() error {
	switch c {
	case CurrencyMXN, CurrencyCOP, CurrencyBRL, CurrencyCLP:
		return nil
	default:
		return fmt.Errorf("invalid currency: %s. Currency must be one of MXN, COP, BRL, CLP", c)
	}
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
	Currency Currency `json:"currency" example:"MXN"`
	// ISO 3166-1 alpha-2 country of the transaction.
	Country Country `json:"country" example:"MX"`
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
	if err := r.Currency.Validate(); err != nil {
		return ErrorCodeValidationError, err
	}
	if decimals := r.Currency.Decimals(); !r.Amount.Equal(r.Amount.Truncate(decimals)) {
		return ErrorCodeValidationError, fmt.Errorf("amount %s has more than %d decimal places allowed for %s", r.Amount, decimals, r.Currency)
	}
	if err := r.Country.Validate(); err != nil {
		return ErrorCodeValidationError, err
	}
	if expected := countryCurrency[r.Country]; r.Currency != expected {
		return ErrorCodeValidationError, fmt.Errorf("currency %s does not match country %s, expected %s", r.Currency, r.Country, expected)
	}
	return r.Card.Validate()
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
