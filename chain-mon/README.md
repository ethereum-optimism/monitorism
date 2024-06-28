# @eth-optimism/chain-mon

`chain-mon` is a collection of chain monitoring services.

## Installation

Clone, install, and build the Monitorism repo:

```
git clone https://github.com/ethereum-optimism/monitorism.git
cd chain-mon
pnpm install
pnpm build
```

## Running a service

Copy `.env.example` into a new file named `.env`, then set the environment variables listed there depending on the service you want to run.
Once your environment variables have been set, run via:

```
pnpm start:<service name>
```

For example, to run `balance-mon`, execute:

```
pnpm start:balance-mon
```

## Deploy Config

This folder contains deployment configuration files that were originally located in the [ethereum-optimism/optimism repository](https://github.com/ethereum-optimism/optimism/tree/develop/packages/contracts-bedrock/deploy-config).

### Background

The `deploy-config` folder has been added to this repository after moving the `chain-mon` package from the [ethereum-optimism/optimism repository](https://github.com/ethereum-optimism/optimism) to the [ethereum-optimism/monitorism repository](https://github.com/ethereum-optimism/monitorism).

### Updates and Upgrades

Please note that these configuration files may need to be upgraded to the version you need to use. Ensure you check for any updates or required changes to align with the latest versions and requirements of your deployment environment.
