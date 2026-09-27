# yuno-challenge

A payment authorization API that fails over across acquirers. Each
authorization request is tried against acquirers in ranked order until one
approves. The request moves on to the next acquirer only when the decline is
retriable: `SUSPECTED_FRAUD`, `POLICY_DECLINE`, `TIMEOUT` or `GENERIC_DECLINE`.
Issuer declines such as `STOLEN_CARD` or `INSUFFICIENT_FUNDS` stop the chain
right away, because every acquirer would decline them.

**Results:** on the 50 sample transactions, a single acquirer approves 52%.
Failover across three acquirers approves 80% (+28 pp). See
[`METRICS.md`](METRICS.md) for the full comparison and example attempt chains,
and [`evidence/`](evidence/) for the raw logs.

## Running locally

You need Go 1.25+. Everything below runs from `backend/`.

```sh
cd backend
go run .                 # listens on :8080
```

In debug mode (the default for `go run`), leaving `API_KEY` unset turns
authentication off. To require a key locally:

```sh
API_KEY=dev-key go run .
```

| Env var | Default | Purpose |
|---|---|---|
| `PORT` | `8080` | HTTP port |
| `API_KEY` | _(empty)_ | Required `X-API-Key` value for `/v1/*` except `/v1/health`. Must be set when `GIN_MODE=release` |
| `GIN_MODE` | `debug` | `release` in the Docker image |
| `ACQUIRER_ORDER` | `AcquirerOne,AcquirerTwo,AcquirerThree` | Initial routing order. A subset such as `AcquirerOne` gives the single-acquirer baseline |
| `DYNAMIC_RANKING` | `true` | `false` keeps `ACQUIRER_ORDER` fixed instead of reordering acquirers by recent approval rate |

Example request:

```sh
curl -H 'X-API-Key: dev-key' -X POST localhost:8080/v1/authorizations -d '{
  "merchantId": "solarbazaar",
  "amount": "14500.00",
  "currency": "MXN",
  "country": "MX",
  "card": {"number": "4761340000000019", "holderName": "Maria Lopez", "expiry": "202812", "cvv": "123"}
}'
```

AcquirerOne no longer accepts Mexico, so it declines this request with
`POLICY_DECLINE`. The request then fails over to AcquirerTwo, which approves
it. Both attempts appear in the response.

### Endpoints

| Method | Path | Description |
|---|---|---|
| `GET` | `/v1/health` | Liveness check (public) |
| `POST` | `/v1/authorizations` | Authorize a transaction. Returns `200` if approved, `402` if declined (hard decline, or every acquirer tried declined) |
| `GET` | `/v1/authorizations` | Full authorization log with each transaction's attempt chain |
| `GET` | `/v1/analytics` | Overall and per-acquirer approval rates, average attempts and decline reasons |

### Tests

```sh
make test               # unit tests
make integration-test   # starts the real server per scenario and writes the
                        # acceptance report to integration-tests/report.html
make integration-test OUT=../evidence   # also saves logs and analytics per scenario
```

The integration test runs three scenarios: a single acquirer, three acquirers
in fixed order, and three acquirers with dynamic ranking.
[`METRICS.md`](METRICS.md) summarizes the committed run.

### Docker

```sh
docker build -f infra/Dockerfile -t yuno-failover-api backend
docker run --rm -p 8080:8080 -e API_KEY=dev-key yuno-failover-api
```

## Deployment

The service runs on [Render](https://render.com) as a Docker web service at
**https://yuno-failover-api.onrender.com**. Try
`curl https://yuno-failover-api.onrender.com/v1/health`. Every other endpoint
needs the `X-API-Key` header. The key is kept in Render and shared with
reviewers directly.

Every push to `main` that touches `backend/**` or `infra/**` deploys
automatically. The service uses Render's free plan, so it sleeps after 15
minutes without traffic, and the first request after that takes about a
minute. Storage is in memory, so a restart or redeploy clears the
authorization log. See [`infra/README.md`](infra/README.md) for the Blueprint
setup and security notes.

## Architecture

The code lives in a single Go module (`backend/`). Packages are grouped by
domain, not by technical layer. Each domain package holds its own HTTP
controller, DTOs, models and logic.

```
backend/
├── main.go                 wiring: config, acquirers, processor, routes
├── authorization/          authorization domain
├── acquirer/               acquirer domain
│   ├── acquirer.go         Acquirer interface, Rules, simulated issuer checks
│   ├── acq1/               AcquirerOne
│   ├── acq2/               AcquirerTwo
│   └── acq3/               AcquirerThree
├── processor/              failover engine
│   └── ranking/            orders acquirers by recent approval rate
├── shared/                 code shared across domains
│   ├── country/            ISO 3166-1 alpha-2 codes
│   └── sharedgin/          Gin middleware (API key) and error responses
├── testdata/               sample requests and the mock acquirer rules
└── integration-tests/      acceptance test + HTML report
```

At the repo root, `METRICS.md` has the metrics summary, `evidence/` has the
logs from the committed run, and `infra/` has the Dockerfile and Render
Blueprint.

### `authorization`: the authorization domain

Holds everything about an authorization request and its outcome:

- request validation (countries MX, CO, BR, CL and their local currencies,
  card number, expiry)
- the `Transaction` and `Attempt` models, which record the full attempt chain
- `DeclineReason` values and which of them are retriable
- the in-memory `Store`, the analytics summary, and the HTTP controller

The controller depends on a small `Processor` interface rather than on the
processor package. `authorization` imports neither `acquirer` nor
`processor`, so both can depend on it without an import cycle.

### `acquirer`: the acquirer domain

Every acquirer implements the `Acquirer` interface:

```go
type Acquirer interface {
    // Name identifies the acquirer in routing config and attempt logs.
    Name() string
    Authorize(ctx context.Context, req authorization.Request) AuthorizationResponse
}
```

`acq1`, `acq2` and `acq3` are the mock implementations. Each one is configured
with `Rules` that decide which requests it approves: accepted countries,
accepted BIN prefixes, or a random success rate. Each one also runs the shared
simulated issuer checks (expired, stolen or invalid card, insufficient funds).
To add an acquirer, create a new subpackage that implements the interface and
register it in `main.go`.

### `processor`: the failover engine

`processor.Processor` routes a request across acquirers and implements
`authorization.Processor`. It only knows acquirers through the `Acquirer`
interface. For each request it:

1. takes the current acquirer order from `ranking`
2. calls each acquirer in turn, with a 2 s per-attempt timeout that counts as
   a retriable `TIMEOUT`
3. stops at the first approval or the first non-retriable decline
4. feeds each approval or retriable decline back into the ranking (issuer
   declines are not counted against the acquirer)
5. saves the transaction with its attempt chain to the store

`processor/ranking` scores each acquirer with an exponentially weighted moving
average of its recent outcomes. The score decays back towards 1 over time, so
an acquirer that starts declining moves down the order automatically and can
recover once it stops getting traffic. Set `DYNAMIC_RANKING=false` to keep
the configured order, which measures failover on its own.

The brief also suggests ranking by country and card BIN. Those factors aren't
implemented: the ranking only uses each acquirer's recent performance across
all traffic.

### `shared`: cross-domain helpers

`shared/country` holds the ISO 3166-1 alpha-2 country codes used in requests
and acquirer rules. `shared/sharedgin` holds the Gin API key middleware and
the common error response format.
