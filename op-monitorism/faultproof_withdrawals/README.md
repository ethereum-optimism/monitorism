# Purpose of the Service
faultproof_withdrawals has the following purpose:
- Monitor Withdrawals: The service listens for WithdrawalProven events on the OptimismPortal contract on L1.
- Validate Withdrawals: It verifies the validity of these withdrawals by checking the corresponding state on L2.
- Detect Forgeries: The service identifies and reports any invalid withdrawals or potential forgeries.

# Prometheus Metrics
This service will optionally expose a [prometeus metrics](https://prometheus.io/docs/concepts/metric_types/).

In order to start the metrics service make sure to either export the variables or setup the right cli args

```bash
export MONITORISM_METRICS_PORT=7300
export MONITORISM_METRICS_ENABLED=true

cd ../
go run ./cmd/monitorism faultproof_withdrawals
```
or 

```bash
cd ../
go run ./cmd/monitorism faultproof_withdrawals --metrics.enabled --metrics.port 7300
```

# Cli options

```bash 
NAME:
   Monitorism faultproof_withdrawals - Monitors proven withdrawals on L1 against L2, for FaultProof compatible chains

USAGE:
   Monitorism faultproof_withdrawals [command options]

DESCRIPTION:
   Monitors proven withdrawals on L1 against L2, for FaultProof compatible chains

OPTIONS:
   --l1.geth.url value             Node URL of L1 peer (default: "127.0.0.1:8545") [$FAULTPROOF_WITHDRAWAL_MON_L1_GETH_URL]
   --l2.node.url value             Node URL of L2 peer (default: "127.0.0.1:9545") [$FAULTPROOF_WITHDRAWAL_MON_L2_OP_NODE_URL]
   --l2.geth.url value             Node URL of L2 peer (default: "127.0.0.1:9546") [$FAULTPROOF_WITHDRAWAL_MON_L2_OP_GETH_URL]
   --event.block.range value       Max block range when scanning for events (default: 1000) [$FAULTPROOF_WITHDRAWAL_MON_EVENT_BLOCK_RANGE]
   --start.block.height value      Starting height to scan for events (default: 0) [$FAULTPROOF_WITHDRAWAL_MON_START_BLOCK_HEIGHT]
   --optimismportal.address value  Address of the OptimismPortal contract [$FAULTPROOF_WITHDRAWAL_MON_OPTIMISM_PORTAL]
   --log.level value               The lowest log level that will be output (default: INFO) [$MONITORISM_LOG_LEVEL]
   --log.format value              Format the log output. Supported formats: 'text', 'terminal', 'logfmt', 'json', 'json-pretty', (default: text) [$MONITORISM_LOG_FORMAT]
   --log.color                     Color the log output if in terminal mode (default: false) [$MONITORISM_LOG_COLOR]
   --log.pid                       Show pid in the log (default: false) [$MONITORISM_LOG_PID]
   --metrics.enabled               Enable the metrics server (default: false) [$MONITORISM_METRICS_ENABLED]
   --metrics.addr value            Metrics listening address (default: "0.0.0.0") [$MONITORISM_METRICS_ADDR]
   --metrics.port value            Metrics listening port (default: 7300) [$MONITORISM_METRICS_PORT]
   --loop.interval.msec value      Loop interval of the monitor in milliseconds (default: 60000) [$MONITORISM_LOOP_INTERVAL_MSEC]
   --help, -h                      show help
   ```