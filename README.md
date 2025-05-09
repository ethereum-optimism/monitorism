- [Monitorism](#monitorism)
  - [Docker Images](#docker-images)
  - [Monitors Components](#monitors-components)
    - [Global Events Monitor](#global-events-monitor)
    - [Liveness Expiration Monitor](#liveness-expiration-monitor)
    - [Withdrawals Monitor](#withdrawals-monitor)
    - [Balances Monitor](#balances-monitor)
    - [Fault Monitor](#fault-monitor)
    - [Multisig Monitor](#multisig-monitor)
    - [Drippie Monitor](#drippie-monitor)
    - [Secrets Monitor](#secrets-monitor)
    - [Transaction Monitor](#transaction-monitor)
    - [ETH Conservation Monitor](#eth-conservation-monitor)
    - [Faultproof Withdrawals](#faultproof-withdrawal)
  - [Defender Components](#defender-components)
    - [HTTP API PSP Executor Service](#http-api-psp-executor-service)
  - [CLI &amp; Docs](#cli--docs)
    - [Bootstrap](#bootstrap)
    - [Command line Options](#command-line-options)

# Monitorism

_Monitorism_ is a tooling suite that supports monitoring and active remediation actions for the OP Stack chain.

The suite is composed of two main components: `op-monitorism` and `op-defender`, that can be used together or separately and see below for more details.

## Docker images

### Op-Monitorism

Op-Monitorism Docker images are published with each release and build, ensuring you have access to the latest features and fixes.

The latest release version for linux/amd64 can be found in the [release notes](https://github.com/ethereum-optimism/monitorism/releases)

To pull the latest Docker image, run:

```bash
docker pull --platform linux/amd64 us-docker.pkg.dev/oplabs-tools-artifacts/images/op-monitorism:latest
```

To pull a specific release version

```bash
docker pull --platform linux/amd64 us-docker.pkg.dev/oplabs-tools-artifacts/images/op-monitorism:v0.0.4
```

Note: The --platform flag is necessary for Mac computers with ARM chips to ensure compatibility with the linux/amd64 architecture.

To build the Docker image locally, execute the following command:

```bash
docker build -t op-monitorism ./op-monitorism
```

Use "-t op-monitorism" to tag the image with the name op-monitorism for easier reference.

## Monitors Components

The `monitors` are passive security services that provide automated monitoring for the OP Stack.
There are components that are designed to make monitoring of the OP stack and alerting on specific events, that could be a sign of a security incident.

The list of all the monitors currently built into `op-monitorism` is below.

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

### Withdrawals Monitor

![6d5477f5585cb49ff2f8bd147c2e7037772de6a1dd128ce4331596b011ce6ea9](https://github.com/user-attachments/assets/ac5e0a61-b495-4254-b32a-86abf61f0dc1)

The withdrawals monitor checks for new withdrawals that have been proven to the `OptimismPortal` contract.
Each withdrawal is checked against the `L2ToL1MessagePasser` contract.

| `op-monitorism/withdrawals` | [README](https://github.com/ethereum-optimism/monitorism/blob/main/op-monitorism/withdrawals/README.md) |
| --------------------------- | ------------------------------------------------------------------------------------------------------- |

### Balances Monitor

![5cd47a6e0f2fb7d921001db9eea24bb62bb892615011d03f275e02a147823827](https://github.com/user-attachments/assets/44884a76-e06d-4f58-a21f-94c2275e9d8b)

The balances monitor simply emits a metric reporting the balances for the configured accounts.

| `op-monitorism/balances` | [README](https://github.com/ethereum-optimism/monitorism/blob/main/op-monitorism/balances/README.md) |
| ------------------------ | ---------------------------------------------------------------------------------------------------- |

### Fault Monitor

<img width="1696" alt="148f61f4600327b94b55be39ca42c58c797d70d7017dbb7d56dbefa208cc7164" src="https://github.com/user-attachments/assets/68ecfaa0-ee6d-46be-b760-a9eb8b232d65">

The fault monitor checks for changes in output roots posted to the `L2OutputOracle` contract.
On change, reconstructing the output root from a trusted L2 source and looking for a match.

| `op-monitorism/fault` | [README](https://github.com/ethereum-optimism/monitorism/blob/main/op-monitorism/fault/README.md) |
| --------------------- | ------------------------------------------------------------------------------------------------- |

### Multisig Monitor

![7dab260ee38122980274fee27b114c590405cff2e5a68e6090290ecb786b68f2](https://github.com/user-attachments/assets/0eeb161b-923a-40fd-b561-468df3d5091d)

The multisig monitor reports the paused status of the `OptimismPortal` contract.
If set, the latest nonce of the configured `Safe` address. And also if set, the latest presigned nonce stored in One Password.
The latest presigned nonce is identified by looking for items in the configured vault that follow a `ready-<nonce>.json` name.
The highest nonce of this item name format is reported.

| `op-monitorism/multisig` | [README](https://github.com/ethereum-optimism/monitorism/blob/main/op-monitorism/multisig/README.md) |
| ------------------------ | ---------------------------------------------------------------------------------------------------- |

### Drippie Monitor

The drippie monitor tracks the execution and executability of drips within a Drippie contract.

| `op-monitorism/drippie` | [README](https://github.com/ethereum-optimism/monitorism/blob/main/op-monitorism/drippie/README.md) |
| ----------------------- | --------------------------------------------------------------------------------------------------- |

### Secrets Monitor

The secrets monitor takes a Drippie contract as a parameter and monitors for any drips within that contract that use the CheckSecrets dripcheck contract. CheckSecrets is a dripcheck that allows a drip to begin once a specific secret has been revealed (after a delay period) and cancels the drip if a second secret is revealed. It's important to monitor for these secrets being revealed as this could be a sign that the secret storage platform has been compromised and someone is attempting to exfiltrate the ETH controlled by that drip.

| `op-monitorism/secrets` | [README](https://github.com/ethereum-optimism/monitorism/blob/main/op-monitorism/secrets/README.md) |
| ----------------------- | --------------------------------------------------------------------------------------------------- |

### Transaction Monitor

The transaction monitor takes in a yaml config in order to run, and monitors transaction sent by a specific address, tracking both cumulative eth sent, as well as tunable thresholds for specific alerts. It is also configurable to support working against factory contracts, right now just the `FaultDisputeGame` factory to ensure the addresses are only interacting with valid fault dispute games.

| `op-monitorism/transaction_monitor` | [README](https://github.com/ethereum-optimism/monitorism/blob/main/op-monitorism/transaction_monitor/README.md) |
| ----------------------------------- | --------------------------------------------------------------------------------------------------------------- |

### ETH Conservation Monitor

The ETH conservation monitor traces L2 blocks and asserts that they adhere to the L2 ETH conservation invariant.

| `op-monitorism/conservation_monitor` | [README](https://github.com/ethereum-optimism/monitorism/blob/main/op-monitorism/conservation_monitor/README.md) |
| ----------------------------------- | --------------------------------------------------------------------------------------------------------------- |

### Faultproof Withdrawal

The Faultproof Withdrawal component monitors ProvenWithdrawals events on the [OptimismPortal](https://github.com/ethereum-optimism/superchain-registry/blob/d454618b6cf885417aa8cc8c760bd9ed0429c131/superchain/configs/mainnet/op.toml#L50) contract and performs checks to detect any violations of invariant conditions on the chain. If a violation is detected, it logs the issue and sets a Prometheus metric for the event.

This component is designed to work exclusively with chains that are already utilizing the [Fault Proofs system](https://docs.optimism.io/stack/protocol/fault-proofs/explainer).
This is a new version of the deprecated [chain-mon faultproof-wd-mon](https://github.com/ethereum-optimism/optimism/tree/chain-mon/v1.2.1/packages/chain-mon/src/faultproof-wd-mon).
For detailed information on how the component works and the algorithms used, please refer to the component README.

| `op-monitorism/faultproof_withdrawals` | [README](https://github.com/ethereum-optimism/monitorism/blob/main/op-monitorism/faultproof_withdrawals/README.md) |
| -------------------------------------- | ------------------------------------------------------------------------------------------------------------------ |

## Defender Components

The _defenders_ are active security service allowing to provide automated defense for the OP Stack.
There are components that are designed to make immediate actions onchain/offchain to protect the assets.

The list of all the defender currently built into `op-defender` is below.

### HTTP API PSP Executor Service

![f112841bad84c59ea3ed1ca380740f5694f553de8755b96b1a40ece4d1c26f81](https://github.com/user-attachments/assets/17235e99-bf25-40a5-af2c-a0d9990c6276)

The PSP Executor Service is made for executing PSP onchain faster to increase our readiness and speed in case of incident response.

| `op-defender/psp_executor` | [README](https://github.com/ethereum-optimism/monitorism/blob/main/op-defender/psp_executor/README.md) |
| -------------------------- | ------------------------------------------------------------------------------------------------------ |

## CLI & Docs

### Bootstrap

After cloning, please run `./bootstrap.sh` to set up the development environment correctly.

### Command line Options

The cli has the ability to spin up a monitor for varying activities, each emitting metrics used to setup alerts.

```
COMMANDS:
   multisig                Monitors OptimismPortal pause status, Safe nonce, and Pre-Signed nonce stored in 1Password
   fault                   Monitors output roots posted on L1 against L2
   withdrawals             Monitors proven withdrawals on L1 against L2
   balances                Monitors account balances
   drippie                 Monitors Drippie contract
   secrets                 Monitors secrets revealed in the CheckSecrets dripcheck
   global_events           Monitors global events with YAML configuration
   liveness_expiration     Monitor the liveness expiration on Gnosis Safe.
   faultproof_withdrawals  Monitors withdrawals on the OptimismPortal in order to detect forgery. Note: Requires chains with Fault Proofs.
   version                 Show version
   help, h                 Shows a list of commands or help for one command
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

