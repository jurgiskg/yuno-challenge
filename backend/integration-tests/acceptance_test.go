//go:build integration

// Package integrationtests runs the failover service end to end and reports how
// much multi-acquirer failover improves the approval rate over a single acquirer.
//
// Each scenario builds and starts the real server with its own ACQUIRER_ORDER
// and DYNAMIC_RANKING (so in-memory state starts fresh), submits the sample authorization requests,
// then fetches the full authorization log and the analytics summary. The test
// analyzes the log itself, cross-checks it against the server's summary, and
// writes an HTML comparison report once every scenario has run (overwritten on
// each run, integration-tests/report.html by default):
//
//	go test -tags integration -count=1 -v ./integration-tests/
//
// Pass -report <file> to write the report elsewhere, and -out <dir> to also save
// each scenario's log, analytics and server output as demo evidence. The report
// never fails the test. Every local server gets the same MOCK_ACQUIRER_SEED, so
// AcquirerThree's random approvals are the same for a given request in every
// scenario and every run.
//
// To test an already deployed service instead, set INTEGRATION_BASE_URL and
// INTEGRATION_API_KEY. That runs a single "deployed" scenario with whatever
// routing the service is configured with; since its store holds other traffic
// too, only the transactions this run submitted are analyzed.
package integrationtests

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"yuno-challenge/authorization"
	"yuno-challenge/shared/sharedgin"
	"yuno-challenge/testdata"
)

const (
	// localAPIKey is the key the locally started servers are given.
	localAPIKey = "integration-test-key"
	// seed selects the sample dataset; 1 matches acquirer/scenarios_test.go.
	seed = 1
	// acquirerSeed makes the mock acquirers' random approvals reproducible.
	acquirerSeed = 1
)

// httpClient allows for a deployed service waking from sleep.
var httpClient = &http.Client{Timeout: 90 * time.Second}

var (
	reportPath = flag.String("report", "report.html", "file to write the HTML acceptance report to, overwritten on each run")
	outDir     = flag.String("out", "", "directory to save each scenario's log, analytics and server output")
)

type scenario struct {
	name          string
	description   string
	acquirerOrder string
	// fixedOrder turns dynamic ranking off, so failover alone is measured.
	fixedOrder bool
}

// scenarios run in order; the first is the baseline the others are compared to.
var scenarios = []scenario{
	{
		name:          "single-acquirer",
		description:   "Single acquirer",
		acquirerOrder: "AcquirerOne",
	},
	{
		name:          "multi-acquirer-fixed",
		description:   "Multi-acquirer, fixed order",
		acquirerOrder: "AcquirerOne,AcquirerTwo,AcquirerThree",
		fixedOrder:    true,
	},
	{
		name:          "multi-acquirer",
		description:   "Multi-acquirer, dynamic ranking",
		acquirerOrder: "AcquirerOne,AcquirerTwo,AcquirerThree",
	},
}

type result struct {
	scenario scenario
	analysis analysis
	log      []authorization.TransactionResponse
}

func TestAcceptance(t *testing.T) {
	reqs := testdata.AuthorizationRequests(seed)

	var results []result
	if baseURL := strings.TrimRight(os.Getenv("INTEGRATION_BASE_URL"), "/"); baseURL != "" {
		sc := scenario{name: "deployed", description: "Deployed service"}
		t.Run(sc.name, func(t *testing.T) {
			srv := &server{baseURL: baseURL, apiKey: os.Getenv("INTEGRATION_API_KEY"), output: &syncBuffer{}}
			srv.waitHealthy(t, 2*time.Minute, nil)
			if r, ok := runScenario(t, sc, srv, reqs, true); ok {
				results = append(results, r)
			}
		})
	} else {
		binary := buildServer(t)
		for _, sc := range scenarios {
			t.Run(sc.name, func(t *testing.T) {
				srv := startServer(t, binary, sc)
				if r, ok := runScenario(t, sc, srv, reqs, false); ok {
					results = append(results, r)
				}
			})
		}
	}

	if len(results) == 0 {
		return
	}
	path, err := filepath.Abs(*reportPath)
	if err != nil {
		t.Fatalf("invalid -report path: %v", err)
	}
	if err := writeReport(path, len(reqs), results); err != nil {
		t.Fatalf("failed to write report: %v", err)
	}
	fmt.Printf("\nAcceptance report: file://%s\n\n", path)
}

// runScenario submits reqs to srv, then fetches and checks the log and analytics.
// A shared server also holds other traffic, so only this run's transactions are
// analyzed and the analytics can only be checked to cover them.
func runScenario(t *testing.T, sc scenario, srv *server, reqs []authorization.CreateAuthorizationRequest, shared bool) (result, bool) {
	t.Helper()
	responses := submit(t, srv, reqs)

	var log authorization.ListAuthorizationsResponse
	srv.get(t, "/v1/authorizations", &log)
	var summary authorization.Summary
	srv.get(t, "/v1/analytics", &summary)

	txns := log.Transactions
	if shared {
		txns = submitted(responses, txns)
	}
	checkLog(t, responses, txns)
	a := analyze(txns)
	if shared {
		if summary.Transactions < a.total {
			t.Errorf("analytics covers %d transactions, fewer than the %d this run submitted", summary.Transactions, a.total)
		}
	} else {
		checkSummary(t, a, summary)
	}
	saveEvidence(t, sc, log, summary, srv)

	if sc.acquirerOrder == "" && len(txns) > 0 {
		sc.acquirerOrder = strings.Join(txns[0].RoutingOrder, ",")
	}
	return result{scenario: sc, analysis: a, log: txns}, !t.Failed()
}

// submitted returns the log entries for responses, in log order.
func submitted(responses []authorization.AuthorizationResponse, log []authorization.TransactionResponse) []authorization.TransactionResponse {
	ids := make(map[string]bool, len(responses))
	for _, r := range responses {
		ids[r.ID] = true
	}
	var out []authorization.TransactionResponse
	for _, txn := range log {
		if ids[txn.ID] {
			out = append(out, txn)
		}
	}
	return out
}

// buildServer compiles the backend into a temporary binary.
func buildServer(t *testing.T) string {
	t.Helper()
	binary := filepath.Join(t.TempDir(), "server")
	cmd := exec.Command("go", "build", "-o", binary, "..")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to build server: %v\n%s", err, out)
	}
	return binary
}

type server struct {
	baseURL string
	apiKey  string
	// output is the server's stdout and stderr; empty for a deployed service.
	output *syncBuffer
}

// syncBuffer is a bytes.Buffer that can be read while the server writes to it.
type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *syncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

// startServer runs the binary on a free port and waits until it is healthy. It
// is stopped when the test finishes; its output is logged if the test failed.
func startServer(t *testing.T, binary string, sc scenario) *server {
	t.Helper()
	port := freePort(t)
	srv := &server{baseURL: "http://127.0.0.1:" + port, apiKey: localAPIKey, output: &syncBuffer{}}

	cmd := exec.Command(binary)
	cmd.Env = append(os.Environ(),
		"PORT="+port,
		"API_KEY="+localAPIKey,
		"ACQUIRER_ORDER="+sc.acquirerOrder,
		fmt.Sprintf("DYNAMIC_RANKING=%t", !sc.fixedOrder),
		fmt.Sprintf("MOCK_ACQUIRER_SEED=%d", acquirerSeed),
		"GIN_MODE=release",
	)
	cmd.Stdout = srv.output
	cmd.Stderr = srv.output
	if err := cmd.Start(); err != nil {
		t.Fatalf("failed to start server: %v", err)
	}
	exited := make(chan error, 1)
	go func() { exited <- cmd.Wait() }()

	t.Cleanup(func() {
		_ = cmd.Process.Signal(os.Interrupt)
		select {
		case <-exited:
		case <-time.After(5 * time.Second):
			_ = cmd.Process.Kill()
			<-exited
		}
		if t.Failed() {
			t.Logf("server output:\n%s", srv.output)
		}
	})

	srv.waitHealthy(t, 10*time.Second, exited)
	return srv
}

// waitHealthy polls /v1/health until it answers 200. exited, if not nil, reports
// the local server process exiting early.
func (s *server) waitHealthy(t *testing.T, timeout time.Duration, exited chan error) {
	t.Helper()
	deadline := time.After(timeout)
	for {
		if resp, err := httpClient.Get(s.baseURL + "/v1/health"); err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return
			}
		}
		select {
		case err := <-exited:
			exited <- err // let the cleanup see it too
			t.Fatalf("server exited before becoming healthy: %v", err)
		case <-deadline:
			t.Fatalf("%s did not become healthy within %s", s.baseURL, timeout)
		case <-time.After(50 * time.Millisecond):
		}
	}
}

func freePort(t *testing.T) string {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to find a free port: %v", err)
	}
	defer l.Close()
	return fmt.Sprint(l.Addr().(*net.TCPAddr).Port)
}

func (s *server) do(t *testing.T, method, path string, body any) (int, []byte) {
	t.Helper()
	var reqBody io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("failed to encode request: %v", err)
		}
		reqBody = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, s.baseURL+path, reqBody)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(sharedgin.APIKeyHeader, s.apiKey)
	resp, err := httpClient.Do(req)
	if err != nil {
		t.Fatalf("%s %s failed: %v", method, path, err)
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("failed to read %s %s response: %v", method, path, err)
	}
	return resp.StatusCode, respBody
}

func (s *server) get(t *testing.T, path string, into any) {
	t.Helper()
	status, body := s.do(t, http.MethodGet, path, nil)
	if status != http.StatusOK {
		t.Fatalf("GET %s: status %d, body: %s", path, status, body)
	}
	if err := json.Unmarshal(body, into); err != nil {
		t.Fatalf("failed to decode GET %s response: %v", path, err)
	}
}

// submit sends the requests one at a time, in order, so the ranking sees them
// in sequence. The service answers 200 when approved and 400 when declined.
func submit(t *testing.T, srv *server, reqs []authorization.CreateAuthorizationRequest) []authorization.AuthorizationResponse {
	t.Helper()
	responses := make([]authorization.AuthorizationResponse, len(reqs))
	for i, req := range reqs {
		status, body := srv.do(t, http.MethodPost, "/v1/authorizations", req)
		var resp authorization.AuthorizationResponse
		if err := json.Unmarshal(body, &resp); err != nil {
			t.Fatalf("request %d: status %d, undecodable body: %s", i, status, body)
		}
		wantStatus := http.StatusBadRequest
		if resp.Status == authorization.StatusApproved {
			wantStatus = http.StatusOK
		}
		if status != wantStatus || resp.ID == "" {
			t.Fatalf("request %d: status %d for a %s response, body: %s", i, status, resp.Status, body)
		}
		responses[i] = resp
	}
	return responses
}

// checkLog verifies the log holds exactly the submitted transactions, in order.
func checkLog(t *testing.T, responses []authorization.AuthorizationResponse, log []authorization.TransactionResponse) {
	t.Helper()
	if len(log) != len(responses) {
		t.Fatalf("log has %d transactions, want %d", len(log), len(responses))
	}
	for i, txn := range log {
		resp := responses[i]
		if txn.ID != resp.ID || txn.Status != resp.Status || len(txn.Attempts) != len(resp.Attempts) {
			t.Errorf("log entry %d = %s %s (%d attempts), response was %s %s (%d attempts)",
				i, txn.ID, txn.Status, len(txn.Attempts), resp.ID, resp.Status, len(resp.Attempts))
		}
	}
}

// checkSummary verifies the server's analytics agree with the test's own
// analysis of the log.
func checkSummary(t *testing.T, a analysis, s authorization.Summary) {
	t.Helper()
	if s.Transactions != a.total || s.Approved != a.approved || s.ApprovedAfterFailover != a.rescued {
		t.Errorf("analytics transactions/approved/afterFailover = %d/%d/%d, log analysis = %d/%d/%d",
			s.Transactions, s.Approved, s.ApprovedAfterFailover, a.total, a.approved, a.rescued)
	}
	if len(s.Acquirers) != len(a.acquirers) {
		t.Errorf("analytics covers %d acquirers, log analysis %d", len(s.Acquirers), len(a.acquirers))
	}
	for _, as := range s.Acquirers {
		st := a.acquirers[as.Acquirer]
		if st == nil || as.Attempts != st.tried || as.Approved != st.approved {
			t.Errorf("analytics for %s = %d tried/%d approved, log analysis = %+v", as.Acquirer, as.Attempts, as.Approved, st)
		}
	}
}

// saveEvidence writes the scenario's log, analytics and server output to -out.
func saveEvidence(t *testing.T, sc scenario, log authorization.ListAuthorizationsResponse, summary authorization.Summary, srv *server) {
	t.Helper()
	if *outDir == "" {
		return
	}
	if err := os.MkdirAll(*outDir, 0o755); err != nil {
		t.Fatalf("failed to create -out dir: %v", err)
	}
	write := func(name string, data []byte) {
		if err := os.WriteFile(filepath.Join(*outDir, sc.name+"-"+name), data, 0o644); err != nil {
			t.Fatalf("failed to write %s: %v", name, err)
		}
	}
	for name, v := range map[string]any{"log.json": log, "analytics.json": summary} {
		b, err := json.MarshalIndent(v, "", "  ")
		if err != nil {
			t.Fatalf("failed to encode %s: %v", name, err)
		}
		write(name, b)
	}
	if out := srv.output.String(); out != "" {
		write("server.log", []byte(out))
	}
}
