### PSP Executor Service

The PSP Executor service is a service designed to execute PSP onchain faster to increase our readiness and speed in case of incident response.

The service is designed to listen on a port and execute a PSP onchain when a request is received.

⚠️ The service has to use an authentication method before calling this API ⚠️

## 1. Usage

### 1. Run HTTP API service

To start the HTTP API service we can use the following oneliner command:
![f112841bad84c59ea3ed1ca380740f5694f553de8755b96b1a40ece4d1c26f81](https://github.com/user-attachments/assets/17235e99-bf25-40a5-af2c-a0d9990c6276)
Settings of the HTTP API service:

| Port                          | API Path             | HTTP Method |
| ----------------------------- | -------------------- | ----------- |
| 8080 (Default can be changed) | `/api/psp_execution` | POST        |

To run the service, you can use the following command:

```shell
go run ../cmd/defender psp_executor --privatekey 2a[..]c6 --safe.address 0x837DE453AD5F21E89771e3c06239d8236c0EFd5E --path /tmp/psps.json --chainid 11155111 --superchainconfig.address 0xC2Be75506d5724086DEB7245bd260Cc9753911Be --rpc.url http://localhost:8545 --port.api 8080
```

Explanation of the options:
| Argument | Value | Explanation |
| ---------------------------- | ------------------------------------------ | ------------------------------------ |
| `--privatekey` | 2a[..]c6 | Private key for transaction signing |
| `--safe.address` | 0x837DE453AD5F21E89771e3c06239d8236c0EFd5E | Address of the Safe contract that presigned the transation|
| `--path` | /tmp/psps.json | Path to JSON file containing PSPs |
| `--chainid` | 11155111 | Chain ID for the network |
| `--superchainconfig.address` | 0xC2Be75506d5724086DEB7245bd260Cc9753911Be | Address of SuperchainConfig contract |
| `--rpc.url` | http://localhost:8545 | URL of the RPC node |
| `--port.api` | 8080 | Port for the HTTP API server |

### 2. Request the HTTP API

To use the HTTP API you can use the following `curl` command with the following fields:

![image](https://github.com/user-attachments/assets/3edc2ee5-6dfd-4872-9bc6-e3ead7444a96)

```bash
curl -X POST http://localhost:8080/api/psp_execution \-H "Content-Type: application/json" \-d '{"Pause":true,"Timestamp":1596240000,"Operator":"tom"}'
```

Explanation of the _mandatory_ fields:
| Field | Description |
| --------- | -------------------------------------------------------------------------------- |
| pause | A boolean value indicating whether to pause (true) or unpause (false) the SuperchainConfig.|
| timestamp | The Unix timestamp when the request is made. |
| operator | The name or identifier of the person initiating the PSP execution. |

### 3. Metrics Server

To monitor the _PSPexecutor service_ the metrics server can be enabled. The metrics server will expose the metrics on the specified address and port.
The metrics are using **Prometheus** and can be set with the following options:
| Option | Description | Default Value | Environment Variable |
| ------------------- | ------------------------- | ------------- | ----------------------------- |
| `--metrics.enabled` | Enable the metrics server | `false` | `DEFENDER_METRICS_ENABLED` |
| `--metrics.addr` | Metrics listening address | `"0.0.0.0"` | `$DEFENDER_METRICS_ADDR` |
| `--metrics.port` | Metrics listening port | `7300` | `$DEFENDER_METRICS_PORT` |

### 4. Options and Configuration

The current options are the following:

```
OPTIONS:
   --rpc.url value                   Node URL of a peer (default: "http://127.0.0.1:8545") [$PSPEXECUTOR_NODE_URL]
   --privatekey value                Privatekey of the account that will issue the pause transaction [$PSPEXECUTOR_PRIVATE_KEY]
   --port.api value                  Port of the API server you want to listen on (e.g. 8080) (default: 8080) [$PSPEXECUTOR_PORT_API]
   --data value                      calldata to execute the pause on mainnet with the signatures [$PSPEXECUTOR_CALLDATA]
   --superchainconfig.address value  SuperChainConfig address to know the current status of the superchainconfig [$PSPEXECUTOR_SUPERCHAINCONFIG_ADDRESS]
   --safe.address value              Safe address that will execute the PSPs [$PSPEXECUTOR_SAFE_ADDRESS]
   --path value                      Absolute path to the JSON file containing the PSPs [$PSPEXECUTOR_PATH_TO_PSPS]
   --chainid value                   ChainID of the current chain that op-defender is running on (default: 0) [$PSPEXECUTOR_CHAIN_ID]
   --log.level value                 The lowest log level that will be output (default: INFO) [$DEFENDER_LOG_LEVEL]
   --log.format value                Format the log output. Supported formats: 'text', 'terminal', 'logfmt', 'json', 'json-pretty', (default: text) [$DEFENDER_LOG_FORMAT]
   --log.color                       Color the log output if in terminal mode (default: false) [$DEFENDER_LOG_COLOR]
   --metrics.enabled                 Enable the metrics server (default: false) [$DEFENDER_METRICS_ENABLED]
   --metrics.addr value              Metrics listening address (default: "0.0.0.0") [$DEFENDER_METRICS_ADDR]
   --metrics.port value              Metrics listening port (default: 7300) [$DEFENDER_METRICS_PORT]
   --help, -h                        show help
```
