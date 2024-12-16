# Purpose of the Service
`Withdrawals` has the following purpose:
- Monitor Withdrawals: The service listens for WithdrawalProven events on the OptimismPortal contract on L1.
- Validate Withdrawals: It verifies the validity of these withdrawals by checking the corresponding state on L2.
- Detect Forgeries: The service identifies and reports any invalid withdrawals or potential forgeries.

NOTE: The withdrawal monitor is only working against chains that are pre-Faultproof. For chains using the Faultproof system, please check the [faultproof_withdrawals service](https://github.com/ethereum-optimism/monitorism/blob/main/op-monitorism/faultproof_withdrawals/README.md).

```bash
OPTIONS:
   --l1.node.url value             Node URL of L1 peer Geth node [$WITHDRAWAL_MON_L1_NODE_URL]
   --l2.node.url value             Node URL of L2 peer Op-Geth node [$WITHDRAWAL_MON_L2_NODE_URL]
   --event.block.range value       Max block range when scanning for events (default: 1000) [$WITHDRAWAL_MON_EVENT_BLOCK_RANGE]
   --start.block.height value      Starting height to scan for events (default: 0) [$WITHDRAWAL_MON_START_BLOCK_HEIGHT]
   --optimismportal.address value  Address of the OptimismPortal contract [$WITHDRAWAL_MON_OPTIMISM_PORTAL]
```
