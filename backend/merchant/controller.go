// Package merchant holds merchant-level configuration such as acquirer routing order.
package merchant

import (
	"context"
	"fmt"
	"net/http"

	"yuno-challenge/acquirer"
	"yuno-challenge/authorization"
	"yuno-challenge/sharedgin"

	"github.com/gin-gonic/gin"
)

// Processor authorizes a transaction across the acquirers; implemented by
// *acquirer.Processor.
type Processor interface {
	Process(ctx context.Context, req acquirer.AuthorizationRequest) authorization.Transaction
}

type Controller struct {
	processor Processor
}

func NewController(processor Processor) Controller {
	return Controller{processor: processor}
}

// CreateAuthorization godoc
// @Summary Authorize a transaction
// @Description Authorize a card transaction, failing over across acquirers on retriable declines.
// @Tags authorization
// @Accept json
// @Produce json
// @Param request body CreateAuthorizationRequest true "Authorization request"
// @Success 200 {object} AuthorizationResponse "Approved"
// @Failure 400 {object} sharedgin.ErrorResponse
// @Failure 500 {object} AuthorizationResponse "Declined by every acquirer tried"
// @Router /authorizations [post]
func (ctrl Controller) CreateAuthorization(ctx *gin.Context) {
	var req CreateAuthorizationRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		sharedgin.ValidationError(ctx, fmt.Errorf("failed to bind request body: %w", err))
		return
	}

	if code, err := req.Validate(); err != nil {
		sharedgin.ValidationErrorWithCode(ctx, err, code)
		return
	}

	txn := ctrl.processor.Process(ctx.Request.Context(), req.ToAuthorizationRequest())
	status := http.StatusOK
	if !txn.Approved {
		status = http.StatusInternalServerError
	}
	ctx.JSON(status, NewAuthorizationResponse(txn))
}
