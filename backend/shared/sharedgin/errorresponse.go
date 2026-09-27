// Package sharedgin holds gin helpers shared across controllers, mirroring
// core/packages/shared/go/sharedgin.
package sharedgin

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

const ErrorCodeValidation = "validation-error"

// ErrorResponse is the body of every non-2xx response.
type ErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message,omitempty"`
} // @name ErrorResponse

// ValidationError will write a 400 response with pre-defined structure serializing
// provided err into the message field. Error code will be set to "validation-error".
func ValidationError(c *gin.Context, err error) {
	ValidationErrorWithCode(c, err, ErrorCodeValidation)
}

// ValidationErrorWithCode will write a 400 response with pre-defined structure serializing
// provided err into the message field. Error code will be set to the provided code.
func ValidationErrorWithCode(c *gin.Context, err error, code string) {
	c.AbortWithStatusJSON(http.StatusBadRequest, ErrorResponse{
		Code:    code,
		Message: err.Error(),
	})
}
