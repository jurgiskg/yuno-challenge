package authorization

import (
	"cmp"
	"maps"
	"slices"
)

// Summary is the success rate analytics over a set of processed transactions.
type Summary struct {
	Transactions int `json:"transactions" example:"50"`
	Approved     int `json:"approved" example:"41"`
	// Share of transactions ultimately approved after all retries, in [0, 1].
	ApprovalRate float64 `json:"approvalRate" example:"0.82"`
	// Approved transactions that the first acquirer tried had declined.
	ApprovedAfterFailover int     `json:"approvedAfterFailover" example:"15"`
	AverageAttempts       float64 `json:"averageAttempts" example:"1.62"`
	// Final decline reasons of declined transactions, most common first.
	DeclineReasons []ReasonCount `json:"declineReasons"`
	// Per-acquirer performance, in name order.
	Acquirers []AcquirerSummary `json:"acquirers"`
} // @name Summary

type AcquirerSummary struct {
	Acquirer string `json:"acquirer" example:"AcquirerOne"`
	// Times the acquirer was tried.
	Attempts int `json:"attempts" example:"50"`
	Approved int `json:"approved" example:"26"`
	// Share of attempts the acquirer approved, in [0, 1].
	ApprovalRate float64 `json:"approvalRate" example:"0.52"`
	// The acquirer's decline reasons, most common first.
	DeclineReasons []ReasonCount `json:"declineReasons"`
} // @name AcquirerSummary

type ReasonCount struct {
	Reason DeclineReason `json:"reason" swaggertype:"string" example:"POLICY_DECLINE"`
	Count  int           `json:"count" example:"12"`
} // @name ReasonCount

// Summarize computes the success rate analytics for txns.
func Summarize(txns []Transaction) Summary {
	s := Summary{Transactions: len(txns)}
	finalReasons := map[DeclineReason]int{}
	type tally struct {
		attempts, approved int
		reasons            map[DeclineReason]int
	}
	acquirers := map[string]*tally{}

	attempts := 0
	for _, txn := range txns {
		attempts += len(txn.Attempts)
		switch {
		case txn.Approved:
			s.Approved++
			if len(txn.Attempts) > 1 {
				s.ApprovedAfterFailover++
			}
		case txn.DeclineReason != "":
			finalReasons[txn.DeclineReason]++
		}
		for _, a := range txn.Attempts {
			t := acquirers[a.Acquirer]
			if t == nil {
				t = &tally{reasons: map[DeclineReason]int{}}
				acquirers[a.Acquirer] = t
			}
			t.attempts++
			if a.Approved {
				t.approved++
			} else {
				t.reasons[a.DeclineReason]++
			}
		}
	}

	s.ApprovalRate = ratio(s.Approved, s.Transactions)
	s.AverageAttempts = ratio(attempts, s.Transactions)
	s.DeclineReasons = reasonCounts(finalReasons)
	s.Acquirers = make([]AcquirerSummary, 0, len(acquirers))
	for _, name := range slices.Sorted(maps.Keys(acquirers)) {
		t := acquirers[name]
		s.Acquirers = append(s.Acquirers, AcquirerSummary{
			Acquirer:       name,
			Attempts:       t.attempts,
			Approved:       t.approved,
			ApprovalRate:   ratio(t.approved, t.attempts),
			DeclineReasons: reasonCounts(t.reasons),
		})
	}
	return s
}

// reasonCounts returns counts most common first, ties broken by reason.
func reasonCounts(counts map[DeclineReason]int) []ReasonCount {
	out := make([]ReasonCount, 0, len(counts))
	for reason, n := range counts {
		out = append(out, ReasonCount{Reason: reason, Count: n})
	}
	slices.SortFunc(out, func(a, b ReasonCount) int {
		return cmp.Or(cmp.Compare(b.Count, a.Count), cmp.Compare(a.Reason, b.Reason))
	})
	return out
}

func ratio(n, total int) float64 {
	if total == 0 {
		return 0
	}
	return float64(n) / float64(total)
}
