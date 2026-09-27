package merchant

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"yuno-challenge/sharedgin"

	"github.com/gin-gonic/gin"
)

const validBody = `{
	"merchantId": "solarbazaar",
	"amount": "14500.00",
	"currency": "MXN",
	"country": "MX",
	"card": {"number": "4532015112830366", "holderName": "Maria Lopez", "expiry": "202812", "cvv": "123"}
}`

func TestCreateAuthorization_Validation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.POST("/authorizations", NewController().CreateAuthorization)

	tests := []struct {
		name       string
		body       string
		wantStatus int
		wantCode   string
	}{
		{"valid request", validBody, http.StatusNotImplemented, "not-implemented"},
		{"malformed json", `{"merchantId":`, http.StatusBadRequest, ErrorCodeValidationError},
		{"numeric amount", strings.Replace(validBody, `"14500.00"`, `14500.5`, 1), http.StatusNotImplemented, "not-implemented"},
		{"non-numeric amount", strings.Replace(validBody, `"14500.00"`, `"abc"`, 1), http.StatusBadRequest, ErrorCodeValidationError},
		{"missing amount", strings.Replace(validBody, `"amount": "14500.00",`, ``, 1), http.StatusBadRequest, ErrorCodeValidationError},
		{"missing merchant", strings.Replace(validBody, `"solarbazaar"`, `" "`, 1), http.StatusBadRequest, ErrorCodeValidationError},
		{"zero amount", strings.Replace(validBody, `"14500.00"`, `"0.00"`, 1), http.StatusBadRequest, ErrorCodeValidationError},
		{"negative amount", strings.Replace(validBody, `"14500.00"`, `"-5"`, 1), http.StatusBadRequest, ErrorCodeValidationError},
		{"too many decimals", strings.Replace(validBody, `"14500.00"`, `"14500.001"`, 1), http.StatusBadRequest, ErrorCodeValidationError},
		{"clp whole amount", strings.NewReplacer(`"MXN"`, `"CLP"`, `"MX"`, `"CL"`, `"14500.00"`, `"850000"`).Replace(validBody), http.StatusNotImplemented, "not-implemented"},
		{"clp fractional amount", strings.NewReplacer(`"MXN"`, `"CLP"`, `"MX"`, `"CL"`, `"14500.00"`, `"850000.50"`).Replace(validBody), http.StatusBadRequest, ErrorCodeValidationError},
		{"unsupported currency", strings.Replace(validBody, `"MXN"`, `"USD"`, 1), http.StatusBadRequest, ErrorCodeValidationError},
		{"unsupported country", strings.Replace(validBody, `"MX"`, `"US"`, 1), http.StatusBadRequest, ErrorCodeValidationError},
		{"currency country mismatch", strings.Replace(validBody, `"MXN"`, `"BRL"`, 1), http.StatusBadRequest, ErrorCodeValidationError},
		{"short card number", strings.Replace(validBody, `4532015112830366`, `45320151`, 1), http.StatusBadRequest, ErrorCodeInvalidCard},
		{"non-digit card number", strings.Replace(validBody, `4532015112830366`, `4532-0151-1283-0366`, 1), http.StatusBadRequest, ErrorCodeInvalidCard},
		{"missing holder name", strings.Replace(validBody, `"Maria Lopez"`, `""`, 1), http.StatusBadRequest, ErrorCodeInvalidCard},
		{"bad cvv", strings.Replace(validBody, `"123"`, `"12a"`, 1), http.StatusBadRequest, ErrorCodeInvalidCard},
		{"bad expiry format", strings.Replace(validBody, `"202812"`, `"12/28"`, 1), http.StatusBadRequest, ErrorCodeInvalidExpiry},
		{"bad expiry month", strings.Replace(validBody, `"202812"`, `"202813"`, 1), http.StatusBadRequest, ErrorCodeInvalidExpiry},
		// Expired cards must reach the acquirers so they can decline with EXPIRED_CARD.
		{"past expiry passes validation", strings.Replace(validBody, `"202812"`, `"202001"`, 1), http.StatusNotImplemented, "not-implemented"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/authorizations", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			engine.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d, body: %s", rec.Code, tt.wantStatus, rec.Body.String())
			}
			var resp sharedgin.ErrorResponse
			if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
				t.Fatalf("failed to decode response: %v", err)
			}
			if resp.Code != tt.wantCode {
				t.Fatalf("code = %q, want %q, message: %s", resp.Code, tt.wantCode, resp.Message)
			}
			if strings.Contains(resp.Message, "4532015112830366") {
				t.Fatalf("response leaks card number: %s", resp.Message)
			}
		})
	}
}
