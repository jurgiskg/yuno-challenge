package authorization

import (
	"reflect"
	"testing"
)

func TestSummarize(t *testing.T) {
	approved := func(acquirer string) Attempt { return Attempt{Acquirer: acquirer, Approved: true} }
	declined := func(acquirer string, reason DeclineReason) Attempt {
		return Attempt{Acquirer: acquirer, DeclineReason: reason}
	}
	txns := []Transaction{
		{Approved: true, Acquirer: "One", Attempts: []Attempt{approved("One")}},
		{Approved: true, Acquirer: "Two", Attempts: []Attempt{declined("One", ReasonPolicyDecline), approved("Two")}},
		{DeclineReason: ReasonStolenCard, Attempts: []Attempt{declined("One", ReasonStolenCard)}},
		{DeclineReason: ReasonGenericDecline, Attempts: []Attempt{
			declined("One", ReasonPolicyDecline), declined("Two", ReasonSuspectedFraud), declined("Three", ReasonGenericDecline),
		}},
	}

	got := Summarize(txns)

	want := Summary{
		Transactions:          4,
		Approved:              2,
		ApprovalRate:          0.5,
		ApprovedAfterFailover: 1,
		AverageAttempts:       7.0 / 4,
		DeclineReasons:        []ReasonCount{{ReasonGenericDecline, 1}, {ReasonStolenCard, 1}},
		Acquirers: []AcquirerSummary{
			{Acquirer: "One", Attempts: 4, Approved: 1, ApprovalRate: 0.25, DeclineReasons: []ReasonCount{{ReasonPolicyDecline, 2}, {ReasonStolenCard, 1}}},
			{Acquirer: "Three", Attempts: 1, ApprovalRate: 0, DeclineReasons: []ReasonCount{{ReasonGenericDecline, 1}}},
			{Acquirer: "Two", Attempts: 2, Approved: 1, ApprovalRate: 0.5, DeclineReasons: []ReasonCount{{ReasonSuspectedFraud, 1}}},
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Summarize() =\n%+v\nwant\n%+v", got, want)
	}
}

func TestSummarize_Empty(t *testing.T) {
	got := Summarize(nil)
	if got.Transactions != 0 || got.ApprovalRate != 0 || got.AverageAttempts != 0 || len(got.Acquirers) != 0 {
		t.Fatalf("Summarize(nil) = %+v, want zero values", got)
	}
}
