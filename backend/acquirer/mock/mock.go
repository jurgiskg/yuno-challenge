// Package mock provides a mock acquirer that approves or declines by
// configurable rules, standing in for a real acquirer integration.
package mock

import (
	"context"
	"errors"
	"strings"
	"time"

	"yuno-challenge/acquirer"
	"yuno-challenge/authorization"
)

type Acquirer struct {
	name  string
	rules acquirer.Rules
}

var _ acquirer.Acquirer = (*Acquirer)(nil)

// New returns a mock acquirer called name, which approves or declines according
// to rules.
func New(name string, rules acquirer.Rules) (*Acquirer, error) {
	if strings.TrimSpace(name) == "" {
		return nil, errors.New("acquirer name must not be blank")
	}
	if err := rules.Validate(); err != nil {
		return nil, err
	}
	return &Acquirer{name: name, rules: rules}, nil
}

func (a *Acquirer) Name() string {
	return a.name
}

// Authorize applies the acquirer's rules first, then the simulated issuer checks.
func (a *Acquirer) Authorize(_ context.Context, req authorization.Request) acquirer.AuthorizationResponse {
	if reason, declined := a.rules.Check(req); declined {
		return acquirer.Declined(reason)
	}
	if reason, declined := acquirer.IssuerDecline(req.Card, time.Now()); declined {
		return acquirer.Declined(reason)
	}
	return acquirer.Approved()
}
