// Package acquirer holds the Acquirer interface, the mock acquirers' rules and
// the simulated issuer checks. The failover engine that routes requests across
// acquirers lives in the processor package.
package acquirer

type Controller struct{}

func NewController() Controller {
	return Controller{}
}
