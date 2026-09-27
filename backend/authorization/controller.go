package authorization

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type Controller struct {
	store *Store
}

func NewController(store *Store) Controller {
	return Controller{store: store}
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
