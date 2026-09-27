package authorization

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"yuno-challenge/shared/country"
	"yuno-challenge/shared/sharedgin"

	"github.com/gin-gonic/gin"
)

const validBody = `{
	"merchantId": "solarbazaar",
	"amount": "14500.00",
	"currency": "MXN",
	"country": "MX",
	"card": {"number": "4532015112830366", "holderName": "Maria Lopez", "expiry": "202812", "cvv": "123"}
}`

// stubProcessor returns txn for every request and remembers the last request.
type stubProcessor struct {
	txn  Transaction
	last Request
}

func (s *stubProcessor) Process(_ context.Context, req Request) Transaction {
	s.last = req
	return s.txn
}

func newTestEngine(processor Processor) *gin.Engine {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.POST("/authorizations", NewController(processor, NewStore()).CreateAuthorization)
	return engine
}

func post(engine *gin.Engine, body string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/authorizations", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	engine.ServeHTTP(rec, req)
	return rec
}

func TestCreateAuthorization_Validation(t *testing.T) {
	engine := newTestEngine(&stubProcessor{txn: Transaction{Approved: true, Acquirer: "AcquirerOne"}})

	tests := []struct {
		name       string
		body       string
		wantStatus int
		wantCode   string
	}{
		{"valid request", validBody, http.StatusOK, ""},
		{"malformed json", `{"merchantId":`, http.StatusBadRequest, ErrorCodeValidationError},
		{"numeric amount", strings.Replace(validBody, `"14500.00"`, `14500.5`, 1), http.StatusOK, ""},
		{"non-numeric amount", strings.Replace(validBody, `"14500.00"`, `"abc"`, 1), http.StatusBadRequest, ErrorCodeValidationError},
		{"missing amount", strings.Replace(validBody, `"amount": "14500.00",`, ``, 1), http.StatusBadRequest, ErrorCodeValidationError},
		{"missing merchant", strings.Replace(validBody, `"solarbazaar"`, `" "`, 1), http.StatusBadRequest, ErrorCodeValidationError},
		{"zero amount", strings.Replace(validBody, `"14500.00"`, `"0.00"`, 1), http.StatusBadRequest, ErrorCodeValidationError},
		{"negative amount", strings.Replace(validBody, `"14500.00"`, `"-5"`, 1), http.StatusBadRequest, ErrorCodeValidationError},
		{"too many decimals", strings.Replace(validBody, `"14500.00"`, `"14500.001"`, 1), http.StatusBadRequest, ErrorCodeValidationError},
		{"clp whole amount", strings.NewReplacer(`"MXN"`, `"CLP"`, `"MX"`, `"CL"`, `"14500.00"`, `"850000"`).Replace(validBody), http.StatusOK, ""},
		{"clp fractional amount", strings.NewReplacer(`"MXN"`, `"CLP"`, `"MX"`, `"CL"`, `"14500.00"`, `"850000.50"`).Replace(validBody), http.StatusBadRequest, ErrorCodeValidationError},
		{"unsupported currency", strings.Replace(validBody, `"MXN"`, `"USD"`, 1), http.StatusBadRequest, ErrorCodeValidationError},
		{"unsupported country", strings.Replace(validBody, `"MX"`, `"US"`, 1), http.StatusBadRequest, ErrorCodeValidationError},
		{"non-iso country", strings.Replace(validBody, `"MX"`, `"XX"`, 1), http.StatusBadRequest, ErrorCodeValidationError},
		{"lowercase country", strings.Replace(validBody, `"MX"`, `"mx"`, 1), http.StatusBadRequest, ErrorCodeValidationError},
		{"currency country mismatch", strings.Replace(validBody, `"MXN"`, `"BRL"`, 1), http.StatusBadRequest, ErrorCodeValidationError},
		{"short card number", strings.Replace(validBody, `4532015112830366`, `45320151`, 1), http.StatusBadRequest, ErrorCodeInvalidCard},
		{"non-digit card number", strings.Replace(validBody, `4532015112830366`, `4532-0151-1283-0366`, 1), http.StatusBadRequest, ErrorCodeInvalidCard},
		{"missing holder name", strings.Replace(validBody, `"Maria Lopez"`, `""`, 1), http.StatusBadRequest, ErrorCodeInvalidCard},
		{"bad cvv", strings.Replace(validBody, `"123"`, `"12a"`, 1), http.StatusBadRequest, ErrorCodeInvalidCard},
		{"bad expiry format", strings.Replace(validBody, `"202812"`, `"12/28"`, 1), http.StatusBadRequest, ErrorCodeInvalidExpiry},
		{"bad expiry month", strings.Replace(validBody, `"202812"`, `"202813"`, 1), http.StatusBadRequest, ErrorCodeInvalidExpiry},
		// Expired cards must reach the acquirers so they can decline with EXPIRED_CARD.
		{"past expiry passes validation", strings.Replace(validBody, `"202812"`, `"202001"`, 1), http.StatusOK, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := post(engine, tt.body)

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

func TestCreateAuthorization_Outcome(t *testing.T) {
	started := time.Date(2026, 9, 27, 12, 0, 0, 0, time.UTC)
	tests := []struct {
		name       string
		txn        Transaction
		wantStatus int
		want       AuthorizationResponse
	}{
		{
			name: "approved after failover",
			txn: Transaction{
				ID: "txn_1", Approved: true, Acquirer: "AcquirerTwo",
				Attempts: []Attempt{
					{Acquirer: "AcquirerOne", StartedAt: started, Duration: 1500 * time.Microsecond, DeclineReason: ReasonPolicyDecline},
					{Acquirer: "AcquirerTwo", StartedAt: started, Duration: time.Millisecond, Approved: true},
				},
			},
			wantStatus: http.StatusOK,
			want: AuthorizationResponse{
				ID: "txn_1", Status: StatusApproved, Acquirer: "AcquirerTwo",
				Attempts: []AttemptResponse{
					{Acquirer: "AcquirerOne", StartedAt: started, DurationMs: 1.5, Status: StatusDeclined, DeclineReason: ReasonPolicyDecline},
					{Acquirer: "AcquirerTwo", StartedAt: started, DurationMs: 1, Status: StatusApproved},
				},
			},
		},
		{
			name: "hard declined",
			txn: Transaction{
				ID: "txn_2", DeclineReason: ReasonStolenCard,
				Attempts: []Attempt{
					{Acquirer: "AcquirerOne", StartedAt: started, Duration: time.Millisecond, DeclineReason: ReasonStolenCard},
				},
			},
			wantStatus: http.StatusPaymentRequired,
			want: AuthorizationResponse{
				ID: "txn_2", Status: StatusDeclined, DeclineReason: ReasonStolenCard,
				Attempts: []AttemptResponse{
					{Acquirer: "AcquirerOne", StartedAt: started, DurationMs: 1, Status: StatusDeclined, DeclineReason: ReasonStolenCard},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			processor := &stubProcessor{txn: tt.txn}
			rec := post(newTestEngine(processor), validBody)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d, body: %s", rec.Code, tt.wantStatus, rec.Body.String())
			}
			var got AuthorizationResponse
			if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
				t.Fatalf("failed to decode response: %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("response = %+v, want %+v", got, tt.want)
			}

			wantReq := Request{
				MerchantID: "solarbazaar",
				Card:       Card{Number: "4532015112830366", HolderName: "Maria Lopez", ExpiryMonth: 12, ExpiryYear: 2028, CVV: "123"},
				Amount:     1450000,
				Currency:   "MXN",
				Country:    country.MX,
			}
			if processor.last != wantReq {
				t.Fatalf("processor got %+v, want %+v", processor.last, wantReq)
			}
		})
	}
}
