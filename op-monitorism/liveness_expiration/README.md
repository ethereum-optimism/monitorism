# Liveness Expiration Monitoring

This Liveness expiration module is a monitoring dedicated for the safes of the Optimism network.
Ensuring that owners that operates the safes and performs any important actions are still actives enough.

![ab27497cea05fbd51b7b1c2ecde5bc69307ac0f27349f6bba4f3f21423116071](https://github.com/ethereum-optimism/monitorism/assets/23560242/af7a7e29-fff5-4df3-82f0-94c2f28fde84)

## CLI and Docs:

```bash
NAME:
   Monitorism liveness_expiration - Monitor the liveness expiration on Gnosis Safe.

USAGE:
   Monitorism liveness_expiration [command options] [arguments...]

DESCRIPTION:
   Monitor the liveness expiration on Gnosis Safe.

OPTIONS:
   --l1.node.url value             Node URL of L1 peer (default: "127.0.0.1:8545") [$LIVENESS_EXPIRATION_MON_L1_NODE_URL]
   --start.block.height value      Starting height to scan for events (still not implemented for now.. The monitoring will start at the last block number) (default: 0) [$LIVENESS_EXPIRATION_MON_START_BLOCK_HEIGHT]
   --livenessmodule.address value  Address of the LivenessModuleAddress contract [$LIVENESS_EXPIRATION_MON_LIVENESS_MODULE_ADDRESS]
   --livenessguard.address value   Address of the LivenessGuardAddress contract [$LIVENESS_EXPIRATION_MON_LIVENESS_GUARD_ADDRESS]
   --safe.address value            Address of the safe contract [$LIVENESS_EXPIRATION_MON_SAFE_ADDRESS]
   --log.level value               The lowest log level that will be output (default: INFO) [$MONITORISM_LOG_LEVEL]
   --log.format value              Format the log output. Supported formats: 'text', 'terminal', 'logfmt', 'json', 'json-pretty', (default: text) [$MONITORISM_LOG_FORMAT]
   --log.color                     Color the log output if in terminal mode (default: false) [$MONITORISM_LOG_COLOR]
   --metrics.enabled               Enable the metrics server (default: false) [$MONITORISM_METRICS_ENABLED]
   --metrics.addr value            Metrics listening address (default: "0.0.0.0") [$MONITORISM_METRICS_ADDR]
   --metrics.port value            Metrics listening port (default: 7300) [$MONITORISM_METRICS_PORT]
   --loop.interval.msec value      Loop interval of the monitor in milliseconds (default: 60000) [$MONITORISM_LOOP_INTERVAL_MSEC]
   --help, -h                      show help
```

### Informations

This tools allows the monitoring of multiples metrics like:

`blockTimestamp`: The block Timestamp of the latest block number on L1.
`highestBlockNumber`: The lastest block number height on L1.
`lastLiveOfAOwner`: Get the last activities for a given safe owner on L1.
`intervalLiveness`: the interval (in seconds) from the LivenessModule on L1.

The logic for the rules detection is not inside the binary `liveness_expiration` as this is integrated with prometheus. The rules are located in the Prometheus/Grafana side.

### Execution

To execute with a oneliner:

```bash
go run ../cmd/monitorism liveness_expiration --safe.address 0xc2819DC788505Aac350142A7A707BF9D03E3Bd03 --l1.node.url https://MySuperRPC --loop.interval.msec 12000 --livenessmodule.address 0x0454092516c9A4d636d3CAfA1e82161376C8a748 --livenessguard.address 0x24424336F04440b1c28685a38303aC33C9D14a25
```

Otherwise create an `.env` file with the environment variables present into the _help section_.
This is useful to run without any CLI arguments.

_Example_:

```bash
LIVENESS_EXPIRATION_MON_SAFE_ADDRESS=0xc2819DC788505Aac350142A7A707BF9D03E3Bd03
LIVENESS_EXPIRATION_MON_LIVENESS_MODULE_ADDRESS=0x0454092516c9A4d636d3CAfA1e82161376C8a748
LIVENESS_EXPIRATION_MON_LIVENESS_GUARD_ADDRESS=0x24424336F04440b1c28685a38303aC33C9D14a25
```
