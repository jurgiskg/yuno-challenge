// Package testdata generates the sample authorization requests used to develop
// and demo the failover engine.
//
// The requests are designed against the mock acquirer rules below so that, with
// AcquirerOne → AcquirerTwo → AcquirerThree routing:
//   - 26 (52%) are approved by the primary
//   - 20 (40%) are declined by the primary with a retriable reason, of which
//     8 are approved by the secondary, 9 fail over to the tertiary and 3 are
//     hard-declined by the secondary
//   - 4 (8%) are hard-declined by the primary
package testdata

import (
	"fmt"
	"math/rand/v2"
	"strconv"

	"yuno-challenge/acquirer"
	"yuno-challenge/merchant"

	"github.com/shopspring/decimal"
)

const MerchantID = "solarbazaar"

// BIN prefixes used in the generated cards.
const (
	BINVisaBanorte     = "453201" // AcquirerTwo blocks the 4532 range
	BINVisaBancolombia = "476134"
	BINMastercardItau  = "522860"
	BINMastercardRisky = "555544" // AcquirerOne recently flagged the 5555 range
	BINAmex            = "377752" // 15-digit PAN, 4-digit CVV
	BINElo             = "636368"
	BINHipercard       = "606282"
	// binTestIssuer covers the acquirer.Card* test cards the issuer hard-declines.
	binTestIssuer = "4000"
)

// Mock acquirer rules the generated requests are designed against.
var (
	// AcquirerOne has stopped accepting Mexico and flagged the 5555 BIN range.
	AcquirerOneRules = acquirer.Rules{
		AcceptedCountries:   []string{"CO", "BR", "CL"},
		AcceptedBINPrefixes: []string{"4532", "4761", "5228", "3777", "6363", "6062", binTestIssuer},
	}
	// AcquirerTwo has no Chilean license and blocks the 4532 BIN range.
	AcquirerTwoRules = acquirer.Rules{
		AcceptedCountries:   []string{"MX", "CO", "BR"},
		AcceptedBINPrefixes: []string{"4761", "5228", "5555", "3777", "6363", "6062", binTestIssuer},
	}
	// AcquirerThree approves 80% of requests at random.
	AcquirerThreeRules = acquirer.Rules{SuccessRate: 0.8}
)

type cardKind int

const (
	cardValid cardKind = iota
	cardExpired
	cardStolen
	cardInsufficientFunds
)

type profile struct {
	country merchant.Country
	bin     string
	kind    cardKind
}

// scenario is a group of requests that take the same path through the acquirers.
// Its profiles are cycled until count requests are generated.
type scenario struct {
	count    int
	profiles []profile
}

var scenarios = []scenario{
	// Approved by AcquirerOne.
	{26, []profile{
		{merchant.CountryColombia, BINVisaBanorte, cardValid},
		{merchant.CountryBrazil, BINElo, cardValid},
		{merchant.CountryChile, BINVisaBancolombia, cardValid},
		{merchant.CountryBrazil, BINHipercard, cardValid},
		{merchant.CountryColombia, BINMastercardItau, cardValid},
		{merchant.CountryChile, BINAmex, cardValid},
		{merchant.CountryBrazil, BINVisaBanorte, cardValid},
		{merchant.CountryColombia, BINVisaBancolombia, cardValid},
		{merchant.CountryChile, BINMastercardItau, cardValid},
		{merchant.CountryBrazil, BINMastercardItau, cardValid},
	}},
	// AcquirerOne declines (POLICY_DECLINE for MX, SUSPECTED_FRAUD for 5555), AcquirerTwo approves.
	{8, []profile{
		{merchant.CountryMexico, BINVisaBancolombia, cardValid},
		{merchant.CountryColombia, BINMastercardRisky, cardValid},
		{merchant.CountryMexico, BINMastercardItau, cardValid},
		{merchant.CountryBrazil, BINMastercardRisky, cardValid},
		{merchant.CountryMexico, BINAmex, cardValid},
		{merchant.CountryMexico, BINMastercardRisky, cardValid},
	}},
	// AcquirerOne and AcquirerTwo both decline with retriable reasons; AcquirerThree decides.
	{9, []profile{
		{merchant.CountryMexico, BINVisaBanorte, cardValid},
		{merchant.CountryChile, BINMastercardRisky, cardValid},
	}},
	// Hard-declined by AcquirerOne, so no failover.
	{4, []profile{
		{merchant.CountryColombia, binTestIssuer, cardStolen},
		{merchant.CountryBrazil, binTestIssuer, cardInsufficientFunds},
		{merchant.CountryChile, BINVisaBancolombia, cardExpired},
		{merchant.CountryColombia, BINMastercardItau, cardExpired},
	}},
	// AcquirerOne declines Mexico, then AcquirerTwo hard-declines, so failover stops before AcquirerThree.
	{3, []profile{
		{merchant.CountryMexico, binTestIssuer, cardStolen},
		{merchant.CountryMexico, binTestIssuer, cardInsufficientFunds},
		{merchant.CountryMexico, BINMastercardItau, cardExpired},
	}},
}

// usdRates are approximate local currency units per USD.
var usdRates = map[merchant.Currency]decimal.Decimal{
	merchant.CurrencyMXN: decimal.NewFromFloat(18.5),
	merchant.CurrencyCOP: decimal.NewFromInt(4100),
	merchant.CurrencyBRL: decimal.NewFromFloat(5.4),
	merchant.CurrencyCLP: decimal.NewFromInt(940),
}

var currencies = map[merchant.Country]merchant.Currency{
	merchant.CountryMexico:   merchant.CurrencyMXN,
	merchant.CountryColombia: merchant.CurrencyCOP,
	merchant.CountryBrazil:   merchant.CurrencyBRL,
	merchant.CountryChile:    merchant.CurrencyCLP,
}

var holderNames = map[merchant.Country][]string{
	merchant.CountryMexico:   {"Maria Lopez", "Jose Hernandez", "Guadalupe Martinez", "Luis Ramirez"},
	merchant.CountryColombia: {"Andres Gomez", "Valentina Rodriguez", "Camilo Restrepo", "Daniela Ospina"},
	merchant.CountryBrazil:   {"Joao Silva", "Ana Souza", "Pedro Oliveira", "Beatriz Santos"},
	merchant.CountryChile:    {"Matias Gonzalez", "Catalina Munoz", "Benjamin Rojas", "Sofia Diaz"},
}

// AuthorizationRequests returns the 50 sample authorization requests, grouped by
// scenario in the order documented on the package. The output is deterministic
// for a given seed.
func AuthorizationRequests(seed uint64) []merchant.CreateAuthorizationRequest {
	rng := rand.New(rand.NewPCG(seed, seed))

	var reqs []merchant.CreateAuthorizationRequest
	for _, s := range scenarios {
		for i := range s.count {
			reqs = append(reqs, newRequest(rng, s.profiles[i%len(s.profiles)]))
		}
	}
	return reqs
}

func newRequest(rng *rand.Rand, p profile) merchant.CreateAuthorizationRequest {
	currency := currencies[p.country]
	// $50.00 to $2000.00 USD equivalent.
	usd := decimal.New(5000+rng.Int64N(195001), -2)
	names := holderNames[p.country]

	return merchant.CreateAuthorizationRequest{
		MerchantID: MerchantID,
		Amount:     usd.Mul(usdRates[currency]).Round(currency.Decimals()),
		Currency:   currency,
		Country:    p.country,
		Card: merchant.CardDetails{
			Number:     cardNumber(rng, p),
			HolderName: names[rng.IntN(len(names))],
			Expiry:     expiry(rng, p.kind),
			CVV:        cvv(rng, p.bin),
		},
	}
}

func cardNumber(rng *rand.Rand, p profile) string {
	switch p.kind {
	case cardStolen:
		return acquirer.CardStolen
	case cardInsufficientFunds:
		return acquirer.CardInsufficientFunds
	}
	length := 16
	if p.bin == BINAmex {
		length = 15
	}
	number := p.bin
	for len(number) < length-1 {
		number += strconv.Itoa(rng.IntN(10))
	}
	return number + luhnCheckDigit(number)
}

// luhnCheckDigit returns the digit that makes number+digit pass the Luhn check.
func luhnCheckDigit(number string) string {
	sum := 0
	for i := range len(number) {
		d := int(number[len(number)-1-i] - '0')
		// Double every second digit counting from the right, starting with the
		// rightmost since the check digit will be appended after it.
		if i%2 == 0 {
			d *= 2
			if d > 9 {
				d -= 9
			}
		}
		sum += d
	}
	return strconv.Itoa((10 - sum%10) % 10)
}

// expiry returns a YYYYMM expiry, 2028-2031 for valid cards and 2023-2025 for expired ones.
func expiry(rng *rand.Rand, kind cardKind) string {
	year := 2028 + rng.IntN(4)
	if kind == cardExpired {
		year = 2023 + rng.IntN(3)
	}
	return fmt.Sprintf("%d%02d", year, 1+rng.IntN(12))
}

func cvv(rng *rand.Rand, bin string) string {
	if bin == BINAmex {
		return fmt.Sprintf("%04d", rng.IntN(10000))
	}
	return fmt.Sprintf("%03d", rng.IntN(1000))
}
