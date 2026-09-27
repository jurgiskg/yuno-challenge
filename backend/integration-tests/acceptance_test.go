//go:build integration

// Package integrationtests runs the failover service end to end and reports how
// much multi-acquirer failover improves the approval rate over a single acquirer.
//
// Each scenario builds and starts the real server with its own ACQUIRER_ORDER
// (so in-memory state starts fresh), submits the sample authorization requests,
// then fetches the full authorization log and the analytics summary. The test
// analyzes the log itself, cross-checks it against the server's summary, and
// prints a comparison report once every scenario has run:
//
//	go test -tags integration -count=1 -v ./integration-tests/
//
// Pass -out <dir> to also save each scenario's log, analytics and server output
// as demo evidence. The report never fails the test: AcquirerThree approves at
// random, so the multi-acquirer numbers vary slightly between runs.
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
	"sync"
	"testing"
	"time"

	"yuno-challenge/authorization"
	"yuno-challenge/merchant"
	"yuno-challenge/sharedgin"
	"yuno-challenge/testdata"
)

const (
	apiKey = "integration-test-key"
	// seed selects the sample dataset; 1 matches acquirer/scenarios_test.go.
	seed = 1
)

var outDir = flag.String("out", "", "directory to save each scenario's log, analytics and server output")

type scenario struct {
	name          string
	description   string
	acquirerOrder string
}

// scenarios run in order; the first is the baseline the others are compared to.
var scenarios = []scenario{
	{
		name:          "single-acquirer",
		description:   "Single acquirer",
		acquirerOrder: "AcquirerOne",
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
}

func TestAcceptance(t *testing.T) {
	binary := buildServer(t)
	reqs := testdata.AuthorizationRequests(seed)

	var results []result
	for _, sc := range scenarios {
		t.Run(sc.name, func(t *testing.T) {
			srv := startServer(t, binary, sc)
			responses := submit(t, srv, reqs)

			var log authorization.ListAuthorizationsResponse
			srv.get(t, "/v1/authorizations", &log)
			var summary authorization.Summary
			srv.get(t, "/v1/analytics", &summary)

			a := analyze(log.Transactions)
			checkLog(t, responses, log.Transactions)
			checkSummary(t, a, summary)
			saveEvidence(t, sc, log, summary, srv)

			results = append(results, result{scenario: sc, analysis: a})
		})
	}

	printReport(os.Stdout, len(reqs), results)
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
	output  *syncBuffer
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
	srv := &server{baseURL: "http://127.0.0.1:" + port, output: &syncBuffer{}}

	cmd := exec.Command(binary)
	cmd.Env = append(os.Environ(),
		"PORT="+port,
		"API_KEY="+apiKey,
		"ACQUIRER_ORDER="+sc.acquirerOrder,
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

	deadline := time.After(10 * time.Second)
	for {
		select {
		case err := <-exited:
			exited <- err // let the cleanup see it too
			t.Fatalf("server exited before becoming healthy: %v", err)
		case <-deadline:
			t.Fatal("server did not become healthy within 10s")
		case <-time.After(50 * time.Millisecond):
		}
		if resp, err := http.Get(srv.baseURL + "/v1/health"); err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return srv
			}
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
	req.Header.Set(sharedgin.APIKeyHeader, apiKey)
	resp, err := http.DefaultClient.Do(req)
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
// in sequence. The service answers 200 when approved and 500 when declined.
func submit(t *testing.T, srv *server, reqs []merchant.CreateAuthorizationRequest) []merchant.AuthorizationResponse {
	t.Helper()
	responses := make([]merchant.AuthorizationResponse, len(reqs))
	for i, req := range reqs {
		status, body := srv.do(t, http.MethodPost, "/v1/authorizations", req)
		var resp merchant.AuthorizationResponse
		if err := json.Unmarshal(body, &resp); err != nil {
			t.Fatalf("request %d: status %d, undecodable body: %s", i, status, body)
		}
		wantStatus := http.StatusInternalServerError
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
func checkLog(t *testing.T, responses []merchant.AuthorizationResponse, log []authorization.TransactionResponse) {
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
	write("server.log", []byte(srv.output.String()))
}
