// Package merchant holds merchant-level configuration such as acquirer routing order.
package merchant

import (
	"fmt"
	"net/http"

	"yuno-challenge/sharedgin"

	"github.com/gin-gonic/gin"
)

type Controller struct{}

func NewController() Controller {
	return Controller{}
}

// CreateAuthorization godoc
// @Summary Authorize a transaction
// @Description Authorize a card transaction, failing over across acquirers on retriable declines.
// @Tags authorization
// @Accept json
// @Produce json
// @Param request body CreateAuthorizationRequest true "Authorization request"
// @Failure 400 {object} sharedgin.ErrorResponse
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

	// TODO: hand the request to the acquirer failover engine and return its result.
	ctx.JSON(http.StatusNotImplemented, sharedgin.ErrorResponse{
		Code:    "not-implemented",
		Message: "acquirer failover is not wired up yet",
	})
}
