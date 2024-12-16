### Multisig Monitor

The multisig monitor reports the paused status of the `OptimismPortal` contract. If set, the latest nonce of the configured `Safe` address. And also if set, the latest presigned nonce stored in One Password. The latest presigned nonce is identified by looking for items in the configured vault that follow a `ready-<nonce>.json` name. The highest nonce of this item name format is reported.

- **NOTE**: In order to read from one password, the `OP_SERVICE_ACCOUNT_TOKEN` environment variable must be set granting the process permission to access the specified vault.

```
OPTIONS:
   --l1.node.url value             [$MULTISIG_MON_L1_NODE_URL]       Node URL of L1 peer (default: "127.0.0.1:8545")
   --optimismportal.address value  [$MULTISIG_MON_OPTIMISM_PORTAL]   Address of the OptimismPortal contract
   --nickname value                [$MULTISIG_MON_NICKNAME]          Nickname of chain being monitored
   --safe.address value            [$MULTISIG_MON_SAFE]              Address of the Safe contract
   --op.vault value                [$MULTISIG_MON_1PASS_VAULT_NAME]  1Pass Vault name storing presigned safe txs following a 'ready-<nonce>.json' item name format
```
