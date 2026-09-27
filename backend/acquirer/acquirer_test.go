package acquirer_test

import (
	"testing"

	"yuno-challenge/acquirer"
	"yuno-challenge/authorization"
	"yuno-challenge/shared/country"
)

func TestRules_SeededSuccessRate(t *testing.T) {
	rules := acquirer.Rules{SuccessRate: 0.8, Seed: 42}
	approved := 0
	const n = 2000
	for i := range n {
		req := authorization.Request{
			MerchantID: "solarbazaar",
			Card:       authorization.Card{Number: "4532015112830366"},
			Amount:     int64(100000 + i),
			Currency:   "MXN",
			Country:    country.MX,
		}
		reason, declined := rules.Check(req)
		if again, declinedAgain := rules.Check(req); again != reason || declinedAgain != declined {
			t.Fatalf("request %d: got %q then %q, want the same answer", i, reason, again)
		}
		if !declined {
			approved++
		}
	}
	if rate := float64(approved) / n; rate < 0.77 || rate > 0.83 {
		t.Fatalf("approval rate = %.3f, want about 0.8", rate)
	}
}
