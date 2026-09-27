// Package acq3 mocks AcquirerThree.
package acq3

import (
	"context"
	"time"

	"yuno-challenge/acquirer"
	"yuno-challenge/authorization"
)

type Acquirer struct {
	rules acquirer.Rules
}

var _ acquirer.Acquirer = (*Acquirer)(nil)

// New returns AcquirerThree, which approves or declines according to rules.
func New(rules acquirer.Rules) (*Acquirer, error) {
	if err := rules.Validate(); err != nil {
		return nil, err
	}
	return &Acquirer{rules: rules}, nil
}

func (a *Acquirer) Name() string {
	return "AcquirerThree"
}

func (a *Acquirer) Authorize(_ context.Context, req authorization.Request) acquirer.AuthorizationResponse {
	if reason, declined := a.rules.Check(req); declined {
		return acquirer.Declined(reason)
	}
	if reason, declined := acquirer.IssuerDecline(req.Card, time.Now()); declined {
		return acquirer.Declined(reason)
	}
	return acquirer.Approved()
}
