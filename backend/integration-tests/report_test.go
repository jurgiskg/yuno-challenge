//go:build integration

package integrationtests

import (
	"bytes"
	_ "embed"
	"fmt"
	"html/template"
	"maps"
	"os"
	"slices"
	"strings"
	"time"

	"yuno-challenge/authorization"
	"yuno-challenge/shared/currency"

	"github.com/shopspring/decimal"
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

//go:embed report.html.tmpl
var reportTemplate string

var reportTmpl = template.Must(template.New("report").Parse(reportTemplate))

type reportView struct {
	GeneratedAt string
	Seed        int
	Submitted   int
	// Uplift compares the last scenario with the baseline; nil with one scenario.
	Uplift    *upliftView
	Compare   []compareRow
	Scenarios []scenarioView
}

type upliftView struct {
	Points   string
	Summary  string
	Positive bool
}

type compareRow struct {
	Label  string
	Indent bool
	Values []string
}

type scenarioView struct {
	Name          string
	Description   string
	AcquirerOrder string
	Approved      int
	Total         int
	ApprovalRate  string
	// Delta is the approval rate change vs the baseline, empty for the baseline.
	Delta         string
	DeltaPositive bool
	Acquirers     []acquirerRow
	FinalReasons  string
	OrderChanges  []orderChangeView
	Transactions  []transactionRow
}

type acquirerRow struct {
	Name, Rate, Reasons string
	Tried, Approved     int
}

type orderChangeView struct {
	Transaction int
	Order       string
}

type transactionRow struct {
	N            int
	Country, BIN string
	Amount       string
	Routing      string
	Attempts     []attemptView
	Approved     bool
	Outcome      string
}

type attemptView struct {
	Acquirer, Outcome, Duration string
	Approved                    bool
}

// writeReport renders the HTML acceptance report to path, replacing any
// previous report.
func writeReport(path string, submitted int, results []result) error {
	view := reportView{
		GeneratedAt: time.Now().Format("2006-01-02 15:04:05 MST"),
		Seed:        seed,
		Submitted:   submitted,
		Compare:     compareRows(results),
	}
	baseline := results[0].analysis
	for i, r := range results {
		sv := newScenarioView(r)
		if i > 0 {
			diff := (r.analysis.approvalRate() - baseline.approvalRate()) * 100
			sv.Delta = fmt.Sprintf("%+.1f pp vs %s", diff, strings.ToLower(results[0].scenario.description))
			sv.DeltaPositive = diff > 0
		}
		view.Scenarios = append(view.Scenarios, sv)
	}
	if len(results) > 1 {
		last := results[len(results)-1]
		diff := (last.analysis.approvalRate() - baseline.approvalRate()) * 100
		view.Uplift = &upliftView{
			Points:   fmt.Sprintf("%+.1f pp", diff),
			Positive: diff > 0,
			Summary: fmt.Sprintf("%s approved %s of transactions vs %s with %s (%+d transactions).",
				last.scenario.description, pct(last.analysis.approvalRate()), pct(baseline.approvalRate()),
				strings.ToLower(results[0].scenario.description), last.analysis.approved-baseline.approved),
		}
	}

	var buf bytes.Buffer
	if err := reportTmpl.Execute(&buf, view); err != nil {
		return err
	}
	return os.WriteFile(path, buf.Bytes(), 0o644)
}

func compareRows(results []result) []compareRow {
	row := func(label string, indent bool, value func(a analysis) string) compareRow {
		r := compareRow{Label: label, Indent: indent}
		for _, res := range results {
			r.Values = append(r.Values, value(res.analysis))
		}
		return r
	}
	return []compareRow{
		row("Approved", false, func(a analysis) string {
			return fmt.Sprintf("%d/%d (%s)", a.approved, a.total, pct(a.approvalRate()))
		}),
		row("on first attempt", true, func(a analysis) string { return fmt.Sprint(a.approvedFirst) }),
		row("rescued by failover", true, func(a analysis) string { return fmt.Sprint(a.rescued) }),
		row("Declined", false, func(a analysis) string { return fmt.Sprint(a.total - a.approved) }),
		row("hard decline (not retried)", true, func(a analysis) string { return fmt.Sprint(a.hardDeclined) }),
		row("every acquirer declined", true, func(a analysis) string { return fmt.Sprint(a.exhausted) }),
		row("Avg attempts per transaction", false, func(a analysis) string {
			return fmt.Sprintf("%.2f", float64(a.attempts)/float64(max(a.total, 1)))
		}),
		row("Routing order changes", false, func(a analysis) string { return fmt.Sprint(max(len(a.orderChanges)-1, 0)) }),
	}
}

func newScenarioView(r result) scenarioView {
	a := r.analysis
	sv := scenarioView{
		Name:          r.scenario.name,
		Description:   r.scenario.description,
		AcquirerOrder: strings.ReplaceAll(r.scenario.acquirerOrder, ",", " > "),
		Approved:      a.approved,
		Total:         a.total,
		ApprovalRate:  pct(a.approvalRate()),
		FinalReasons:  formatReasons(a.finalReasons),
	}
	for _, name := range slices.Sorted(maps.Keys(a.acquirers)) {
		st := a.acquirers[name]
		sv.Acquirers = append(sv.Acquirers, acquirerRow{
			Name: name, Tried: st.tried, Approved: st.approved,
			Rate: pct(rate(st.approved, st.tried)), Reasons: formatReasons(st.reasons),
		})
	}
	if len(a.orderChanges) > 1 {
		for _, c := range a.orderChanges {
			sv.OrderChanges = append(sv.OrderChanges, orderChangeView{Transaction: c.transaction, Order: strings.Join(c.order, " > ")})
		}
	}
	for i, txn := range r.log {
		row := transactionRow{
			N:        i + 1,
			Country:  string(txn.Country),
			BIN:      txn.BIN,
			Amount:   formatAmount(txn.AmountMinor, txn.Currency),
			Routing:  strings.Join(txn.RoutingOrder, " > "),
			Approved: txn.Status == authorization.StatusApproved,
			Outcome:  "Approved by " + txn.Acquirer,
		}
		if !row.Approved {
			row.Outcome = "Declined: " + string(txn.DeclineReason)
		}
		for _, at := range txn.Attempts {
			outcome := "approved"
			if at.Status != authorization.StatusApproved {
				outcome = string(at.DeclineReason)
			}
			row.Attempts = append(row.Attempts, attemptView{
				Acquirer: at.Acquirer, Outcome: outcome, Approved: at.Status == authorization.StatusApproved,
				Duration: formatDuration(at.DurationMs),
			})
		}
		sv.Transactions = append(sv.Transactions, row)
	}
	return sv
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

// formatDuration renders sub-millisecond attempts (the mocks answer in
// microseconds) in µs, anything slower in ms.
func formatDuration(ms float64) string {
	if ms < 1 {
		return fmt.Sprintf("%.0f µs", ms*1000)
	}
	return fmt.Sprintf("%.1f ms", ms)
}

// formatAmount renders minor units in major units, e.g. 1450000 MXN as "14500.00 MXN".
func formatAmount(minor int64, cur currency.Code) string {
	decimals := cur.Decimals()
	return decimal.New(minor, -decimals).StringFixed(decimals) + " " + string(cur)
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
