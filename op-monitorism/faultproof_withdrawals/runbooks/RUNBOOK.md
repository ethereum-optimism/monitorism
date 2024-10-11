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
On L1, it will make use of [OptimismPortal2](https://github.com/ethereum-optimism/optimism/blob/develop/packages/contracts-bedrock/src/L1/OptimismPortal2.sol) and [FaultDisputeGame](https://github.com/ethereum-optimism/optimism/blob/develop/packages/contracts-bedrock/src/dispute/FaultDisputeGame.sol).

The monitor is driven by the event [WithdrawalProvenExtension1(bytes32 indexed withdrawalHash, address indexed proofSubmitter)](https://github.com/ethereum-optimism/optimism/blob/dd2b21ce786f4c1b722bda270348597182153c8e/packages/contracts-bedrock/src/L1/OptimismPortal2.sol#L144C5-L144C102). Every time an event is emitted, the monitor will check if this withdrawal is legitimate or a forgery attempt.

## ⚠️ Disclaimer: work in progress!

This runbook may contain references to actions and specifications not included in this repository. This runbook is provided as a guideline for incident response to the scenarios detailed herein, but some details may be redacted or missing due to the sensitive nature of the information. Where information is redacted or missing, we will try to make this clear.

The mechanism and the content of the alert is not yet published, so the name of the alerts and the contents of the alerts will depend on your own setup.

## Automated runbooks
Along side this runbook we have some "automated" runbooks. These runbooks can be used to execute some actions either during triaging of an alert or during containement of an incident. These are basically [Jupiter notebooks](https://jupyter.org/), a mix of executable code and markdown, that makes them perfect for putting together instructions and "executable instructions"
Automated runbooks are in the **automated** subfolder**. 
Each runbook is useful for a specific task. After starting the runbook make sure you select the one you need to execute.

---
## Alerts
An incident will be declared upon receiving an alert. The metrics described below trigger various alerts with differing severities. Each alert necessitates specific actions.

### faultproof-withdrawal-forgery-detected

| **Network** | **Severity Level** | **Impact**                      | **Reaction**                                             | **Actions**                                                                                                                                       |
|-------------|--------------------|---------------------------------|----------------------------------------------------------|----------------------------------------------------------------------------------------------------------------------------------------------------|
| Mainnet     | SEV1               | Potential loss of funds          | Immediate action required                                | - Notify the Faultproof (FP) team<br>- Follow the critical incident procedures (private)                                                                              |
| Sepolia     | SEV3               | Potential loss of funds          | Assess the acceptability of the loss on Sepolia<br>Investigate the specific withdrawal | - Notify the Faultproof (FP) team<br>- Follow the critical incident procedures (⚠️ private procedure) |

#### Alert Description

A withdrawal event is considered a forgery if any of the following conditions apply:
1. The `withdrawalHash` is not found on L2, which can be verified by querying the [L2ToL1MessagePasser](https://github.com/ethereum-optimism/optimism/blob/develop/packages/contracts-bedrock/src/L2/L2CrossDomainMessenger.sol).
2. The [outputRoot provided](https://github.com/ethereum-optimism/optimism/blob/dd2b21ce786f4c1b722bda270348597182153c8e/packages/contracts-bedrock/src/L1/OptimismPortal2.sol#L314C15-L314C25) does not match what is recorded in the [L2 block rootState](https://github.com/ethereum-optimism/monitorism/blob/c0b2ecdf4404888e5ceccf6ad14e35c5e5c52664/op-monitorism/faultproof_withdrawals/validator/op_node_helper.go#L47).

However, there are exceptions to these conditions. The event is still considered valid if:
1. The [outputRoot provided](https://github.com/ethereum-optimism/optimism/blob/dd2b21ce786f4c1b722bda270348597182153c8e/packages/contracts-bedrock/src/L1/OptimismPortal2.sol#L314C15-L314C25) is part of a FaultDisputeGame that has been [blacklisted](https://github.com/ethereum-optimism/monitorism/blob/c0b2ecdf4404888e5ceccf6ad14e35c5e5c52664/op-monitorism/faultproof_withdrawals/validator/optimism_portal2_helper.go#L75), or where the game has ended in a [CHALLENGER_WIN](https://github.com/ethereum-optimism/monitorism/blob/c0b2ecdf4404888e5ceccf6ad14e35c5e5c52664/op-monitorism/faultproof_withdrawals/monitor.go#L326).
2. The [outputRoot provided](https://github.com/ethereum-optimism/optimism/blob/dd2b21ce786f4c1b722bda270348597182153c8e/packages/contracts-bedrock/src/L1/OptimismPortal2.sol#L314C15-L314C25) belongs to a FaultDisputeGame that is still [IN_PROGRESS](https://github.com/ethereum-optimism/monitorism/blob/c0b2ecdf4404888e5ceccf6ad14e35c5e5c52664/op-monitorism/faultproof_withdrawals/monitor.go#L336). The metric [faultproof_withdrawals_invalid_proposal_withdrawals_events_count](#faultproof_withdrawals_invalid_proposal_withdrawals_events_count) is used to track such events.

#### Triage Phase

The alert (⚠️ private alert details) includes the transaction hash that triggered the event. The first step after receiving the alert is to verify whether the attack is real or if it resulted from a monitoring system error or a node issue.

To confirm the attack, begin by reviewing the event details and ensuring the conditions for the attack are met.

You can use the automated `op-monitorism/faultproof_withdrawals/runbooks/automated/triage_potential_attack_event.ipynb` runbook for this process.


### faultproof-potential-withdrawal-forgery-detected

| **Network** | **Severity Level** | **Impact**                      | **Reaction**                                             | **Actions**                                                                                                                                       |
|-------------|--------------------|---------------------------------|----------------------------------------------------------|----------------------------------------------------------------------------------------------------------------------------------------------------|
| Mainnet     | SEV3               | No immediate impact                       | Investigate the attack, monitor how the attack is proceeding to make sure to be ready to react quickly in case of success                               |  If new type of attack, write a report and eventually create a feature request for improving monitorism monitoring capabilities                                                                            |
| Sepolia     | SEV3               | No immediate impact                       | Investigate the attack, monitor how the attack is proceeding to make sure to be ready to react quickly in case of success                              | If new type of attack, write a report and eventually create a feature request for improving monitorism monitoring capabilities |

##### Alert Description
An event is considered a potential forgery if any of the following conditions apply:
1. The withdrawalHash is not present on L2. We check this by querying [L2ToL1MessagePasser](https://github.com/ethereum-optimism/optimism/blob/develop/packages/contracts-bedrock/src/L2/L2CrossDomainMessenger.sol).
2. The [outputRoot provided](https://github.com/ethereum-optimism/optimism/blob/dd2b21ce786f4c1b722bda270348597182153c8e/packages/contracts-bedrock/src/L1/OptimismPortal2.sol#L314C15-L314C25) does not match what we see on [L2 block rootState](https://github.com/ethereum-optimism/monitorism/blob/c0b2ecdf4404888e5ceccf6ad14e35c5e5c52664/op-monitorism/faultproof_withdrawals/validator/op_node_helper.go#L47).

and
1. Dispute Game status is IN_PROGRESS

#### Triage Phase

The alert (⚠️ private alert details) includes the transaction hash that triggered the event. The first step after receiving the alert is to verify whether the attack is real or if it resulted from a monitoring system error or a node issue.

To confirm the attack, begin by reviewing the event details and ensuring the conditions for the attack are met.

You can use the automated `op-monitorism/faultproof_withdrawals/runbooks/automated/triage_potential_attack_event.ipynb` runbook for this process.

### faultproof-suspicious-withdrawal-forgery-detected

| **Network** | **Severity Level** | **Impact**                      | **Reaction**                                             | **Actions**                                                                                                                                       |
|-------------|--------------------|---------------------------------|----------------------------------------------------------|----------------------------------------------------------------------------------------------------------------------------------------------------|
| Mainnet     | SEV5               | No immediate impact                       | Investigate the attack, if not investigated already                               |  If new type of attack, write a report and eventually create a feature request for improving monitorism monitoring capabilities                                                                            |
| Sepolia     | SEV5               | No immediate impact                       | Investigate the attack, if not investigated already                              | If new type of attack, write a report and eventually create a feature request for improving monitorism monitoring capabilities |

##### Alert Description
An event is considered a potential forgery if any of the following conditions apply:
1. The withdrawalHash is not present on L2. We check this by querying [L2ToL1MessagePasser](https://github.com/ethereum-optimism/optimism/blob/develop/packages/contracts-bedrock/src/L2/L2CrossDomainMessenger.sol).
2. The [outputRoot provided](https://github.com/ethereum-optimism/optimism/blob/dd2b21ce786f4c1b722bda270348597182153c8e/packages/contracts-bedrock/src/L1/OptimismPortal2.sol#L314C15-L314C25) does not match what we see on [L2 block rootState](https://github.com/ethereum-optimism/monitorism/blob/c0b2ecdf4404888e5ceccf6ad14e35c5e5c52664/op-monitorism/faultproof_withdrawals/validator/op_node_helper.go#L47).

and
1. Dispute Game status is CHALLENGER_WIN

#### Triage Phase

The alert (⚠️ private alert details) includes the transaction hash that triggered the event. The first step after receiving the alert is to verify whether the attack is real or if it resulted from a monitoring system error or a node issue.

To confirm the attack, begin by reviewing the event details and ensuring the conditions for the attack are met.

You can use the automated `op-monitorism/faultproof_withdrawals/runbooks/automated/triage_potential_attack_event.ipynb` runbook for this process.

### faultproof-withdrawal-forgery-detection-stalled

| **Network** | **Severity Level** | **Impact**                                          | **Cause**                             | **Actions**                                                  |
|-------------|--------------------|-----------------------------------------------------|---------------------------------------|--------------------------------------------------------------|
| Mainnet     | SEV2               | - Security max reaction time is reduced<br>- May not be able to react properly to an attack | Daemon is not processing withdrawals  | - Understand the issue with the daemon<br>- If necessary, restart the service |
| Sepolia     | SEV3               | - Security max reaction time is reduced<br>- May not be able to react properly to an attack | Daemon is not processing withdrawals  | - Understand the issue with the daemon<br>- If necessary, restart the service |

#### Triage Phase

The alert (⚠️ private alert details) includes the transaction hash that triggered the event. The first step after receiving the alert is to verify whether the monitoring is not processing event anymore and is stalled for some internal issue or the chain is in reality not processing any events since more then a day.

To confirm review the chain, see when it was last withdrawals event on it and confirm if the event happened or not within 24 hours.

You can use the automated `op-monitorism/faultproof_withdrawals/runbooks/automated/triage_detection_stalled.ipynb` runbook for this process.

##### Alert Description
This alert monitors the number of withdrawal events that are considered normal in a chain. If the number of withdrawal events goes below a specified threshold, we trigger this alert.

### faultproof-withdrawal-forgery-detection-error-unhandled

| **Network** | **Severity Level** | **Impact**                                          | **Cause**                               | **Actions**                                                                                     |
|-------------|--------------------|-----------------------------------------------------|-----------------------------------------|-------------------------------------------------------------------------------------------------|
| Mainnet     | SEV2               | - Security max reaction time is reduced<br>- May not be able to react properly to an attack | Too many errors; something may be wrong | - Find out what the error is<br>- Decide if the daemon needs to be patched or configuration changed<br>- If necessary, restart the service |
| Sepolia     | SEV3               | - Security max reaction time is reduced<br>- May not be able to react properly to an attack | Too many errors; something may be wrong | - Find out what the error is<br>- Decide if the daemon needs to be patched or configuration changed<br>- If necessary, restart the service |

##### Alert Description
This alert will be triggered when the number of connection errors goes above a specified threshold. Errors should always be very limited or absent in the monitoring. When present, it often means there is an issue with communication between the monitor and the trusted nodes used for monitoring.

#### Triage Phase

The alert (⚠️ private alert details) includes a dashboard link where you can review logs to diagnose potential issues with the monitoring system and understand the cause of the alert.

---
## Metrics and Alerts Conditions

### `faultproof_withdrawals_potential_attack_on_defender_wins_games_count`

- **Description:** Number of attacks detected in which the defender wins. In this case, the faultproof system failed, and an attacker was able to forge a withdrawal.
- **Alert:**
  - **Condition:** If the value exceeds **0**.
  - **Action:** Immediately investigate potential security breaches. Review transaction validation mechanisms.
  - **Alert Name:** [faultproof-withdrawal-forgery-detected](#faultproof-withdrawal-forgery-detected)

### `faultproof_withdrawals_potential_attack_on_in_progress_games_count`

- **Description:** This is an attack in which the dispute game is still in progress. This is not yet a problem, as the faultproof system should be able to challenge this game and end with CHALLENGER_WIN. In the latter case, this will not be an issue.
- **Alert:**
  - **Condition:** If the value exceeds **0**.
  - **Action:** In this case, we need to keep an eye on this event. This is not yet a problem, and may not become a problem, but we may want to investigate who is attempting an attack.
  - **Alert Name:** [faultproof-potential-withdrawal-forgery-detected](#faultproof-withdrawal-forgery-detected)

### `faultproof_withdrawals_suspicious_events_on_challenger_wins_games_count`

- **Description:** This is the total number of suspicious withdrawals. In normal circumstances, this metric will be incremented by 1, and `faultproof_withdrawals_potential_attack_on_in_progress_games_count` will be decremented by 1.
- **Type:** Counter
- **Alert:**
  - **Condition:** If the value is not increasing as expected.
  - **Action:** No immediate actions are required. This information is useful for threat hunting and measuring how many attack attempts have been detected since monitoring began.
  - **Alert Name:** [faultproof-suspicious-withdrawal-forgery-detected](#faultproof-withdrawal-forgery-detection-stalled)

### `faultproof_withdrawals_node_connection_failures_total`

- **Description:** Total number of node connection failures.
- **Alert:**
  - **Condition:** If the value increases over time.
  - **Action:** Investigate network issues or node outages. Check logs for connection errors and attempt to reconnect.
  - **Alert Name:** [faultproof-withdrawal-forgery-detection-error-unhandled](#faultproof-withdrawal-forgery-detection-error-unhandled)

### `faultproof_withdrawals_events_processed_total`

- **Description:** Number of withdrawal events processed. If the number of withdrawals observed on the chain drops below a certain threshold (e.g., usually more than 1 per day), this alert will be triggered.
- **Alert:**
  - **Condition:** If the metric's increase falls below a set threshold.
  - **Action:** Investigate whether the chain had withdrawals during the relevant period to confirm whether the issue is related to monitoring or a lack of activity on the network.
  - **Alert Name:** [faultproof-withdrawal-forgery-detection-stalled](#faultproof-withdrawal-forgery-detection-stalled)

### `faultproof_withdrawals_withdrawals_processed_total`

- **Description:** Number of withdrawals processed. These withdrawals are complete, and they are forgotten.

### `faultproof_withdrawals_initial_l1_height`

- **Description:** Indicates the initial L1 (Layer 1) block height at the start of the monitoring period.

### `faultproof_withdrawals_invalid_proposal_withdrawals_events_count`

- **Description:** Tracks the number of invalid proposal withdrawal events.

### `faultproof_withdrawals_latest_l1_height`

- **Description:** Indicates the latest observed L1 block height.

### `faultproof_withdrawals_next_l1_height`

- **Description:** Represents the next expected L1 block height.

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
