package authorization

import (
	"context"
	"fmt"
	"net/http"

	"yuno-challenge/shared/sharedgin"

	"github.com/gin-gonic/gin"
)

// Processor authorizes a transaction across the acquirers; implemented by
// *processor.Processor.
type Processor interface {
	Process(ctx context.Context, req Request) Transaction
}

type Controller struct {
	processor Processor
	store     *Store
}

func NewController(processor Processor, store *Store) Controller {
	return Controller{processor: processor, store: store}
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
// @Failure 402 {object} AuthorizationResponse "Declined: a hard decline, or every acquirer tried declined"
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

	txn := ctrl.processor.Process(ctx.Request.Context(), req.ToRequest())
	status := http.StatusOK
	if !txn.Approved {
		status = http.StatusPaymentRequired
	}
	ctx.JSON(status, NewAuthorizationResponse(txn))
}

// ListAuthorizations godoc
// @Summary List processed authorizations
// @Description Full authorization log: every processed transaction with its attempt chain, oldest first.
// @Tags authorization
// @Produce json
// @Success 200 {object} ListAuthorizationsResponse
// @Router /authorizations [get]
func (ctrl Controller) ListAuthorizations(ctx *gin.Context) {
	txns := ctrl.store.List()
	resp := ListAuthorizationsResponse{Transactions: make([]TransactionResponse, len(txns))}
	for i, txn := range txns {
		resp.Transactions[i] = NewTransactionResponse(txn)
	}
	ctx.JSON(http.StatusOK, resp)
}

// GetAnalytics godoc
// @Summary Authorization success rate analytics
// @Description Overall and per-acquirer approval rates, average attempts and decline reasons over every processed transaction.
// @Tags authorization
// @Produce json
// @Success 200 {object} Summary
// @Router /analytics [get]
func (ctrl Controller) GetAnalytics(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, Summarize(ctrl.store.List()))
}
