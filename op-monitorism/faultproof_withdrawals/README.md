# Purpose of the Service
faultproof_withdrawals has the following purpose:
- Monitor Withdrawals: The service listens for WithdrawalProven events on the OptimismPortal contract on L1.
- Validate Withdrawals: It verifies the validity of these withdrawals by checking the corresponding state on L2.
- Detect Forgeries: The service identifies and reports any invalid withdrawals or potential forgeries.

## Enable Metrics
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
## Available Metrics and Meaning


# Cli options

```bash 
go run ./cmd/monitorism faultproof_withdrawals --help
NAME:
   Monitorism faultproof_withdrawals - Monitors withdrawals on the OptimismPortal in order to detect forgery. Note: Requires chains with Fault Proofs.

USAGE:
   Monitorism faultproof_withdrawals [command options]

DESCRIPTION:
   Monitors withdrawals on the OptimismPortal in order to detect forgery. Note: Requires chains with Fault Proofs.

OPTIONS:
   --l1.geth.url value             L1 execution layer node URL [$FAULTPROOF_WITHDRAWAL_MON_L1_GETH_URL]
   --l2.node.url value             L2 rollup node consensus layer (op-node) URL [$FAULTPROOF_WITHDRAWAL_MON_L2_OP_NODE_URL]
   --l2.geth.url value             L2 OP Stack execution layer client(op-geth) URL [$FAULTPROOF_WITHDRAWAL_MON_L2_OP_GETH_URL]
   --event.block.range value       Max block range when scanning for events (default: 1000) [$FAULTPROOF_WITHDRAWAL_MON_EVENT_BLOCK_RANGE]
   --start.block.height value      Starting height to scan for events. This will take precedence if set. (default: 0) [$FAULTPROOF_WITHDRAWAL_MON_START_BLOCK_HEIGHT]
   --start.block.hours.ago value   How many hours in the past to start to check for forgery. Default will be 336 (14 days) days if not set. The real block to start from will be found within the hour precision. (default: 0) [$FAULTPROOF_WITHDRAWAL_MON_START_HOURS_IN_THE_PAST]
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

## Example run on sepolia op chain

```bash
L1_GETH_URL="https://..."
L2_OP_NODE_URL="https://..."
L2_OP_GETH_URL="https://..."

export MONITORISM_LOOP_INTERVAL_MSEC=100
export MONITORISM_METRICS_PORT=7300
export MONITORISM_METRICS_ENABLED=true
export FAULTPROOF_WITHDRAWAL_MON_L1_GETH_URL="$L1_GETH_URL"
export FAULTPROOF_WITHDRAWAL_MON_L2_OP_NODE_URL="$L2_OP_NODE_URL"
export FAULTPROOF_WITHDRAWAL_MON_L2_OP_GETH_URL="$L2_OP_GETH_URL"
export FAULTPROOF_WITHDRAWAL_MON_OPTIMISM_PORTAL="0x16Fc5058F25648194471939df75CF27A2fdC48BC"
export FAULTPROOF_WITHDRAWAL_MON_START_BLOCK_HEIGHT=5914813
export FAULTPROOF_WITHDRAWAL_MON_EVENT_BLOCK_RANGE=1000


go run ./cmd/monitorism faultproof_withdrawals
```

Metrics will be avialable at [http://localhost:7300](http://localhost:7300)