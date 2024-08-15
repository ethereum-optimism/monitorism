### PSP Executor Service

The PSP Executor service is a service designed to execute PSP onchain faster to increase our readiness and speed in case of incident response.

The service is design to listen on a port and execute a PSP onchain when a request is received.

⚠️ The service as to use a authentification method before calling this API ⚠️

### Options and Configuration

The current options are the following:

```
OPTIONS:
   --rpc-url value             Node URL of a peer (default: "http://127.0.0.1:8545") [$PSPEXECUTOR_MON_NODE_URL]
   --privatekey value          Private key of the account that will issue the pause () [$PSPEXECUTOR_MON_PRIVATE_KEY]
   --receiver.address value    The receiver address of the pause request. [$PSPEXECUTOR_MON_RECEIVER_ADDRESS]
   --port.api value            Port of the API server you want to listen on (e.g. 8080). (default: "8080") [$PSPEXECUTOR_MON_PORT_API]
   --data value                calldata to execute the pause on mainnet with the signatures. [$PSPEXECUTOR_MON_CALLDATA]
   --log.level value           The lowest log level that will be output (default: INFO) [$MONITORISM_LOG_LEVEL]
   --log.format value          Format the log output. Supported formats: 'text', 'terminal', 'logfmt', 'json', 'json-pretty', (default: text) [$MONITORISM_LOG_FORMAT]
   --log.color                 Color the log output if in terminal mode (default: false) [$MONITORISM_LOG_COLOR]
   --metrics.enabled           Enable the metrics server (default: false) [$MONITORISM_METRICS_ENABLED]
   --metrics.addr value        Metrics listening address (default: "0.0.0.0") [$MONITORISM_METRICS_ADDR]
   --metrics.port value        Metrics listening port (default: 7300) [$MONITORISM_METRICS_PORT]
   --loop.interval.msec value  Loop interval of the monitor in milliseconds (default: 60000) [$MONITORISM_LOOP_INTERVAL_MSEC]
   --help, -h                  show help
```

### Usage cURL HTTP API

To use the HTTP API you can use the following `curl` command:

```bash
curl -X POST http://${HTTP_API_PSP}:${PORT}/api/psp_execution \-H "Content-Type: application/json" \-d '{"pause": true, "timestamp": 1719432011, "operator": "Tom"}'
```
