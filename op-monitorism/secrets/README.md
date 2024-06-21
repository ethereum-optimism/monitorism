### Secrets Monitor

The secrets monitor takes a Drippie contract as a parameter and monitors for any drips within that contract that use the CheckSecrets dripcheck contract. CheckSecrets is a dripcheck that allows a drip to begin once a specific secret has been revealed (after a delay period) and cancels the drip if a second secret is revealed. It's important to monitor for these secrets being revealed as this could be a sign that the secret storage platform has been compromised and someone is attempting to exflitrate the ETH controlled by that drip.

```
OPTIONS:
   --l1.node.url value         Node URL of L1 peer (default: "127.0.0.1:8545") [$SECRETS_MON_L1_NODE_URL]
   --drippie.address value     Address of the Drippie contract [$SECRETS_MON_DRIPPIE]
   --log.level value           The lowest log level that will be output (default: INFO) [$MONITORISM_LOG_LEVEL]
   --log.format value          Format the log output. Supported formats: 'text', 'terminal', 'logfmt', 'json', 'json-pretty', (default: text) [$MONITORISM_LOG_FORMAT]
   --log.color                 Color the log output if in terminal mode (default: false) [$MONITORISM_LOG_COLOR]
   --metrics.enabled           Enable the metrics server (default: false) [$MONITORISM_METRICS_ENABLED]
   --metrics.addr value        Metrics listening address (default: "0.0.0.0") [$MONITORISM_METRICS_ADDR]
   --metrics.port value        Metrics listening port (default: 7300) [$MONITORISM_METRICS_PORT]
   --loop.interval.msec value  Loop interval of the monitor in milliseconds (default: 60000) [$MONITORISM_LOOP_INTERVAL_MSEC]
```
