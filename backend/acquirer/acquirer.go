// Package acquirer holds the Acquirer interface every acquirer integration
// implements; see the mock subpackage for the simulated acquirers. The failover
// engine that routes requests across acquirers lives in the processor package.
package acquirer

import (
	"context"

	"yuno-challenge/authorization"
)

// Acquirer is implemented by every acquirer; see the mock subpackage.
type Acquirer interface {
	// Name identifies the acquirer in routing config and attempt logs.
	Name() string
	Authorize(ctx context.Context, req authorization.Request) AuthorizationResponse
}

type AuthorizationResponse struct {
	Approved bool
	// DeclineReason is empty when Approved is true.
	DeclineReason authorization.DeclineReason
}

func Approved() AuthorizationResponse {
	return AuthorizationResponse{Approved: true}
}

func Declined(reason authorization.DeclineReason) AuthorizationResponse {
	return AuthorizationResponse{DeclineReason: reason}
}
