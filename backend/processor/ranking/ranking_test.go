package ranking

import (
	"slices"
	"sync"
	"testing"
	"time"
)

var names = []string{"AcquirerOne", "AcquirerTwo", "AcquirerThree"}

// newTestRanking returns a ranking whose clock only moves when advance is called.
func newTestRanking(t *testing.T) (r *Ranking, advance func(time.Duration)) {
	t.Helper()
	r, err := New(names)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	now := time.Date(2026, 9, 27, 12, 0, 0, 0, time.UTC)
	r.now = func() time.Time { return now }
	return r, func(d time.Duration) { now = now.Add(d) }
}

func record(t *testing.T, r *Ranking, name string, approved bool, times int) {
	t.Helper()
	for range times {
		if err := r.Record(name, approved); err != nil {
			t.Fatalf("Record(%q): %v", name, err)
		}
	}
}

func assertOrder(t *testing.T, r *Ranking, want ...string) {
	t.Helper()
	if got := r.Order(); !slices.Equal(got, want) {
		t.Errorf("Order() = %v, want %v", got, want)
	}
}

func TestNew_Validation(t *testing.T) {
	tests := []struct {
		name  string
		names []string
	}{
		{"no acquirers", nil},
		{"blank name", []string{"AcquirerOne", " "}},
		{"duplicate name", []string{"AcquirerOne", "AcquirerTwo", "AcquirerOne"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := New(tt.names); err == nil {
				t.Error("New() succeeded, want error")
			}
		})
	}
}

func TestOrder_StartsInConfiguredOrder(t *testing.T) {
	r, _ := newTestRanking(t)
	assertOrder(t, r, names...)

	record(t, r, "AcquirerThree", true, 5)
	assertOrder(t, r, names...)
}

func TestOrder_DeclinesDemote(t *testing.T) {
	r, _ := newTestRanking(t)

	record(t, r, "AcquirerOne", false, 1)
	assertOrder(t, r, "AcquirerTwo", "AcquirerThree", "AcquirerOne")

	record(t, r, "AcquirerTwo", false, 3)
	assertOrder(t, r, "AcquirerThree", "AcquirerOne", "AcquirerTwo")
}

func TestOrder_ApprovalsPromote(t *testing.T) {
	r, _ := newTestRanking(t)
	record(t, r, "AcquirerOne", false, 3)
	record(t, r, "AcquirerTwo", false, 1)
	assertOrder(t, r, "AcquirerThree", "AcquirerTwo", "AcquirerOne")

	record(t, r, "AcquirerOne", true, 10)
	assertOrder(t, r, "AcquirerThree", "AcquirerOne", "AcquirerTwo")
}

func TestOrder_RecoversOverTime(t *testing.T) {
	r, advance := newTestRanking(t)
	record(t, r, "AcquirerOne", false, 5)
	assertOrder(t, r, "AcquirerTwo", "AcquirerThree", "AcquirerOne")

	before := r.Standings()[2].Score
	advance(recoveryHalfLife)
	after := r.Standings()[2].Score
	if want := 1 - (1-before)/2; after < want-1e-9 || after > want+1e-9 {
		t.Errorf("score after one half-life = %v, want %v", after, want)
	}

	advance(12 * recoveryHalfLife)
	assertOrder(t, r, names...)
}

func TestRecord_UnknownAcquirer(t *testing.T) {
	r, _ := newTestRanking(t)
	if err := r.Record("AcquirerFour", true); err == nil {
		t.Error("Record() succeeded, want error")
	}
}

func TestRanking_ConcurrentUse(t *testing.T) {
	r, err := New(names)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	var wg sync.WaitGroup
	for i := range 50 {
		wg.Go(func() {
			for j := range 100 {
				name := names[(i+j)%len(names)]
				if err := r.Record(name, j%3 != 0); err != nil {
					t.Errorf("Record(%q): %v", name, err)
				}
				if got := len(r.Order()); got != len(names) {
					t.Errorf("len(Order()) = %d, want %d", got, len(names))
				}
			}
		})
	}
	wg.Wait()
}
