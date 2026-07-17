# Withdrawals V2 Monitor

The V2 withdrawals monitor re-verifies, **from L1 only**, that every withdrawal the
`OptimismPortal2` contract accepted carries a proof that a correct verifier also
accepts. It is a *portal proof-verification integrity* monitor: it catches the case
where the portal accepted a withdrawal proof that is actually invalid
(`bad_withdrawal_proof`, P0).

## Withdrawal Validation Process

For each `WithdrawalProvenExtension1` event emitted by `OptimismPortal2`:

1. **Trace the proving transaction** (`debug_traceTransaction`, `callTracer`) and
   collect every internal `proveWithdrawalTransaction` call reaching the portal.
   The proof arguments are *ephemeral* (never stored on-chain) and a wrapper/relayer
   contract can construct them internally so they never appear in the top-level
   calldata — the trace is the only reliable source.
2. **Match by withdrawal hash**: a batch relayer proves many withdrawals in one tx,
   so the prove call whose `hashWithdrawal(_tx)` equals the event's hash is selected.
3. **Read the dispute game root claim** the withdrawal was proven against
   (`provenWithdrawals[wdHash][submitter].disputeGameProxy` → `game.rootClaim()`).
   This is a plain data read — **not** a game-validity gate.
4. **Independently re-verify** the two checks the portal performs:
   - `(a)` `keccak256(abi.encode(outputRootProof)) == game.rootClaim()`
   - `(b)` the withdrawal hash is included in the `L2ToL1MessagePasser` storage trie
     rooted at `outputRootProof.messagePasserStorageRoot`, with sentinel value `1`.
5. If the portal accepted the withdrawal but a correct verifier rejects `(a)` or
   `(b)`, that is a **P0** — and it holds regardless of whether the dispute game is
   valid, so proof-system bugs are caught even on invalid games.

There is **no L2 dependency** — the monitor only talks to an L1 archive+trace node.

### Metrics

- `withdrawals_v2_valid_withdrawals_total`: withdrawals whose proof re-verified correctly.
- `withdrawals_v2_invalid_withdrawals_total{reason}`: **P0** — the portal accepted a proof
  a correct verifier rejects. `reason` is one of:
  - `bad_withdrawal_proof`: storage-inclusion proof does not prove inclusion.
  - `bad_output_root_binding`: `keccak(outputRootProof)` does not equal the game root claim.
- `withdrawals_v2_unverifiable_withdrawals_total{reason}`: counts each prove event that
  could not be re-verified on first sight (then parked for async retry). `reason` is one of
  `trace_unavailable`, `no_prove_call_found`, `decode_error`, `game_read_error`.

Counters are labelled only by `reason` (aggregate) to keep cardinality bounded; the
offending tx and withdrawal hashes are in the structured logs and the pending store.
- `withdrawals_v2_pending_withdrawals`: **gauge** — prove events awaiting a terminal
  valid/invalid verdict. The async retry loop drives each to a verdict, so no event is
  ever silently dropped after a transient RPC failure.
- `withdrawals_v2_oldest_pending_seconds`: **gauge** — age of the oldest still-pending
  event. This is the alert signal: a value that keeps climbing means an event never
  reached a verdict and needs an operator.

Every prove event reaches a terminal verdict: `valid`, `invalid` (P0), or it stays
counted in the pending gauges until it does. Dispute games are resolved by the traced
`_disputeGameIndex` via `gameAtIndex` (immutable), and the output-root claim is selected
exactly as the portal does — `rootClaimByChainId(l2ChainId)` for super game types
(4/5/7/9), `rootClaim()` for legacy — so the re-verification mirrors on-chain semantics
for all supported game types.

## Requirements

- **L1 archive + trace node**: the node must serve `debug_traceTransaction` for the
  blocks being scanned. A pruned full node only retains ~128 blocks of state, so it
  can only trace withdrawals proven in the last ~25 minutes; scanning older blocks
  (or backfilling) requires a true archive (`--gcmode=archive --state.scheme=hash`).
  When a trace is unavailable the monitor parks the event as pending and retries it
  asynchronously (see the pending gauges) rather than stalling or dropping it.

### Startup modes

- **Backfill** (one-time): set `--start.block` to a historical block; the monitor scans
  forward from there (inclusive).
- **Steady-state** (default): omit `--start.block`. The monitor starts near the finalized
  head and, on each startup, re-scans `--lookback.blocks` (default 900) below the head so
  any events proven while it was down are re-evaluated. This makes the in-memory pending
  set durable across restarts without external storage.

## CLI Usage

### Command Structure

```bash
go run ../cmd/monitorism withdrawals-v2 [options]
```

### Available Options

```bash
OPTIONS:
   --l1.node.url value             Node URL of L1 archive+trace Geth node (must serve debug_traceTransaction) [$WITHDRAWALS_V2_MON_L1_NODE_URL]
   --start.block value             Starting L1 block number to scan (one-time backfill); omit for steady-state [$WITHDRAWALS_V2_MON_START_BLOCK]
   --lookback.blocks value         When no --start.block is set, re-scan this many blocks below finalized on startup (default: 900) [$WITHDRAWALS_V2_MON_LOOKBACK_BLOCKS]
   --poll.interval value           Polling interval for scanning L1 blocks (default: 1s) [$WITHDRAWALS_V2_MON_POLL_INTERVAL]
   --optimism.portal.address value Address of the OptimismPortal2 contract [$WITHDRAWALS_V2_MON_OPTIMISM_PORTAL]
   --log.level value               The lowest log level that will be output (default: INFO) [$MONITORISM_LOG_LEVEL]
   --log.format value              Format the log output. Supported formats: 'text', 'terminal', 'logfmt', 'json', 'json-pretty', (default: text) [$MONITORISM_LOG_FORMAT]
   --log.color                     Color the log output if in terminal mode (default: false) [$MONITORISM_LOG_COLOR]
   --metrics.enabled               Enable the metrics server (default: false) [$MONITORISM_METRICS_ENABLED]
   --metrics.addr value            Metrics listening address (default: "0.0.0.0") [$MONITORISM_METRICS_ADDR]
   --metrics.port value            Metrics listening port (default: 7300) [$MONITORISM_METRICS_PORT]
   --loop.interval.msec value      Loop interval of the monitor in milliseconds (default: 60000) [$MONITORISM_LOOP_INTERVAL_MSEC]
   --help, -h                      show help
```

### Example Usage

```bash
# Monitor OP Mainnet withdrawals starting from block 21000000
go run ../cmd/monitorism withdrawals-v2 \
  --l1.node.url "http://your-l1-archive-node:8545" \
  --start.block 21000000 \
  --optimism.portal.address "0xbEb5Fc579115071764c7423A4f12eDde41f106Ed" \
  --poll.interval 12s \
  --metrics.enabled \
  --metrics.port 7301
```
