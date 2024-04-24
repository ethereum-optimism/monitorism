## Monitorism
*Monitorism* is an blockchain surveillance tooling that supports monitoring for the OP Stack and EVM-compatible chains.


⚠️ Caution: *Monitorism* is currently in its beta phase and is under active migration 🔨. This implies that *Monitorism* is presently not fully stable. ⚠️


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
