# Runbook: Incident Response for Faultproof Withdrawals
- [Runbook](#runbook)
    - [Overview](#overview)
    - [Incident Management](#incident-management)
    - [General Metrics and Alerts Descriptions](#general-metrics-and-alerts-descriptions)
        - [faultproof_withdrawals_forgeries_withdrawals_events_count](#faultproof_withdrawals_forgeries_withdrawals_events_count)
        - [faultproof_withdrawals_initial_l1_height](#faultproof_withdrawals_initial_l1_height)
        - [faultproof_withdrawals_invalid_proposal_withdrawals_events_count](#faultproof_withdrawals_invalid_proposal_withdrawals_events_count)
        - [faultproof_withdrawals_latest_l1_height](#faultproof_withdrawals_latest_l1_height)
        - [faultproof_withdrawals_next_l1_height](#faultproof_withdrawals_next_l1_height)
        - [faultproof_withdrawals_node_connection_failures_total](#faultproof_withdrawals_node_connection_failures_total)
        - [faultproof_withdrawals_number_of_detected_forgeries](#faultproof_withdrawals_number_of_detected_forgeries)
        - [faultproof_withdrawals_number_of_invalid_withdrawals](#faultproof_withdrawals_number_of_invalid_withdrawals)
        - [faultproof_withdrawals_processed_provenwithdrawalsextension1_events_total](#faultproof_withdrawals_processed_provenwithdrawalsextension1_events_total)
        - [faultproof_withdrawals_withdrawals_validated_total](#faultproof_withdrawals_withdrawals_validated_total)
    - [General Incident Response Guidelines](#general-incident-response-guidelines)
    - [Conclusion](#conclusion)

## Overview

This document serves as a guide for incident response based on key metrics related to faultproof withdrawals. It describes the alerts triggered by specific conditions in the system and provides guidelines on how to handle these alerts.

---
## Incident Management
An incident will be declared upon receiving an alert. The metrics described below trigger various alerts with differing severities. Each alert necessitates specific actions.

## Metrics and Alerts Conditions

### `faultproof_withdrawals_forgeries_withdrawals_events_count`

- **Description:** Tracks the number of forgery withdrawal events.
- **Type:** Gauge
- **Alert:**
  - **Condition:** If the value exceeds **0**.
  - **Action:** Immediately investigate to determine the cause of the forgery events. Check the integrity of the withdrawal processing logic and system for potential security breaches.

### `faultproof_withdrawals_initial_l1_height`

- **Description:** Indicates the initial L1 (Layer 1) block height at the start of the monitoring period.
- **Type:** Gauge

### `faultproof_withdrawals_invalid_proposal_withdrawals_events_count`

- **Description:** Tracks the number of invalid proposal withdrawal events.
- **Type:** Gauge
- **Alert:**
  - **Condition:** If the value exceeds **0**.
  - **Action:** Investigate the cause of invalid proposals. This could indicate errors in the proposal validation logic or issues in L1-L2 communication.

### `faultproof_withdrawals_latest_l1_height`

- **Description:** Indicates the latest observed L1 block height.
- **Type:** Gauge

### `faultproof_withdrawals_next_l1_height`

- **Description:** Represents the next expected L1 block height.
- **Type:** Gauge

### `faultproof_withdrawals_node_connection_failures_total`

- **Description:** Total number of node connection failures.
- **Type:** Counter
- **Alert:**
  - **Condition:** If the value increases over time.
  - **Action:** Investigate network issues or node outages. Check logs for connection errors and attempt to reconnect.

### `faultproof_withdrawals_number_of_detected_forgeries`

- **Description:** Number of detected forgeries in the system.
- **Type:** Gauge
- **Alert:**
  - **Condition:** If the value exceeds **0**.
  - **Action:** Immediately investigate potential security breaches. Review transaction validation mechanisms.

### `faultproof_withdrawals_number_of_invalid_withdrawals`

- **Description:** Number of invalid withdrawals processed.
- **Type:** Gauge
- **Alert:**
  - **Condition:** If the value exceeds **0**.
  - **Action:** Examine the reasons for invalid withdrawals. Check for issues in withdrawal requests and validation logic.

### `faultproof_withdrawals_processed_provenwithdrawalsextension1_events_total`

- **Description:** Total number of processed `ProvenWithdrawalsExtension1` events.
- **Type:** Counter

### `faultproof_withdrawals_withdrawals_validated_total`

- **Description:** Total number of withdrawals validated successfully.
- **Type:** Counter
- **Alert:**
  - **Condition:** If the value is not increasing as expected.
  - **Action:** Investigate potential issues in the validation process. Ensure that the system is processing withdrawals correctly.

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
