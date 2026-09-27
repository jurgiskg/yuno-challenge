# yuno-challenge

A payment authorization API that fails over across acquirers. A request is
tried against acquirers in ranked order until one approves. It moves to the
next acquirer only on a retriable decline (`SUSPECTED_FRAUD`,
`POLICY_DECLINE`, `TIMEOUT`, `GENERIC_DECLINE`). Issuer declines such as
`STOLEN_CARD` stop the chain.

On the 50 sample transactions, a single acquirer approves 52% and failover
approves 86%. See [`METRICS.md`](METRICS.md).

## Running locally

Requires Go 1.25+.

```sh
cd backend
go run .                 # listens on :8080, no auth when API_KEY is unset
API_KEY=dev-key go run . # require X-API-Key
```

| Env var | Default | Purpose |
|---|---|---|
| `PORT` | `8080` | HTTP port |
| `API_KEY` | _(empty)_ | Required `X-API-Key` for `/v1/*` except `/v1/health`. Mandatory when `GIN_MODE=release` |
| `GIN_MODE` | `debug` | `release` in the Docker image |
| `ACQUIRER_ORDER` | `AcquirerOne,AcquirerTwo,AcquirerThree` | Initial routing order. `AcquirerOne` alone gives the single-acquirer baseline |
| `DYNAMIC_RANKING` | `true` | `false` keeps `ACQUIRER_ORDER` fixed |

```sh
curl -H 'X-API-Key: dev-key' -X POST localhost:8080/v1/authorizations -d '{
  "merchantId": "solarbazaar",
  "amount": "14500.00",
  "currency": "MXN",
  "country": "MX",
  "card": {"number": "4761340000000019", "holderName": "Maria Lopez", "expiry": "202812", "cvv": "123"}
}'
```

AcquirerOne declines Mexico with `POLICY_DECLINE`, so this fails over to
AcquirerTwo, which approves. The response lists both attempts.

| Method | Path | Description |
|---|---|---|
| `GET` | `/v1/health` | Liveness check (public) |
| `POST` | `/v1/authorizations` | Authorize a transaction: `200` approved, `400` declined or invalid |
| `GET` | `/v1/authorizations` | Authorization log with each attempt chain |
| `GET` | `/v1/analytics` | Approval rates, average attempts and decline reasons, overall and per acquirer |

### Tests

```sh
make test                               # unit tests
make integration-test OUT=../evidence   # single- vs multi-acquirer report and evidence
```

### Docker

```sh
docker build -f infra/Dockerfile -t yuno-failover-api backend
docker run --rm -p 8080:8080 -e API_KEY=dev-key yuno-failover-api
```

## Deployment

Deployed on Render at **https://yuno-failover-api.onrender.com**. Endpoints
other than `/v1/health` need the `X-API-Key` header. The key was shared with
Alejandro Serafin Lopez Perez; contact him to get it.

Every push to `main` that touches `backend/**` or `infra/**` deploys
automatically. On the free plan the service sleeps after 15 idle minutes, so
the first request takes about a minute, and a redeploy clears the in-memory
log. See [`infra/README.md`](infra/README.md).

## Metrics

[`METRICS.md`](METRICS.md) compares a single acquirer, three acquirers in a
fixed order and three with dynamic ranking. It includes example attempt
chains. The raw logs are in [`evidence/`](evidence/), and every chain is in
[`report.html`](https://htmlpreview.github.io/?https://github.com/jurgiskg/yuno-challenge/blob/main/backend/integration-tests/report.html).

## Architecture

One Go module in `backend/`, with packages grouped by domain.

```
backend/
├── main.go             wiring: config, acquirers, processor, routes
├── authorization/      request validation, Transaction/Attempt models, decline reasons,
│                       in-memory store, analytics, HTTP controller
├── acquirer/           Acquirer interface, Rules, simulated issuer checks
│   └── mock/           mock acquirer, configured with a name and Rules
├── processor/          failover engine
│   └── ranking/        orders acquirers by recent approval rate
├── shared/
│   ├── country/        ISO 3166-1 alpha-2 codes
│   └── sharedgin/      API key middleware, error responses
├── testdata/           sample requests and mock acquirer rules
└── integration-tests/  acceptance test and HTML report
```

- **`acquirer`:** `main.go` creates AcquirerOne, AcquirerTwo and AcquirerThree
  with `mock.New(name, rules)`. Each applies its `Rules` (accepted countries,
  accepted BIN prefixes or a random success rate), then the shared issuer
  checks (expired, stolen or invalid card, insufficient funds).
- **`processor`:** takes the order from `ranking` and calls each acquirer
  with a 2 s timeout, which counts as a retriable `TIMEOUT`. It stops at the
  first approval or non-retriable decline and saves the attempt chain. Only
  approvals and retriable declines feed the ranking.
- **`processor/ranking`:** scores each acquirer with an exponentially
  weighted moving average of recent outcomes. Scores decay back towards 1, so
  a demoted acquirer recovers once it stops getting traffic.
- **`authorization`:** imports neither `acquirer` nor `processor`, which
  avoids an import cycle. Its controller depends on a small `Processor`
  interface.

## Tradeoffs

- **No real database.** The authorization log and the ranking scores are in
  memory, so a restart clears them and they can't be shared between
  instances. In production the log would go in a database behind the same
  `Store` methods, and the ranking in shared storage such as Redis.
- **Layers are kept together inside each domain.** A bigger project would
  split each domain, for example into `authorization/http` and
  `authorization/repo`. With a few files per domain, that would only add
  indirection.
- **`shared/` has a single consumer.** In a monorepo it would be its own
  module reused across services.
- **Smart ranking is incomplete.** It uses one signal, a global approval
  rate. A real router would also rank by country, currency, BIN or card
  brand, latency, fees and merchant agreements. It would skip acquirers that
  can't handle a request, rather than learning that from declines.
- **`400` for a declined authorization is debatable** and should be agreed
  with the merchant. A `500` could also be argued, because the request failed
  at third parties (the acquirers), not because of a problem in our code.
  The same `400` is returned for invalid requests, and the response body
  tells the two apart.
- **Acquirer rules are hardcoded in `testdata`.** A mock acquirer takes its
  name and `Rules` as configuration, so the acquirers could be loaded from a
  config file or env vars instead.
- **Integration tests are written in Go** as a test package inside the
  service module, for simplicity. A real project would use a dedicated
  integration test framework in a separate Nx package in the monorepo.
- **Render for deployment.** It deploys automatically on every push to
  `main` with little setup. A real deployment would manage infrastructure as
  code with Terraform, which is more flexible and easier to maintain.
