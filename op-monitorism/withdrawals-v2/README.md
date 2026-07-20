# Withdrawals V2 Monitor

The V2 withdrawals monitor re-verifies, **from L1 only**, proofs accepted by
`OptimismPortal2`. It is a *portal proof-verification integrity* monitor: it catches
the case where the portal accepted a proof that a correct verifier rejects
(`bad_withdrawal_proof` or `bad_output_root_binding`, P0).

This monitor is deliberately not a complete end-to-end withdrawal monitor. It does
not watch `WithdrawalFinalized`, check `sentMessages[withdrawalHash]` against the
canonical safe L2 at finalization, or detect duplicate finalizations. Keep
`faultproof-withdrawals` (or equivalent companion coverage) running for those
predicates; this monitor must not be used by itself as the basis for sunsetting that
coverage.

## Withdrawal Validation Process

For each `WithdrawalProvenExtension1` event emitted by `OptimismPortal2`:

1. **Trace the proving transaction** (`debug_traceTransaction`, `callTracer`) and
   collect every internal `proveWithdrawalTransaction` call reaching the portal.
   The proof arguments are *ephemeral* (never stored on-chain) and a wrapper/relayer
   contract can construct them internally so they never appear in the top-level
   calldata — the trace is the only reliable source.
2. **Match by withdrawal hash and event position**: a batch relayer can prove many
   withdrawals, including the same hash more than once, so matching candidates are
   paired with events in trace/log order.
3. **Recover the exact historical factory and game** from the successful prove
   frame's `gameAtIndex(_disputeGameIndex)` call and return data. The monitor never
   reads mutable `provenWithdrawals` and never applies a startup-cached factory to
   historical events, so re-proves and AnchorStateRegistry/factory migrations cannot
   redirect verification to the wrong game index.
4. **Read the dispute game's output-root claim**. Legacy games use `rootClaim()`;
   super games (types 4/5/7/9) use
   `rootClaimByChainId(systemConfig.l2ChainId())`, matching the portal. This is a
   plain data read — **not** a game-validity gate.
5. **Independently re-verify** the two checks the portal performs:
   - `(a)` `keccak256(abi.encode(outputRootProof)) == selected output-root claim`
   - `(b)` the withdrawal hash is included in the `L2ToL1MessagePasser` storage trie
     rooted at `outputRootProof.messagePasserStorageRoot`, with sentinel value `1`.
6. If the portal accepted the proof but a correct verifier rejects `(a)` or
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
offending tx and withdrawal hashes are in structured logs and the in-memory pending
store.

- `withdrawals_v2_pending_withdrawals`: **gauge** — prove events awaiting a terminal
  valid/invalid verdict. The async retry loop keeps each event pending for the life of
  the process rather than dropping it after a transient RPC failure.
- `withdrawals_v2_oldest_pending_seconds`: **gauge** — age of the oldest still-pending
  event. This is the alert signal: a value that keeps climbing means an event never
  reached a verdict and needs an operator.

While the process is running, every unresolved prove event stays counted in the
pending gauges until it reaches `valid` or `invalid` (P0). On steady-state restart,
the monitor replays at least the portal's on-chain proof-maturity delay plus a 24-hour
safety margin and reconstructs pending age from each event block's timestamp. This
prevents a restart from losing an event that could still be awaiting finalization or
resetting its age alert. Operators investigating an older unresolved event can use an
explicit `--start.block` backfill.

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
- **Steady-state** (default): omit `--start.block`. On every startup the monitor reads
  `proofMaturityDelaySeconds()` from the portal and replays that full wall-clock window
  plus 24 hours. `--lookback.blocks` (default 900) is an additional minimum; whichever
  start is earlier wins. Pending age is restored from the L1 event block timestamp.

## CLI Usage

### Command Structure

```bash
go run ../cmd/monitorism withdrawals-v2 [options]
```

### Available Options

```bash
OPTIONS:
   --l1.node.url value             Node URL of L1 archive+trace Geth node (must serve debug_traceTransaction) [$WITHDRAWALS_V2_MON_L1_NODE_URL]
   --start.block value             Starting L1 block number to scan (one-time backfill); omit for maturity-window steady-state replay [$WITHDRAWALS_V2_MON_START_BLOCK]
   --lookback.blocks value         Additional minimum block-count replay on startup (default: 900) [$WITHDRAWALS_V2_MON_LOOKBACK_BLOCKS]
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
