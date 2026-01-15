### Fault Monitor

The fault monitor checks for changes in output roots posted to the `L2OutputOracle` contract. On change, reconstructing the output root from a trusted L2 source and looking for a match

NOTE: Fault monitor only working against chains Pre-Faultproof. For chains using Faultproof system please check [dispute-mon service](https://github.com/ethereum-optimism/optimism/blob/develop/op-dispute-mon/README.md)

```
OPTIONS:
   --l1.node.url value             Node URL of L1 peer Geth node [$FAULT_MON_L1_NODE_URL]
   --l2.node.url value             Node URL of L2 peer Op-Geth node [$FAULT_MON_L2_NODE_URL]
   --start.output.index value      Output index to start from. -1 to find first unfinalized index (default: -1) [$FAULT_MON_START_OUTPUT_INDEX]
   --optimismportal.address value  Address of the OptimismPortal contract [$FAULT_MON_OPTIMISM_PORTAL]
   --l2oo.address value            Address of the L2OutputOracle contract (alternative to optimismportal.address) [$FAULT_MON_L2OO_ADDRESS]
```

On mismatch the `isCurrentlyMismatched` metrics is set to `1`.
