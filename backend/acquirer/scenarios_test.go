package acquirer_test

import (
	"context"
	"reflect"
	"testing"

	"yuno-challenge/acquirer"
	"yuno-challenge/acquirer/acq1"
	"yuno-challenge/acquirer/acq2"
	"yuno-challenge/acquirer/acq3"
	"yuno-challenge/country"
	"yuno-challenge/testdata"
)

// TestAuthorizationRequests_Scenarios checks the generated test data meets the
// challenge's distribution requirements against the mock acquirer rules.
func TestAuthorizationRequests_Scenarios(t *testing.T) {
	primary := mustNew(t, acq1.New, testdata.AcquirerOneRules)
	secondary := mustNew(t, acq2.New, testdata.AcquirerTwoRules)
	// AcquirerThree's 80% success rate is random, so check the tertiary path
	// without it: requests reaching it must only be declined by chance.
	tertiary := mustNew(t, acq3.New, acquirer.Rules{})

	reqs := testdata.AuthorizationRequests(1)
	if len(reqs) != 50 {
		t.Fatalf("got %d requests, want 50", len(reqs))
	}

	var primaryApproved, primaryRetriable, primaryHard, secondaryApproved, reachedTertiary, secondaryHard int
	bins := map[string]bool{}
	countries := map[country.Code]bool{}
	for i, req := range reqs {
		if code, err := req.Validate(); err != nil {
			t.Fatalf("request %d fails validation (%s): %v", i, code, err)
		}
		bins[req.Card.Number[:6]] = true
		countries[req.Country] = true

		authReq := req.ToRequest()
		resp := primary.Authorize(context.Background(), authReq)
		switch {
		case resp.Approved:
			primaryApproved++
			continue
		case !resp.DeclineReason.Retriable():
			primaryHard++
			continue
		}
		primaryRetriable++

		resp = secondary.Authorize(context.Background(), authReq)
		switch {
		case resp.Approved:
			secondaryApproved++
			continue
		case !resp.DeclineReason.Retriable():
			secondaryHard++
			continue
		}
		reachedTertiary++

		if resp = tertiary.Authorize(context.Background(), authReq); !resp.Approved {
			t.Errorf("request %d reached tertiary but was declined with %s", i, resp.DeclineReason)
		}
	}

	counts := []int{primaryApproved, primaryRetriable, primaryHard, secondaryApproved, reachedTertiary, secondaryHard}
	if want := []int{26, 20, 4, 8, 9, 3}; !reflect.DeepEqual(counts, want) {
		t.Errorf("counts [primaryApproved primaryRetriable primaryHard secondaryApproved reachedTertiary secondaryHard] = %v, want %v", counts, want)
	}
	if len(bins) < 5 {
		t.Errorf("got %d distinct BINs, want at least 5", len(bins))
	}
	if len(countries) != 4 {
		t.Errorf("got %d distinct countries, want 4", len(countries))
	}
}

func TestAuthorizationRequests_Deterministic(t *testing.T) {
	if !reflect.DeepEqual(testdata.AuthorizationRequests(7), testdata.AuthorizationRequests(7)) {
		t.Fatal("same seed produced different requests")
	}
}

func mustNew[A acquirer.Acquirer](t *testing.T, newFn func(acquirer.Rules) (A, error), rules acquirer.Rules) A {
	t.Helper()
	a, err := newFn(rules)
	if err != nil {
		t.Fatalf("failed to create acquirer: %v", err)
	}
	return a
}
