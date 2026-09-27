# Metrics summary: single acquirer vs failover

The same 50 sample transactions (`testdata.AuthorizationRequests(1)`) were run
through three configurations. Each run started a fresh server with
`MOCK_ACQUIRER_SEED=1`, so AcquirerThree's random decision for a given
request is the same in every scenario and every run:

| | Single acquirer | Multi-acquirer, fixed order | Multi-acquirer, dynamic ranking |
|---|---:|---:|---:|
| Config | `ACQUIRER_ORDER=AcquirerOne` | `ACQUIRER_ORDER=AcquirerOne,AcquirerTwo,AcquirerThree`<br>`DYNAMIC_RANKING=false` | `ACQUIRER_ORDER=AcquirerOne,AcquirerTwo,AcquirerThree` |
| **Approved** | **26/50 (52%)** | **43/50 (86%)** | **43/50 (86%)** |
| on first attempt | 26 | 26 | 41 |
| rescued by failover | 0 | 17 | 2 |
| Declined | 24 | 7 | 7 |
| hard decline (not retried) | 4 | 7 | 7 |
| every acquirer declined | 20 | 0 | 0 |
| Avg attempts per transaction | 1.00 | 1.58 | 1.08 |
| Routing order changes | 0 | 0 | 4 |

**Single acquirer: 52% approved. With failover: 86% approved (+34 pp, 17 more
transactions).**

- **Failover** rescued 17 of the 24 single-acquirer declines: 8 by AcquirerTwo
  and 9 by AcquirerThree.
- **Dynamic ranking** approves the same 43 transactions with 32% fewer
  acquirer calls (1.08 vs 1.58 per transaction). After AcquirerOne starts
  declining the Mexican and `5555` traffic, the ranking moves it down and
  later transactions go straight to an acquirer that approves them. Fewer
  attempts means lower latency at checkout. Ranking can't change which
  transactions are approved here: each one still reaches every acquirer that
  could approve it.
- Hard declines rise from 4 to 7. That's not a regression: three stolen,
  expired or no-funds cards that AcquirerOne rejected on policy grounds now
  reach an acquirer that runs the issuer checks. They still end in a decline,
  just with the correct reason, and the chain stops there.

## Mock acquirer rules

| Acquirer | Declines |
|---|---|
| AcquirerOne | Mexico (`POLICY_DECLINE`), BIN `5555` (`SUSPECTED_FRAUD`) |
| AcquirerTwo | Chile (`POLICY_DECLINE`), BIN `4532` (`SUSPECTED_FRAUD`) |
| AcquirerThree | 20% at random (`GENERIC_DECLINE`), reproducible with `MOCK_ACQUIRER_SEED` |
| All (simulated issuer) | expired, stolen or invalid cards, insufficient funds (non-retriable) |

The rules are in `backend/testdata/authorizations.go`. The dataset has 40%
retriable declines from the primary, 18% declined by the secondary and
approved by the tertiary, and 8% hard declines from the primary.
The brief asked for at least 30%, 10% and 5%. `acquirer/scenarios_test.go`
enforces those minimums.

## Failover in action

Examples from the fixed-order run. The full chains are in
[`evidence/multi-acquirer-fixed-log.json`](evidence/multi-acquirer-fixed-log.json).

| # | Country | BIN | Attempt chain | Outcome |
|---|---|---|---|---|
| 1 | CO | 453201 | AcquirerOne ✓ | Approved by AcquirerOne |
| 27 | MX | 476134 | AcquirerOne `POLICY_DECLINE` → AcquirerTwo ✓ | Approved by AcquirerTwo |
| 35 | MX | 453201 | AcquirerOne `POLICY_DECLINE` → AcquirerTwo `SUSPECTED_FRAUD` → AcquirerThree ✓ | Approved by AcquirerThree |
| 36 | CL | 555544 | AcquirerOne `SUSPECTED_FRAUD` → AcquirerTwo `POLICY_DECLINE` → AcquirerThree ✓ | Approved by AcquirerThree |
| 44 | CO | 400000 | AcquirerOne `STOLEN_CARD` | Hard decline, not retried |
| 48 | MX | 400000 | AcquirerOne `POLICY_DECLINE` → AcquirerTwo `STOLEN_CARD` | Hard decline, chain stops |

The same transactions with dynamic ranking
([`evidence/multi-acquirer-log.json`](evidence/multi-acquirer-log.json)). Once
AcquirerOne is demoted they skip it:

| # | Country | BIN | Attempt chain | Outcome |
|---|---|---|---|---|
| 35 | MX | 453201 | AcquirerTwo `SUSPECTED_FRAUD` → AcquirerThree ✓ | Approved by AcquirerThree |
| 36 | CL | 555544 | AcquirerThree ✓ | Approved by AcquirerThree |
| 44 | CO | 400000 | AcquirerThree `STOLEN_CARD` | Hard decline, not retried |

The service logs the same chain once per transaction, from
[`evidence/multi-acquirer-fixed-server.log`](evidence/multi-acquirer-fixed-server.log):

```
INFO  processor/processor.go:172  Transaction approved by AcquirerThree  {"transactionId": "txn_be58fcd02561e558", "merchantId": "solarbazaar", "country": "MX", "bin": "453201", "routingOrder": ["AcquirerOne", "AcquirerTwo", "AcquirerThree"], "attempts": 3, "chain": ["AcquirerOne:POLICY_DECLINE(1.666µs)", "AcquirerTwo:SUSPECTED_FRAUD(1.25µs)", "AcquirerThree:APPROVED(9.75µs)"]}
```

The dynamic-ranking run changed the routing order 4 times:

| From transaction | Routing order |
|---|---|
| #1 | AcquirerOne > AcquirerTwo > AcquirerThree |
| #28 | AcquirerTwo > AcquirerThree > AcquirerOne |
| #36 | AcquirerThree > AcquirerOne > AcquirerTwo |
| #47 | AcquirerOne > AcquirerTwo > AcquirerThree |
| #49 | AcquirerTwo > AcquirerThree > AcquirerOne |

## Evidence files

[`evidence/`](evidence/) has, for each scenario (`single-acquirer`,
`multi-acquirer-fixed`, `multi-acquirer`):

- `*-log.json`: `GET /v1/authorizations`, the full attempt chain per transaction
- `*-analytics.json`: `GET /v1/analytics`, success rates by acquirer and decline reasons
- `*-server.log`: the server's output, with one log line per transaction

The HTML report with every chain side by side is
[`backend/integration-tests/report.html`](backend/integration-tests/report.html)
([rendered](https://htmlpreview.github.io/?https://github.com/jurgiskg/yuno-challenge/blob/main/backend/integration-tests/report.html)).

## Reproduce

```sh
cd backend
make integration-test OUT=../evidence
```

The integration test sets `MOCK_ACQUIRER_SEED=1`, so it reproduces these
numbers exactly.
