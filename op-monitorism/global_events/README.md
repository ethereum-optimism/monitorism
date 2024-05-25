## Global Events monitoring

This monitoring modules for the yaml rules added with the format ⚠️ This readme will be move into the global readme in the future.

```yaml
# This watches all contacts for OP, Mode, and Base mainnets for two logs.
# These logs are emitted by Safes, so this effectively watches for all
# transactions from any Safe on these chains.

version: 1.X
name: My Super Explicit Alert Name.
priority: P0
addresses:
  - 0x000000000000000000000000000000000000CAFE
events:
  - signature: FunctionYouWantToMonitore(bytes32,uint256)
  - signature: FunctionYouWantToMonitore(bytes32,uint256)
```

To run it:

```bash
go run . global_events --nickname MySuperNickName --l1.node.url https://localhost:8545  --PathYamlRules ../rules --loop.interval.msec 12000
```
