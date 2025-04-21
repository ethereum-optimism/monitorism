# ETH Conservation Monitor

A service that monitors OP Stack blocks to ensure that no ETH is minted or burned (except for the special cases of
deposits with a non-zero mint value.)

## ETH Conservation Invariant

Fee distribution on the OP Stack is altered from L1, and all fees are sent to [vaults][fee-vault]. There is currently no
in-protocol route to burn ETH on the OP Stack other than `SELFDESTRUCT`. ETH can _only_ be minted on L2 by deposit
transactions.

The invariant to be checked by this monitoring service is:

$$
\displaylines{
    TB(b) = \text{balance of account all touched accounts in block \`b' at block \`b', including fee vaults}
    \\
    TDM(b) = \text{amount of ETH minted by deposit transactions in block \`b'}
    \\
    TB(b-1) \ge TB(b) - TDM(b)
}
$$

## Features

- Monitor chains for the ETH conservation invariant.
- Prometheus metrics for monitoring and alerting

## Metrics

The service exports the following Prometheus metrics:

- `conservation_mon_invariant_held`: Total number blocks that the invariant has held for.
- `conservation_mon_invariant_violations`: Total number of blocks that the invariant has been broken within.

## Usage

```bash
monitorism conservation_monitor \
  --node.url <l2_el_url>
```

_NOTE_: `l2_el_url` must have `debug_traceBlockByHash` exposed.

[fee-vault]: https://specs.optimism.io/protocol/exec-engine.html?highlight=vault#fee-vaults
