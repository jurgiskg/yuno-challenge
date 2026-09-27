// Package acquirer holds the mock acquirers and the failover logic that routes
// authorization requests across them.
package acquirer

type Controller struct{}

func NewController() Controller {
	return Controller{}
}
