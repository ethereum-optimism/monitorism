# Defenders

_op-defender_ is an active security service allowing to provide automated defense for the OP Stack.

The following the commands are currently available:

```bash
NAME:
   Defender - OP Stack Automated Defense

USAGE:
   Defender [global options] command [command options]

VERSION:
   0.1.0-unstable

DESCRIPTION:
   OP Stack Automated Defense

COMMANDS:
   psp_executor  Service to execute PSPs through API.
   version       Show version
   help, h       Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --help, -h     show help
   --version, -v  print the version
```

Each _defender_ has some common configuration, configurable both via cli or env with defaults.

```bash
OPTIONS:
   --rpc-url value             Node URL of a peer (default: "http://127.0.0.1:8545") [$PSPEXECUTOR_MON_NODE_URL]
   --privatekey value          Private key of the account that will issue the pause () [$PSPEXECUTOR_MON_PRIVATE_KEY]
   --receiver.address value    The receiver address of the pause request. [$PSPEXECUTOR_MON_RECEIVER_ADDRESS]
   --port.api value            Port of the API server you want to listen on (e.g. 8080). (default: "8080") [$PSPEXECUTOR_MON_PORT_API]
   --data value                calldata to execute the pause on mainnet with the signatures. [$PSPEXECUTOR_MON_CALLDATA]
   --log.level value           The lowest log level that will be output (default: INFO) [$MONITORISM_LOG_LEVEL]
   --log.format value          Format the log output. Supported formats: 'text', 'terminal', 'logfmt', 'json', 'json-pretty', (default: text) [$MONITORISM_LOG_FORMAT]
   --log.color                 Color the log output if in terminal mode (default: false) [$MONITORISM_LOG_COLOR]
   --metrics.enabled           Enable the metrics server (default: false) [$MONITORISM_METRICS_ENABLED]
   --metrics.addr value        Metrics listening address (default: "0.0.0.0") [$MONITORISM_METRICS_ADDR]
   --metrics.port value        Metrics listening port (default: 7300) [$MONITORISM_METRICS_PORT]
   --loop.interval.msec value  Loop interval of the monitor in milliseconds (default: 60000) [$MONITORISM_LOOP_INTERVAL_MSEC]
   --help, -h                  show help


```

### PSP Executor Service

![f112841bad84c59ea3ed1ca380740f5694f553de8755b96b1a40ece4d1c26f81](https://github.com/user-attachments/assets/17235e99-bf25-40a5-af2c-a0d9990c6276)

The PSP Executor Service is made for executing PSP onchain faster to increase our readiness and speed in case of incident response.

| `op-defender/psp_executor` | [README](https://github.com/ethereum-optimism/monitorism/blob/main/op-defender/psp_executor/README.md) |
| -------------------------- | ------------------------------------------------------------------------------------------------------ |
