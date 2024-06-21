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
