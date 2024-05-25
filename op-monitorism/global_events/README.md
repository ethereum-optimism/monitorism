## Global Events monitoring

This monitoring modules for the yaml rules added with the format ⚠️ This readme will be move into the global readme in the future.

The rules are located here: `op-monitorism/global_events/rules/` then we have multiples folders depending the networks you want to monitore (`mainnet` or `sepolia`) for now.
```yaml
# This is a TEMPLATE file please copy this one
# This watches all contacts for OP, Mode, and Base mainnets for two logs.
version: 1.0
name: Template SafeExecution Events (Success/Failure) L1 # Please put the L1 or L2 at the end of the name.
priority: P5 # This is a test, so it is a P5
#If addresses is empty like below it will watch all addresses otherwise you can address specific addresses.
addresses:
  # - 0xbEb5Fc579115071764c7423A4f12eDde41f106Ed # Specific Addresses /!\ We are not supporting EIP 3770 yet, if the address is not starting by 0x, this will panic by safety measure.
events:
  - signature: ExecutionFailure(bytes32,uint256) # List of the events to watch for the addresses.
  - signature: ExecutionSuccess(bytes32,uint256) # List of the events to watch for the addresses.
```

To run it:

```bash
go run . global_events --nickname MySuperNickName --l1.node.url https://localhost:8545  --PathYamlRules ../rules --loop.interval.msec 12000
```
