//go:build integration

package integrationtests

import (
	"fmt"
	"io"
	"maps"
	"slices"
	"strings"
	"text/tabwriter"

	"yuno-challenge/authorization"
)

// analysis is the test's own reading of an authorization log, independent of
// the server's analytics endpoint.
type analysis struct {
	total    int
	approved int
	// approvedFirst were approved by the first acquirer tried; rescued by a later one.
	approvedFirst int
	rescued       int
	// hardDeclined stopped at a non-retriable decline; exhausted were declined
	// with a retriable reason by every acquirer tried.
	hardDeclined int
	exhausted    int
	attempts     int
	acquirers    map[string]*acquirerStats
	// finalReasons counts declined transactions by their final decline reason.
	finalReasons map[authorization.DeclineReason]int
	// orderChanges lists each point the routing order changed, starting with the first.
	orderChanges []orderChange
}

type acquirerStats struct {
	tried, approved int
	reasons         map[authorization.DeclineReason]int
}

type orderChange struct {
	// transaction is the 1-based position of the first transaction routed this way.
	transaction int
	order       []string
}

func analyze(log []authorization.TransactionResponse) analysis {
	a := analysis{
		total:        len(log),
		acquirers:    map[string]*acquirerStats{},
		finalReasons: map[authorization.DeclineReason]int{},
	}
	for i, txn := range log {
		a.attempts += len(txn.Attempts)
		switch {
		case txn.Status == authorization.StatusApproved && len(txn.Attempts) == 1:
			a.approved++
			a.approvedFirst++
		case txn.Status == authorization.StatusApproved:
			a.approved++
			a.rescued++
		case txn.DeclineReason.Retriable():
			a.exhausted++
			a.finalReasons[txn.DeclineReason]++
		default:
			a.hardDeclined++
			a.finalReasons[txn.DeclineReason]++
		}

		for _, at := range txn.Attempts {
			st := a.acquirers[at.Acquirer]
			if st == nil {
				st = &acquirerStats{reasons: map[authorization.DeclineReason]int{}}
				a.acquirers[at.Acquirer] = st
			}
			st.tried++
			if at.Status == authorization.StatusApproved {
				st.approved++
			} else {
				st.reasons[at.DeclineReason]++
			}
		}

		if n := len(a.orderChanges); n == 0 || !slices.Equal(a.orderChanges[n-1].order, txn.RoutingOrder) {
			a.orderChanges = append(a.orderChanges, orderChange{transaction: i + 1, order: txn.RoutingOrder})
		}
	}
	return a
}

func (a analysis) approvalRate() float64 { return rate(a.approved, a.total) }

func printReport(w io.Writer, submitted int, results []result) {
	if len(results) == 0 {
		return
	}
	fmt.Fprintf(w, "\n=== Acceptance report: %d sample transactions (testdata seed %d) per scenario ===\n\n", submitted, seed)

	tw := tabwriter.NewWriter(w, 0, 0, 3, ' ', 0)
	row := func(label string, value func(r result) string) {
		cells := []string{label}
		for _, r := range results {
			cells = append(cells, value(r))
		}
		fmt.Fprintln(tw, strings.Join(cells, "\t"))
	}
	row("", func(r result) string { return r.scenario.description })
	row("Acquirers (configured order)", func(r result) string { return r.scenario.acquirerOrder })
	row("Approved", func(r result) string {
		return fmt.Sprintf("%d/%d (%s)", r.analysis.approved, r.analysis.total, pct(r.analysis.approvalRate()))
	})
	row("  on first attempt", func(r result) string { return fmt.Sprint(r.analysis.approvedFirst) })
	row("  rescued by failover", func(r result) string { return fmt.Sprint(r.analysis.rescued) })
	row("Declined", func(r result) string { return fmt.Sprint(r.analysis.total - r.analysis.approved) })
	row("  hard decline (not retried)", func(r result) string { return fmt.Sprint(r.analysis.hardDeclined) })
	row("  every acquirer declined", func(r result) string { return fmt.Sprint(r.analysis.exhausted) })
	row("Avg attempts per transaction", func(r result) string {
		return fmt.Sprintf("%.2f", float64(r.analysis.attempts)/float64(max(r.analysis.total, 1)))
	})
	row("Routing order changes", func(r result) string { return fmt.Sprint(len(r.analysis.orderChanges) - 1) })
	tw.Flush()

	baseline := results[0]
	for _, r := range results[1:] {
		diff := r.analysis.approvalRate() - baseline.analysis.approvalRate()
		fmt.Fprintf(w, "\n%s: %s approved vs %s with %s (%+.1f pp, %+d transactions).\n",
			r.scenario.description, pct(r.analysis.approvalRate()), pct(baseline.analysis.approvalRate()),
			strings.ToLower(baseline.scenario.description), diff*100, r.analysis.approved-baseline.analysis.approved)
	}

	for _, r := range results {
		fmt.Fprintf(w, "\n--- %s: per-acquirer performance ---\n", r.scenario.description)
		tw := tabwriter.NewWriter(w, 0, 0, 3, ' ', 0)
		fmt.Fprintln(tw, "Acquirer\tTried\tApproved\tRate\tDecline reasons")
		for _, name := range slices.Sorted(maps.Keys(r.analysis.acquirers)) {
			st := r.analysis.acquirers[name]
			fmt.Fprintf(tw, "%s\t%d\t%d\t%s\t%s\n", name, st.tried, st.approved, pct(rate(st.approved, st.tried)), formatReasons(st.reasons))
		}
		tw.Flush()
		fmt.Fprintf(w, "Final decline reasons: %s\n", formatReasons(r.analysis.finalReasons))

		if len(r.analysis.orderChanges) > 1 {
			fmt.Fprintf(w, "Routing order (dynamic ranking), by first transaction routed that way:\n")
			for _, c := range r.analysis.orderChanges {
				fmt.Fprintf(w, "  #%-3d %s\n", c.transaction, strings.Join(c.order, " > "))
			}
		}
	}
	fmt.Fprintln(w)
}

// formatReasons renders counts most common first, e.g. "POLICY_DECLINE 12, STOLEN_CARD 2".
func formatReasons(counts map[authorization.DeclineReason]int) string {
	if len(counts) == 0 {
		return "-"
	}
	reasons := slices.Collect(maps.Keys(counts))
	slices.SortFunc(reasons, func(a, b authorization.DeclineReason) int {
		if counts[a] != counts[b] {
			return counts[b] - counts[a]
		}
		return strings.Compare(string(a), string(b))
	})
	parts := make([]string, len(reasons))
	for i, reason := range reasons {
		parts[i] = fmt.Sprintf("%s %d", reason, counts[reason])
	}
	return strings.Join(parts, ", ")
}

func rate(n, total int) float64 {
	if total == 0 {
		return 0
	}
	return float64(n) / float64(total)
}

func pct(r float64) string {
	return fmt.Sprintf("%.1f%%", r*100)
}
