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
   --l2.node.url value             [DEPRECATED] L2 rollup node consensus layer (op-node) URL [$FAULTPROOF_WITHDRAWAL_MON_L2_OP_NODE_URL]
   --l2.geth.url value             L2 OP Stack execution layer client(op-geth) URL [$FAULTPROOF_WITHDRAWAL_MON_L2_OP_GETH_URL]
   --l2.geth.backup.urls value     Backup L2 OP Stack execution layer client URLs (format: name=url,name2=url2) [$FAULTPROOF_WITHDRAWAL_MON_L2_OP_GETH_BACKUP_URLS]
   --event.block.range value       Max block range when scanning for events (default: 1000) [$FAULTPROOF_WITHDRAWAL_MON_EVENT_BLOCK_RANGE]
   --start.block.height value      Fixed L1 height to start scanning for events from. Takes precedence over --start.block.hours.ago when set (>= 0); leave at the default -1 to start dynamically from --start.block.hours.ago instead. (default: -1) [$FAULTPROOF_WITHDRAWAL_MON_START_BLOCK_HEIGHT]
   --start.block.hours.ago value   How many hours before the current L1 tip to start scanning for forgeries, recomputed on each (re)start. Defaults to 672 (28 days) when unset. Ignored when --start.block.height is set. The starting block is resolved to within one-hour precision. (default: 0) [$FAULTPROOF_WITHDRAWAL_MON_START_HOURS_IN_THE_PAST]
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

## Choosing the start point

On startup the monitor needs an L1 block to begin scanning `WithdrawalProven` events from. There are two mutually exclusive ways to set it:

| Env var | Meaning | Default |
| --- | --- | --- |
| `FAULTPROOF_WITHDRAWAL_MON_START_HOURS_IN_THE_PAST` | Start `N` hours before the **current L1 tip**, recomputed on every (re)start. | `672` (28 days) |
| `FAULTPROOF_WITHDRAWAL_MON_START_BLOCK_HEIGHT` | Start from a **fixed** L1 block height. | `-1` (unset → use hours-ago) |

**Precedence:** if `START_BLOCK_HEIGHT` is set (`>= 0`) it wins and `START_HOURS_IN_THE_PAST` is ignored (the monitor logs a warning if both are set). Leave `START_BLOCK_HEIGHT` unset (`-1`) to use the dynamic window.

**Prefer the hours-ago window for production.** Validating a withdrawal requires an `eth_getProof` against the L2 node at the dispute game's L2 block. Pruned / non-archive L2 nodes (e.g. op-reth) only retain trie state for a recent window (~29 days), so a fixed `START_BLOCK_HEIGHT` older than that window makes every proof fail with `failed to get proof from any node` and the monitor stalls. The 28-day default keeps each restart inside that window. Only pin `START_BLOCK_HEIGHT` when the L2 node is a full **archive** node.

## Example run on sepolia op chain

```bash
L1_GETH_URL="https://..."
L2_OP_NODE_URL="https://..."  # [DEPRECATED] This URL is no longer required
L2_OP_GETH_URL="https://..."
L2_OP_GETH_BACKUP_URLS="backup1=https://...,backup2=https://..."

export MONITORISM_LOOP_INTERVAL_MSEC=100
export MONITORISM_METRICS_PORT=7300
export MONITORISM_METRICS_ENABLED=true
export FAULTPROOF_WITHDRAWAL_MON_L1_GETH_URL="$L1_GETH_URL"
export FAULTPROOF_WITHDRAWAL_MON_L2_OP_NODE_URL="$L2_OP_NODE_URL"
export FAULTPROOF_WITHDRAWAL_MON_L2_OP_GETH_URL="$L2_OP_GETH_URL"
export FAULTPROOF_WITHDRAWAL_MON_L2_OP_GETH_BACKUP_URLS="$L2_OP_GETH_BACKUP_URLS"
export FAULTPROOF_WITHDRAWAL_MON_OPTIMISM_PORTAL="0x16Fc5058F25648194471939df75CF27A2fdC48BC"
export FAULTPROOF_WITHDRAWAL_MON_EVENT_BLOCK_RANGE=1000
# Start 28 days before the current L1 tip (recomputed on each restart). See "Choosing the start point".
# Leave START_BLOCK_HEIGHT unset unless the L2 node is a full archive node.
export FAULTPROOF_WITHDRAWAL_MON_START_HOURS_IN_THE_PAST=672

go run ./cmd/monitorism faultproof_withdrawals
```

Metrics will be available at [http://localhost:7300](http://localhost:7300)
