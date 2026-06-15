# Withdrawals V2 Monitor

The V2 version of the withdrawals monitor is a simple monitor that validates withdrawals made through the OptimismPortal2 contract.

## Withdrawal Validation Process

1. **Event Monitoring**: Scans L1 blocks for `WithdrawalProvenExtension1` events from OptimismPortal2
2. **L2 State Verification**: Checks if the withdrawal hash exists in the L2ToL1MessagePasser contract on L2
3. **Failure Analysis**: For invalid withdrawals, performs additional analysis to categorize the failure

### Failure Classification

The monitor categorizes invalid withdrawals into two types:

- **`bad_output_root` (P3 - Acceptable)**: The dispute game has an incorrect output root. This is expected behavior when invalid dispute games are created.
- **`bad_withdrawal_proof` (P0 - Critical)**: The dispute game output root is correct, but the withdrawal proof itself is invalid. This indicates a serious security issue that needs immediate attention.

### Metrics

- `withdrawals_v2_valid_withdrawals_total`: Counter of valid withdrawals processed
- `withdrawals_v2_invalid_withdrawals_total`: Counter of invalid withdrawals, labeled with failure reason, transaction hash, and withdrawal hash

## CLI Usage

### Command Structure

```bash
go run ../cmd/monitorism withdrawals-v2 [options]
```

### Available Options

```bash
OPTIONS:
   --l1.node.url value             Node URL of L1 peer Geth node [$WITHDRAWALS_V2_MON_L1_NODE_URL]
   --l2.node.url value             Node URL of L2 peer Op-Geth node [$WITHDRAWALS_V2_MON_L2_NODE_URL]
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
# Monitor OP Mainnet withdrawals starting from block 18000000
go run ../cmd/monitorism withdrawals-v2 \
  --l1.node.url "https://mainnet.infura.io/v3/YOUR_API_KEY" \
  --l2.node.url "https://mainnet.optimism.io" \
  --start.block 18000000 \
  --optimism.portal.address "0xbEb5Fc579115071764c7423A4f12eDde41f106Ed" \
  --poll.interval 12s \
  --metrics.enabled \
  --metrics.port 7301
```
