# Monitorism

A blockchain surveillance tool that supports monitoring for the OP Stack and EVM-compatible chains.

## Monitors

In addition the common configuration, each monitor also has their specific configuration

- **Note**: The environment variable prefix for monitor-specific configuration is different than the global monitor config described above.

### Fault Monitor

The fault monitor checks for changes in output roots posted to the `L2OutputOracle` contract. On change, reconstructing the output root from a trusted L2 source and looking for a match
`op-monitorism/fault` -> https://github.com/ethereum-optimism/monitorism/blob/aba37ff58b3503018ae5dbad5e5473af670834bc/op-monitorism/fault/README.md#L1

### Withdrawals Monitor

The withdrawals monitor checks for new withdrawals that have been proven to the `OptimismPortal` contract. Each withdrawal is checked against the `L2ToL1MessagePasser` contract
`op-monitorism/withdrawals` -> https://github.com/ethereum-optimism/monitorism/blob/aba37ff58b3503018ae5dbad5e5473af670834bc/op-monitorism/withdrawals/README.md#L1

### Balances Monitor

The balances monitor simply emits a metric reporting the balances for the configured accounts.
`op-monitorism/balances` -> https://github.com/ethereum-optimism/monitorism/blob/aba37ff58b3503018ae5dbad5e5473af670834bc/op-monitorism/balances/README.md#L1

### Multisig Monitor

The multisig monitor reports the paused status of the `OptimismPortal` contract. If set, the latest nonce of the configued `Safe` address. And also if set, the latest presigned nonce stored in One Password. The latest presigned nonce is identifyed by looking for items in the configued vault that follow a `ready-<nonce>.json` name. The highest nonce of this item name format is reported.

`op-monitorism/multisig` -> https://github.com/ethereum-optimism/monitorism/blob/aba37ff58b3503018ae5dbad5e5473af670834bc/op-monitorism/multisig/README.md#L1

### Drippie Monitor

The drippie monitor tracks the execution and executability of drips within a Drippie contract.
`op-monitorism/drippie` -> https://github.com/ethereum-optimism/monitorism/blob/aba37ff58b3503018ae5dbad5e5473af670834bc/op-monitorism/drippie/README.md#L1

### Secrets Monitor

The secrets monitor takes a Drippie contract as a parameter and monitors for any drips within that contract that use the CheckSecrets dripcheck contract. CheckSecrets is a dripcheck that allows a drip to begin once a specific secret has been revealed (after a delay period) and cancels the drip if a second secret is revealed. It's important to monitor for these secrets being revealed as this could be a sign that the secret storage platform has been compromised and someone is attempting to exflitrate the ETH controlled by that drip.
`op-monitorism/secrets` -> https://github.com/ethereum-optimism/monitorism/blob/aba37ff58b3503018ae5dbad5e5473af670834bc/op-monitorism/secrets/README.md#L1

### Global Events Monitor

The Gloval Events Monitor is made for to taking YAML rules as configuration and monitoring the events that are emitted on the chain.
`op-monitorism/global_events` -> https://github.com/ethereum-optimism/monitorism/blob/aba37ff58b3503018ae5dbad5e5473af670834bc/op-monitorism/global_events/README.md#L1

### Liveness Expiration Monitor

The Liveness Expiration Monitor is made for monitoring the liveness expiration on Safes.
`op-monitorism/liveness_expiration` -> https://github.com/ethereum-optimism/monitorism/blob/aba37ff58b3503018ae5dbad5e5473af670834bc/op-monitorism/liveness_expiration/README.md#L1

## CLI and Docs

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
