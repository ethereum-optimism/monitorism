# Runbook: Incident Response for Faultproof Withdrawals
- [Runbook](#runbook)
    - [Overview](#overview)
    - [Incident Management](#incident-management)
    - [General Metrics and Alerts Descriptions](#general-metrics-and-alerts-descriptions)
    - [General Incident Response Guidelines](#general-incident-response-guidelines)
    - [Conclusion](#conclusion)

## Overview

This document serves as a guide for incident response based on key metrics related to faultproof withdrawals. It describes the alerts triggered by specific conditions in the system and provides guidelines on how to handle these alerts.

FaultproofWithdrawal will monitor one L2 chain, let's say op-chain, and one L1 chain, let's say mainnet.
On L2, it will make use of [L2ToL1MessagePasser](https://github.com/ethereum-optimism/optimism/blob/develop/packages/contracts-bedrock/src/L2/L2CrossDomainMessenger.sol).
On L1, it will make use of [OptimismPortal2](https://github.com/ethereum-optimism/optimism/blob/develop/packages/contracts-bedrock/src/L1/OptimismPortal2.sol), [FaultDisputeGame](https://github.com/ethereum-optimism/optimism/blob/develop/packages/contracts-bedrock/src/dispute/FaultDisputeGame.sol).

The monitor is driven by the event [WithdrawalProvenExtension1(bytes32 indexed withdrawalHash, address indexed proofSubmitter)](https://github.com/ethereum-optimism/optimism/blob/dd2b21ce786f4c1b722bda270348597182153c8e/packages/contracts-bedrock/src/L1/OptimismPortal2.sol#L144C5-L144C102). Every time an event is emitted, the monitor will check if this withdrawal is legitimate or a forgery attempt.

## Disclaimer

This runbook may contain references to actions and specifications not included in this repository. This runbook is provided as a guideline for incident response to the scenarios detailed herein, but some details may be redacted or missing due to the sensitive nature of the information. Where information is redacted or missing, we will try to make this clear.

---
## Alerts
An incident will be declared upon receiving an alert. The metrics described below trigger various alerts with differing severities. Each alert necessitates specific actions.

### faultproof-withdrawal-forgery-detected

| **Network** | **Severity Level** | **Impact**                      | **Reaction**                                             | **Actions**                                                                                                                                       |
|-------------|--------------------|---------------------------------|----------------------------------------------------------|----------------------------------------------------------------------------------------------------------------------------------------------------|
| Mainnet     | SEV1               | Fund loss                       | Immediate action required                                | - Page Faultproof (FP) team<br>- Follow critical incident procedures                                                                               |
| Sepolia     | SEV3               | Fund loss (assumed 3.5 days)    | Assess acceptability of loss on Sepolia<br>Investigate the specific withdrawal | **Initial Actions:**<br>- Page FP team<br><br>**After Coordinating with FP Team:**<br>1. Blacklist the game (FP)<br>2. Execute presigned pause (Security)<br>3. Revert to permissioned game (FP) |

##### Alert Description
An event is considered a forgery if any of the following conditions apply:
1. The withdrawalHash is not present on L2. We check this by querying [L2ToL1MessagePasser](https://github.com/ethereum-optimism/optimism/blob/develop/packages/contracts-bedrock/src/L2/L2CrossDomainMessenger.sol).
2. The [outputRoot provided](https://github.com/ethereum-optimism/optimism/blob/dd2b21ce786f4c1b722bda270348597182153c8e/packages/contracts-bedrock/src/L1/OptimismPortal2.sol#L314C15-L314C25) does not match what we see on [L2 block rootState](https://github.com/ethereum-optimism/monitorism/blob/c0b2ecdf4404888e5ceccf6ad14e35c5e5c52664/op-monitorism/faultproof_withdrawals/validator/op_node_helper.go#L47).

There are exceptions to the rule above. The event is still considered valid if:
1. The [outputRoot provided](https://github.com/ethereum-optimism/optimism/blob/dd2b21ce786f4c1b722bda270348597182153c8e/packages/contracts-bedrock/src/L1/OptimismPortal2.sol#L314C15-L314C25) belongs to a FaultDisputeGame that is [blacklisted](https://github.com/ethereum-optimism/monitorism/blob/c0b2ecdf4404888e5ceccf6ad14e35c5e5c52664/op-monitorism/faultproof_withdrawals/validator/optimism_portal2_helper.go#L75), or to a FaultDisputeGame for which the game is in state [CHALLENGER_WIN](https://github.com/ethereum-optimism/monitorism/blob/c0b2ecdf4404888e5ceccf6ad14e35c5e5c52664/op-monitorism/faultproof_withdrawals/monitor.go#L326).
2. The [outputRoot provided](https://github.com/ethereum-optimism/optimism/blob/dd2b21ce786f4c1b722bda270348597182153c8e/packages/contracts-bedrock/src/L1/OptimismPortal2.sol#L314C15-L314C25) belongs to a FaultDisputeGame that has the status [IN_PROGRESS](https://github.com/ethereum-optimism/monitorism/blob/c0b2ecdf4404888e5ceccf6ad14e35c5e5c52664/op-monitorism/faultproof_withdrawals/monitor.go#L336). We use the [faultproof_withdrawals_invalid_proposal_withdrawals_events_count](#faultproof_withdrawals_invalid_proposal_withdrawals_events_count) metric to track this event.

#### Investigation
To investigate this alert, we need to delineate the possible causes that generate it. This alert will trigger when a forgery is detected.

The investigation has two parts:
1. Investigation to triage if the event is a true positive.
2. Investigation to act.

TODO: Detail the following steps.

### faultproof-withdrawal-forgery-detection-stalled

| **Network** | **Severity Level** | **Impact**                                          | **Cause**                             | **Actions**                                                  |
|-------------|--------------------|-----------------------------------------------------|---------------------------------------|--------------------------------------------------------------|
| Mainnet     | SEV2               | - Security max reaction time is reduced<br>- May not be able to react properly to an attack | Daemon is not processing withdrawals  | - Understand the issue with the daemon<br>- If necessary, restart the service |
| Sepolia     | SEV3               | - Security max reaction time is reduced<br>- May not be able to react properly to an attack | Daemon is not processing withdrawals  | - Understand the issue with the daemon<br>- If necessary, restart the service |

##### Alert Description
This alert monitors the number of withdrawal events that are considered normal in a chain. If the number of withdrawal events goes below a specified threshold, we trigger this alert.

#### Investigation
The investigation has two parts:
1. Investigation to triage if the event is a true positive.
2. Investigation to act.

TODO: Detail the following steps.

### faultproof-withdrawal-forgery-detection-error-unhandled

| **Network** | **Severity Level** | **Impact**                                          | **Cause**                               | **Actions**                                                                                     |
|-------------|--------------------|-----------------------------------------------------|-----------------------------------------|-------------------------------------------------------------------------------------------------|
| Mainnet     | SEV2               | - Security max reaction time is reduced<br>- May not be able to react properly to an attack | Too many errors; something may be wrong | - Find out what the error is<br>- Decide if the daemon needs to be patched or configuration changed<br>- If necessary, restart the service |
| Sepolia     | SEV3               | - Security max reaction time is reduced<br>- May not be able to react properly to an attack | Too many errors; something may be wrong | - Find out what the error is<br>- Decide if the daemon needs to be patched or configuration changed<br>- If necessary, restart the service |

##### Alert Description
This alert will be triggered when the number of connection errors goes above a specified threshold. Errors should always be very limited or absent in the monitoring. When present, it often means there is an issue with communication between the monitor and the trusted nodes used for monitoring.

#### Investigation

The investigation has two parts:
1. Investigation to triage if the event is a true positive.
2. Investigation to act.

TODO: Detail the following steps.

---
## Metrics and Alerts Conditions

### `faultproof_withdrawals_number_of_detected_forgeries`

- **Description:** Number of detected forgeries in the system.
- **Alert:**
  - **Condition:** If the value exceeds **0**.
  - **Action:** Immediately investigate potential security breaches. Review transaction validation mechanisms.
  - **Alert Name:** [faultproof-withdrawal-forgery-detected](#faultproof-withdrawal-forgery-detected)

### `faultproof_withdrawals_forgeries_withdrawals_events_count`

- **Description:** Tracks the number of forgery withdrawal events.
- **Alert:**
  - **Condition:** If the value exceeds **0**.
  - **Action:** Immediately investigate to determine the cause of the forgery events. Check the integrity of the withdrawal processing logic and system for potential security breaches.
  - **Alert Name:** [faultproof-withdrawal-forgery-detected](#faultproof-withdrawal-forgery-detected)

### `faultproof_withdrawals_withdrawals_validated_total`

- **Description:** Total number of withdrawals validated successfully.
- **Type:** Counter
- **Alert:**
  - **Condition:** If the value is not increasing as expected.
  - **Action:** Investigate potential issues in the validation process. Ensure that the system is processing withdrawals correctly.
  - **Alert Name:** [faultproof-withdrawal-forgery-detection-stalled](#faultproof-withdrawal-forgery-detection-stalled)

### `faultproof_withdrawals_node_connection_failures_total`

- **Description:** Total number of node connection failures.
- **Alert:**
  - **Condition:** If the value increases over time.
  - **Action:** Investigate network issues or node outages. Check logs for connection errors and attempt to reconnect.
  - **Alert Name:** [faultproof-withdrawal-forgery-detection-error-unhandled](#faultproof-withdrawal-forgery-detection-error-unhandled)

### `faultproof_withdrawals_initial_l1_height`

- **Description:** Indicates the initial L1 (Layer 1) block height at the start of the monitoring period.

### `faultproof_withdrawals_invalid_proposal_withdrawals_events_count`

- **Description:** Tracks the number of invalid proposal withdrawal events.

### `faultproof_withdrawals_latest_l1_height`

- **Description:** Indicates the latest observed L1 block height.

### `faultproof_withdrawals_next_l1_height`

- **Description:** Represents the next expected L1 block height.

### `faultproof_withdrawals_number_of_invalid_withdrawals`

- **Description:** Number of invalid withdrawals processed.

### `faultproof_withdrawals_processed_provenwithdrawalsextension1_events_total`

- **Description:** Total number of processed `ProvenWithdrawalsExtension1` events.

---

## Incident Response

- **Investigate System Logs:** For each alert, review the logs from the affected services to gather more information.
- **Check Node Health:** Verify the status of all nodes involved, especially if connection failures are reported.
- **Coordinate with Relevant Teams:** Communicate with development, operations, and security teams as necessary.
- **Document Findings:** Record the incident details, root cause, and resolution steps for future reference.

---

## Conclusion

This runbook provides guidance on responding to incidents related to faultproof withdrawals based on specific metrics. Regular monitoring and prompt response to alerts are crucial to maintaining system integrity and performance.

---
