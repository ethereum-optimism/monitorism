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
- `withdrawals_v2_invalid_withdrawals_total{reason,txhash,wdhash}`: **P0** — the portal
  accepted a proof a correct verifier rejects. `reason` is one of:
  - `bad_withdrawal_proof`: storage-inclusion proof does not prove inclusion.
  - `bad_output_root_binding`: `keccak(outputRootProof)` does not equal the game root claim.
- `withdrawals_v2_unverifiable_withdrawals_total{reason,txhash,wdhash}`: **blind spot**
  (not P0) — the monitor could not re-verify and needs operator attention. `reason` is
  one of `trace_unavailable`, `no_prove_call_found`, `decode_error`, `game_read_error`.

## Requirements

- **L1 archive + trace node**: the node must serve `debug_traceTransaction` for the
  blocks being scanned. A pruned full node only retains ~128 blocks of state, so it
  can only trace withdrawals proven in the last ~25 minutes; scanning older blocks
  (or backfilling) requires a true archive (`--gcmode=archive --state.scheme=hash`).
  When a trace is unavailable the monitor emits `unverifiable_withdrawals_total`
  rather than stalling.

## CLI Usage

### Command Structure

```bash
go run ../cmd/monitorism withdrawals-v2 [options]
```

### Available Options

```bash
OPTIONS:
   --l1.node.url value             Node URL of L1 archive+trace Geth node (must serve debug_traceTransaction) [$WITHDRAWALS_V2_MON_L1_NODE_URL]
   --start.block value             Starting L1 block number to scan [$WITHDRAWALS_V2_MON_START_BLOCK]
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
