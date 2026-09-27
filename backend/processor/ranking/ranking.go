// Package ranking orders acquirers by their recent approval performance, so the
// failover engine tries the acquirer most likely to approve first.
//
// Each acquirer carries a score in [0, 1]: an exponentially weighted moving
// average of its recent outcomes (1 = approved, 0 = declined). Every score starts
// at 1, so until outcomes are recorded the configured order is used as-is. Scores
// decay back towards 1 over time, which lets a demoted acquirer recover once it
// stops getting traffic instead of staying last forever.
//
// It works on acquirer names only, so it doesn't depend on the acquirer package.
package ranking

import (
	"errors"
	"fmt"
	"math"
	"slices"
	"strings"
	"sync"
	"time"
)

const (
	// smoothing is the weight of the newest outcome in the moving average; higher
	// reacts faster to changes but is noisier.
	smoothing = 0.2
	// recoveryHalfLife is how long it takes a score to recover half its distance to 1.
	recoveryHalfLife = 15 * time.Minute
	// scorePrecision is the number of steps scores are rounded to before comparing,
	// so near-equal scores keep the configured order instead of flip-flopping.
	scorePrecision = 100
)

// Standing is an acquirer's position in the ranking.
type Standing struct {
	Name string
	// Score is the acquirer's recent approval rate in [0, 1].
	Score float64
}

type entry struct {
	name string
	// score is the moving average as of updatedAt; use scoreAt to read it.
	score     float64
	updatedAt time.Time
}

// Ranking holds the acquirer order in memory. It is safe for concurrent use.
type Ranking struct {
	mu sync.Mutex
	// entries is in configured order, which breaks ties between equal scores.
	entries []entry
	now     func() time.Time
}

// New returns a ranking of the named acquirers, initially in the given order.
func New(names []string) (*Ranking, error) {
	if len(names) == 0 {
		return nil, errors.New("at least one acquirer is required")
	}
	entries := make([]entry, 0, len(names))
	for _, name := range names {
		if strings.TrimSpace(name) == "" {
			return nil, errors.New("acquirer name must not be blank")
		}
		if slices.ContainsFunc(entries, func(e entry) bool { return e.name == name }) {
			return nil, fmt.Errorf("duplicate acquirer %q", name)
		}
		entries = append(entries, entry{name: name, score: 1})
	}
	return &Ranking{entries: entries, now: time.Now}, nil
}

// Record adjusts the named acquirer's score after it handled an authorization
// attempt. Only record outcomes the acquirer is responsible for: approvals and
// retriable declines. Issuer declines (stolen card, insufficient funds, ...)
// would happen on any acquirer and shouldn't count against it.
func (r *Ranking) Record(name string, approved bool) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	i := slices.IndexFunc(r.entries, func(e entry) bool { return e.name == name })
	if i < 0 {
		return fmt.Errorf("unknown acquirer %q", name)
	}

	now := r.now()
	outcome := 0.0
	if approved {
		outcome = 1
	}
	e := &r.entries[i]
	e.score = (1-smoothing)*e.scoreAt(now) + smoothing*outcome
	e.updatedAt = now
	return nil
}

// Order returns the acquirer names, best first.
func (r *Ranking) Order() []string {
	standings := r.Standings()
	names := make([]string, len(standings))
	for i, s := range standings {
		names[i] = s.Name
	}
	return names
}

// Standings returns every acquirer with its current score, best first.
func (r *Ranking) Standings() []Standing {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := r.now()
	standings := make([]Standing, len(r.entries))
	for i, e := range r.entries {
		standings[i] = Standing{Name: e.name, Score: e.scoreAt(now)}
	}
	// Stable, so acquirers with equal rounded scores keep their configured order.
	slices.SortStableFunc(standings, func(a, b Standing) int {
		return int(rounded(b.Score) - rounded(a.Score))
	})
	return standings
}

// scoreAt returns the score decayed towards 1 for the time since the last outcome.
func (e entry) scoreAt(now time.Time) float64 {
	if e.updatedAt.IsZero() {
		return e.score
	}
	elapsed := now.Sub(e.updatedAt)
	if elapsed <= 0 {
		return e.score
	}
	remaining := math.Pow(0.5, float64(elapsed)/float64(recoveryHalfLife))
	return 1 - (1-e.score)*remaining
}

func rounded(score float64) int64 {
	return int64(math.Round(score * scorePrecision))
}
