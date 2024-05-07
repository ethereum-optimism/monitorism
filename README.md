# Monitorism
A blockchain surveillance tool that supports monitoring for the OP Stack and EVM-compatible chains.

⚠️  Caution: *Monitorism* is currently in its beta phase and is under active migration 🔨. This implies that *Monitorism* is presently not fully stable. ⚠️

The cli has the ability to spin up a monitor for varying activities, each emmitting metrics used to setup alerts.
```
COMMANDS:
   multisig     Monitors OptimismPortal pause status, Safe nonce, and Pre-Signed nonce stored in 1Password
   fault        Monitors output roots posted on L1 against L2
   withdrawals  Monitors proven withdrawals on L1 against L2
   balances     Monitors account balances
```

Each monitor has some common configuration, configurable both via cli arguments or environment variables and defaults.
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

## Monitors

In addition the common configuration, each available monitor also accept their specific configuration

* **Note**: The environment variable prefix for monitor-configuration is different than the global monitor config described above.

### Fault Monitor

The fault monitor checks for changes in output roots posted to the `L2OutputOracle` contract. On change, reconstructing the output root from a trusted L2 source and looking for a match
```
OPTIONS:
   --l1.node.url value             [$FAULT_MON_L1_NODE_URL]         Node URL of L1 peer (default: "127.0.0.1:8545")                              
   --l2.node.url value             [$FAULT_MON_L2_NODE_URL]         Node URL of L2 peer (default: "127.0.0.1:9545")                              
   --start.output.index value      [$FAULT_MON_START_OUTPUT_INDEX]  Output index to start from. -1 to find first unfinalized index (default: -1) 
   --optimismportal.address value  [$FAULT_MON_OPTIMISM_PORTAL]     Address of the OptimismPortal contract                                       
```

On mismatch the `isCurrentlyMismatched` metrics is set to `1`.

### Withdrawals Monitor

The withdrawals monitor checks for new withdrawals that haven proven to the `OptimismPortal` contract. Each withdrawal is checked against the `L2ToL1MessagePasser` contract
```
OPTIONS:
   --l1.node.url value             [$WITHDRAWAL_MON_L1_NODE_URL]         Node URL of L1 peer (default: "127.0.0.1:8545")          
   --l2.node.url value             [$WITHDRAWAL_MON_L2_NODE_URL]         Node URL of L2 peer (default: "127.0.0.1:9545")          
   --event.block.range value       [$WITHDRAWAL_MON_EVENT_BLOCK_RANGE]   Max block range when scanning for events (default: 1000) 
   --start.block.height value      [$WITHDRAWAL_MON_START_BLOCK_HEIGHT]  Starting height to scan for events                       
   --optimismportal.address value  [$WITHDRAWAL_MON_OPTIMISM_PORTAL]     Address of the OptimismPortal contract                   
```

If a proven withdrawal is missing from L2, the `isDetectingForgeries` metrics is set to `1`.

### Balances Monitor

The balances monitor simply emits a metric reporting the balances for the configured accounts.
```
OPTIONS:
   --node.url value                                             [$BALANCE_MON_NODE_URL]  Node URL of a peer (default: "127.0.0.1:8545") 
   --accounts address:nickname [ --accounts address:nickname ]  [$BALANCE_MON_ACCOUNTS]  One or accounts formatted via address:nickname 
```

### Multisig Monitor

The multisig monitor reports the paused status of the `OptimismPortal` contract. If set, the latest nonce of the configued `Safe` address. And also if set, the latest presigned nonce stored in One Password. The latest presigned nonce is identifyed by looking for items in the configued vault that follow a `ready-<nonce>.json` name. The highest nonce of this item name format is reported.

* **NOTE**: In order to read from one password, the `OP_SERVICE_ACCOUNT_TOKEN` environment variable must be set granting the process permission to access the specified vault.

```
OPTIONS:
   --l1.node.url value             Node URL of L1 peer (default: "127.0.0.1:8545")                                               [$MULTISIG_MON_L1_NODE_URL]
   --optimismportal.address value  Address of the OptimismPortal contract                                                        [$MULTISIG_MON_OPTIMISM_PORTAL]
   --nickname value                Nickname of chain being monitored                                                             [$MULTISIG_MON_NICKNAME]
   --safe.address value            Address of the Safe contract                                                                  [$MULTISIG_MON_SAFE]
   --op.vault value                1Pass Vault name storing presigned safe txs following a 'ready-<nonce>.json' item name format [$MULTISIG_MON_1PASS_VAULT_NAME]
```

## Typescript
Locate into the folder `ts/`.

### chain-mon (ts)

[![codecov](https://codecov.io/gh/ethereum-optimism/optimism/branch/develop/graph/badge.svg?token=0VTG7PG7YR&flag=chain-mon-tests)](https://codecov.io/gh/ethereum-optimism/optimism)

`chain-mon` is a collection of chain monitoring services.

## Installation

Clone, install, and build the Optimism monorepo:

```
git clone https://github.com/ethereum-optimism/optimism.git
pnpm install
pnpm build
```

## Running a service

Copy `.env.example` into a new file named `.env`, then set the environment variables listed there depending on the service you want to run.
Once your environment variables have been set, run via:

```
pnpm start:<service name>
```

For example, to run `drippie-mon`, execute:

```
pnpm start:drippie-mon
```


## Golang
Locate into the folder `go/`
For now empty.
