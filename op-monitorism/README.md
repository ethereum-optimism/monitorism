# Monitors

`op-monitorism` is a collection of monitoring tools for the OP stack. Each monitor is designed to track a specific aspect of the Optimism stack and emit metrics that can be used to set up alerts.

The following the commands are currently available:

```bash
NAME:
   Monitorism - OP Stack Monitoring

USAGE:
   Monitorism [global options] command [command options]

VERSION:
   0.1.0-unstable

DESCRIPTION:
   OP Stack Monitoring

COMMANDS:
   multisig             Monitors OptimismPortal pause status, Safe nonce, and Pre-Signed nonce stored in 1Password
   fault                Monitors output roots posted on L1 against L2
   withdrawals          Monitors proven withdrawals on L1 against L2
   balances             Monitors account balances
   drippie              Monitors Drippie contract
   secrets              Monitors secrets revealed in the CheckSecrets dripcheck
   global_events        Monitors global events with YAML configuration
   liveness_expiration  Monitor the liveness expiration on Gnosis Safe.
   version              Show version
   help, h              Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --help, -h     show help
   --version, -v  print the version
```

Each _monitor_ has some common configuration, configurable both via cli or env with defaults.

```bash
OPTIONS:
   --log.level value           [$MONITORISM_LOG_LEVEL]           The lowest log level that will be output (default: INFO)
   --log.format value          [$MONITORISM_LOG_FORMAT]          Format the log output. Supported formats: 'text', 'terminal', 'logfmt', 'json', 'json-pretty', (default: text)
   --log.color                 [$MONITORISM_LOG_COLOR]           Color the log output if in terminal mode (default: false)
   --metrics.enabled           [$MONITORISM_METRICS_ENABLED]     Enable the metrics server (default: false)
   --metrics.addr value        [$MONITORISM_METRICS_ADDR]        Metrics listening address (default: "0.0.0.0")
   --metrics.port value        [$MONITORISM_METRICS_PORT]        Metrics listening port (default: 7300)
   --loop.interval.msec value  [$MONITORISM_LOOP_INTERVAL_MSEC]  Loop interval of the monitor in milliseconds (default: 60000)
```

### Global Events Monitor

![df2b94999628ce8eee98fb60f45667e54be9b13db82add6aa77888f355137329](https://github.com/ethereum-optimism/monitorism/assets/23560242/b8d36a0f-8a17-4e22-be5a-3e9f3586b3ab)

The Global Events Monitor is made for to taking YAML rules as configuration and monitoring the events that are emitted on the chain.

| `op-monitorism/global_events` | [README](https://github.com/ethereum-optimism/monitorism/blob/main/op-monitorism/global_events/README.md) |
| ----------------------------- | --------------------------------------------------------------------------------------------------------- |

### Liveness Expiration Monitor

![ab27497cea05fbd51b7b1c2ecde5bc69307ac0f27349f6bba4f3f21423116071](https://github.com/ethereum-optimism/monitorism/assets/23560242/af7a7e29-fff5-4df3-82f0-94c2f28fde84)

The Liveness Expiration Monitor is made for monitoring the liveness expiration on Safes.

| `op-monitorism/liveness_expiration` | [README](https://github.com/ethereum-optimism/monitorism/blob/main/op-monitorism/liveness_expiration/README.md) |
| ----------------------------------- | --------------------------------------------------------------------------------------------------------------- |

### Fault Monitor

The fault monitor checks for changes in output roots posted to the `L2OutputOracle` contract.
On change, reconstructing the output root from a trusted L2 source and looking for a match.

| `op-monitorism/fault` | [README](https://github.com/ethereum-optimism/monitorism/blob/main/op-monitorism/fault/README.md) |
| --------------------- | ------------------------------------------------------------------------------------------------- |

### Withdrawals Monitor

The withdrawals monitor checks for new withdrawals that have been proven to the `OptimismPortal` contract.
Each withdrawal is checked against the `L2ToL1MessagePasser` contract.

| `op-monitorism/withdrawals` | [README](https://github.com/ethereum-optimism/monitorism/blob/main/op-monitorism/withdrawals/README.md) |
| --------------------------- | ------------------------------------------------------------------------------------------------------- |

### Balances Monitor

The balances monitor simply emits a metric reporting the balances for the configured accounts.

| `op-monitorism/balances` | [README](https://github.com/ethereum-optimism/monitorism/blob/main/op-monitorism/balances/README.md) |
| ------------------------ | ---------------------------------------------------------------------------------------------------- |

### Multisig Monitor

The multisig monitor reports the paused status of the `OptimismPortal` contract.
If set, the latest nonce of the configued `Safe` address. And also if set, the latest presigned nonce stored in One Password.
The latest presigned nonce is identifyed by looking for items in the configued vault that follow a `ready-<nonce>.json` name.
The highest nonce of this item name format is reported.

| `op-monitorism/multisig` | [README](https://github.com/ethereum-optimism/monitorism/blob/main/op-monitorism/multisig/README.md) |
| ------------------------ | ---------------------------------------------------------------------------------------------------- |

### Drippie Monitor

The drippie monitor tracks the execution and executability of drips within a Drippie contract.

| `op-monitorism/drippie` | [README](https://github.com/ethereum-optimism/monitorism/blob/main/op-monitorism/multisig/README.md) |
| ----------------------- | ---------------------------------------------------------------------------------------------------- |

### Secrets Monitor

The secrets monitor takes a Drippie contract as a parameter and monitors for any drips within that contract that use the CheckSecrets dripcheck contract. CheckSecrets is a dripcheck that allows a drip to begin once a specific secret has been revealed (after a delay period) and cancels the drip if a second secret is revealed. It's important to monitor for these secrets being revealed as this could be a sign that the secret storage platform has been compromised and someone is attempting to exflitrate the ETH controlled by that drip.

| `op-monitorism/secrets` | [README](https://github.com/ethereum-optimism/monitorism/blob/main/op-monitorism/multisig/README.md) |
| ----------------------- | ---------------------------------------------------------------------------------------------------- |

## CLI and Docs

## Development

After cloning, please run `./bootstrap.sh` to set up the development environment correctly.

## Intro

The cli has the ability to spin up a monitor for varying activities, each emmitting metrics used to setup alerts.

```
COMMANDS:
   multisig     Monitors OptimismPortal pause status, Safe nonce, and Pre-Signed nonce stored in 1Password
   fault        Monitors output roots posted on L1 against L2
   withdrawals  Monitors proven withdrawals on L1 against L2
   balances     Monitors account balances
   secrets      Monitors secrets revealed in the CheckSecrets dripcheck
```

Each monitor has some common configuration, configurable both via cli or env with defaults.

```
OPTIONS:
   --log.level value           [$MONITORISM_LOG_LEVEL]           The lowest log level that will be output (default: INFO)
   --log.format value          [$MONITORISM_LOG_FORMAT]          Format the log output. Supported formats: 'text', 'terminal', 'logfmt', 'json', 'json-pretty', (default: text)
   --log.color                 [$MONITORISM_LOG_COLOR]           Color the log output if in terminal mode (default: false)
   --metrics.enabled           [$MONITORISM_METRICS_ENABLED]     Enable the metrics server (default: false)
   --metrics.addr value        [$MONITORISM_METRICS_ADDR]        Metrics listening address (default: "0.0.0.0")
   --metrics.port value        [$MONITORISM_METRICS_PORT]        Metrics listening port (default: 7300)
   --loop.interval.msec value  [$MONITORISM_LOOP_INTERVAL_MSEC]  Loop interval of the monitor in milliseconds (default: 60000)
```
