# Metrics summary: single acquirer vs failover

The same 50 sample transactions (`testdata.AuthorizationRequests(1)`) were run
through three configurations. Each run started a fresh server:

| | Single acquirer | Multi-acquirer, fixed order | Multi-acquirer, dynamic ranking |
|---|---:|---:|---:|
| Config | `ACQUIRER_ORDER=AcquirerOne` | `ACQUIRER_ORDER=AcquirerOne,AcquirerTwo,AcquirerThree`<br>`DYNAMIC_RANKING=false` | `ACQUIRER_ORDER=AcquirerOne,AcquirerTwo,AcquirerThree` |
| **Approved** | **26/50 (52%)** | **40/50 (80%)** | **40/50 (80%)** |
| on first attempt | 26 | 26 | 38 |
| rescued by failover | 0 | 14 | 2 |
| Declined | 24 | 10 | 10 |
| hard decline (not retried) | 4 | 7 | 7 |
| every acquirer declined | 20 | 3 | 3 |
| Avg attempts per transaction | 1.00 | 1.58 | 1.20 |
| Routing order changes | 0 | 0 | 3 |

**Single acquirer: 52% approved. With failover: 80% approved (+28 pp, 14 more
transactions).**

- **Failover alone** (fixed order) accounts for the whole uplift. 14 of the 24
  single-acquirer declines were rescued: 8 by AcquirerTwo and 6 by
  AcquirerThree.
- **Dynamic ranking** reaches the same approval rate with 24% fewer acquirer
  calls (1.20 vs 1.58 per transaction). After AcquirerOne starts declining the
  Mexican and `5555` traffic, the ranking moves it down and later transactions
  go straight to an acquirer that approves them. Fewer attempts means lower
  latency at checkout.
- Hard declines rise from 4 to 7. That's not a regression: three stolen,
  expired or no-funds cards that AcquirerOne rejected on policy grounds now
  reach an acquirer that runs the issuer checks. They still end in a decline,
  just with the correct reason, and the chain stops there.

AcquirerThree approves 80% at random, so the multi-acquirer numbers can vary
slightly between runs. The numbers above are from the committed run.

## Mock acquirer rules

| Acquirer | Declines |
|---|---|
| AcquirerOne | Mexico (`POLICY_DECLINE`), BIN `5555` (`SUSPECTED_FRAUD`) |
| AcquirerTwo | Chile (`POLICY_DECLINE`), BIN `4532` (`SUSPECTED_FRAUD`) |
| AcquirerThree | 20% at random (`GENERIC_DECLINE`) |
| All (simulated issuer) | expired, stolen or invalid cards, insufficient funds (non-retriable) |

The rules are in `backend/testdata/authorizations.go`. In this run the dataset
had 40% retriable declines from the primary, 12% declined by the secondary and
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
| 37 | MX | 453201 | AcquirerOne `POLICY_DECLINE` → AcquirerTwo `SUSPECTED_FRAUD` → AcquirerThree `GENERIC_DECLINE` | Declined after 3 attempts |
| 44 | CO | 400000 | AcquirerOne `STOLEN_CARD` | Hard decline, not retried |
| 48 | MX | 400000 | AcquirerOne `POLICY_DECLINE` → AcquirerTwo `STOLEN_CARD` | Hard decline, chain stops |

The service logs the same chain once per transaction, from
[`evidence/multi-acquirer-fixed-server.log`](evidence/multi-acquirer-fixed-server.log):

```
INFO  processor/processor.go:172  Transaction approved by AcquirerThree  {"transactionId": "txn_25b775eb852e1ea6", "merchantId": "solarbazaar", "country": "MX", "bin": "453201", "routingOrder": ["AcquirerOne", "AcquirerTwo", "AcquirerThree"], "attempts": 3, "chain": ["AcquirerOne:POLICY_DECLINE(916ns)", "AcquirerTwo:SUSPECTED_FRAUD(583ns)", "AcquirerThree:APPROVED(708ns)"]}
```

The dynamic-ranking run changed the routing order 3 times:

| From transaction | Routing order |
|---|---|
| #1 | AcquirerOne > AcquirerTwo > AcquirerThree |
| #28 | AcquirerTwo > AcquirerThree > AcquirerOne |
| #36 | AcquirerThree > AcquirerOne > AcquirerTwo |
| #50 | AcquirerThree > AcquirerTwo > AcquirerOne |

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
