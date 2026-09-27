// Package authorization holds the outcome of authorizing a transaction across
// acquirers: decline reasons, the attempt chain, and the store that keeps it
// for analytics.
//
// It deliberately doesn't import acquirer, so acquirers and the failover engine
// there can depend on it without an import cycle.
package authorization

type DeclineReason string

const (
	// Retriable: another acquirer may approve the same transaction.
	ReasonSuspectedFraud DeclineReason = "SUSPECTED_FRAUD" // acquirer's fraud filter, not the issuer's
	ReasonPolicyDecline  DeclineReason = "POLICY_DECLINE"  // acquirer's risk policy
	ReasonTimeout        DeclineReason = "TIMEOUT"
	ReasonGenericDecline DeclineReason = "GENERIC_DECLINE"

	// Non-retriable: the card itself is the problem, so every acquirer will decline.
	ReasonInsufficientFunds DeclineReason = "INSUFFICIENT_FUNDS"
	ReasonStolenCard        DeclineReason = "STOLEN_CARD"
	ReasonInvalidCard       DeclineReason = "INVALID_CARD"
	ReasonExpiredCard       DeclineReason = "EXPIRED_CARD"
)

// Retriable reports whether the transaction should fail over to the next acquirer.
func (r DeclineReason) Retriable() bool {
	switch r {
	case ReasonSuspectedFraud, ReasonPolicyDecline, ReasonTimeout, ReasonGenericDecline:
		return true
	}
	return false
}
