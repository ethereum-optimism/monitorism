### Multisig Registry Monitor

The Multisig Registry Monitor is a service designed to continuously monitor Gnosis Safe multisig wallets by comparing their onchain state with records stored in a Notion database. The service validates threshold settings, signer counts, and balance levels to ensure multisig configurations remain accurate and secure.

The service provides real-time alerting through webhooks and comprehensive metrics for monitoring multisig wallet health across networks.

⚠️ The service requires valid Notion API credentials and database access ⚠️

## 1. Usage

### 1. Run Monitoring Service

To start the multisig monitoring service, use the following command:

```shell
go run ../cmd/monitorism multisig --notion.database.id 24ffb7d8d2cc80e8885ee1bb3bc1f53b --notion.token secret_abc123 --l1.node.url https://mainnet.infura.io/v3/your-key --nickname multisig_registry
```

**Core Monitoring Features:**

| Feature | Description |
| ------- | ----------- |
| Threshold Validation | Compares onchain Safe threshold with Notion records |
| Signer Count Validation | Verifies number of Safe owners matches Notion data |
| Balance Monitoring | Tracks native token balances in USD with risk assessment |
| Risk Level Assessment | Automatically categorizes Safes based on balance thresholds |
| Webhook Alerts | Sends formatted alerts to Discord/Slack for anomalies |

### 2. Configuration Options

**Required Arguments:**
| Argument | Example Value | Explanation |
| -------- | ------------- | ----------- |
| `--notion.database.id` | 24ffb7d8d2cc80e8885ee1bb3bc1f53b | Notion database ID containing Safe records |
| `--notion.token` | secret_abc123 | Notion integration token (API key) |
| `--l1.node.url` | https://mainnet.infura.io/v3/your-key | Ethereum RPC node URL |
| `--nickname` | mainnet | Network identifier for metrics labeling |

**Optional Arguments:**
| Argument | Default Value | Explanation |
| -------- | ------------- | ----------- |
| `--webhook.url` | "" | Webhook URL for sending alerts (Discord/Slack) |
| `--high.value.threshold.usd` | 1000000 | USD threshold for high-value Safe validation ($1M) |

### 3. Notion Database Schema

The service expects a Notion database with the following properties: 
**Required Columns:**
| Column Name | Type | Description |
| ----------- | ---- | ----------- |
| Name | title | Safe wallet name |
| Address | text | Ethereum address of the Safe |
| Threshold | number | Required number of signatures |
| Signer count | number | Total number of Safe owners |
| Risk Band | select | Risk level (Critical, High, Medium, Low) |

**Optional Columns:**
| Column Name | Type | Description |
| ----------- | ---- | ----------- |
| Networks | multi-select | Supported networks |
| Multisig Lead | people | Responsible team members |
| Has Monitoring | checkbox | Monitoring status |
| Has Backup Chat | checkbox | Backup chat status | 
| Last Review By | people | Last Reviewer of the Template | 
| Last Review Date | date | Last Review date by a Reviewer | 


### 4. Alert Examples

The service automatically sends alerts for various conditions:

**Threshold Mismatch:**
<img width="1081" height="330" alt="9e1522fd9f2b24d8e9be96f43531b4e00ca0cb17b00a1647834695205988b29d" src="https://github.com/user-attachments/assets/c7759302-10a2-4f20-8ffb-4297db4e1fea" />


**Signer Mismatch:**
<img width="924" height="281" alt="61600282cec0e8bc627e4919b8af31fe185b1fa4b8413d3605d09731fa54839d" src="https://github.com/user-attachments/assets/8dd5f9a1-5c84-4505-99e2-6701c07c9ba1" />



**Safe Balance Criticity Alerts:**
<img width="1086" height="334" alt="e2e8a6d8640404b48baa6a268dc1c52291eba1867d1c42487f510618e78e7de5" src="https://github.com/user-attachments/assets/4c813df2-23b9-45bf-9230-67345ff97b20" />



### 5. Metrics Server

The service exposes Prometheus metrics for monitoring:

**Key Metrics:**
```golang
multisig_registry_threshold_mismatch        // 1 if mismatch detected, 0 if matches
multisig_registry_signer_count_mismatch     // 1 if signer count differs, 0 if matches  
multisig_registry_safe_native_balance_eth   // ETH balance of each Safe
multisig_registry_safe_native_balance_usd   // USD balance of each Safe
multisig_registry_safe_risk_level           // Risk level: 1=low, 2=medium, 3=high, 4=critical
multisig_registry_safe_accessible          // 1 if Safe accessible, 0 if not
multisig_registry_unexpected_errors         // Counter for various error types
multisig_registry_total_safes_monitored     // Total number of Safes being monitored
```

### 7. Price Data Sources

ETH price fetching includes automatic failover:

1. **Primary:** CoinGecko API
2. **Fallback:** Binance API

If the primary source fails, the service automatically switches to the backup source to ensure continuous monitoring.
You can also add more custom feed if necessary for more robusteness. 

### 8. Command Examples

**Basic Monitoring (Mainnet):**
```shell
./monitorism multisig \
  --notion.database.id=your-database-id \
  --notion.token=secret_your-token \
  --l1.node.url=https://mainnet.infura.io/v3/your-key \
  --nickname=mainnet
```

**With Webhook Alerts:**
```shell
./monitorism multisig \
  --notion.database.id=your-database-id \
  --notion.token=secret_your-token \
  --l1.node.url=https://mainnet.infura.io/v3/your-key \
  --nickname=mainnet \
  --webhook.url=https://hooks.slack.com/services/your/webhook/url
```

**Custom High-Value Threshold ($5M):**
```shell
./monitorism multisig \
  --notion.database.id=your-database-id \
  --notion.token=secret_your-token \
  --l1.node.url=https://mainnet.infura.io/v3/your-key \
  --nickname=mainnet \
  --high.value.threshold.usd=5000000
```

**Using Environment Variables:**
```shell
export OP_MONITORISM_NOTION_DATABASE_ID=your-database-id
export OP_MONITORISM_NOTION_TOKEN=secret_your-token
export OP_MONITORISM_WEBHOOK_URL=https://hooks.slack.com/services/your/webhook/url
export OP_MONITORISM_HIGH_VALUE_THRESHOLD_USD=2000000

./monitorism multisig --l1.node.url=https://mainnet.infura.io/v3/your-key --nickname=mainnet
```

### 9. Options and Configuration

Using the `--help` flag will show all available options:

**OPTIONS:**

```shell
   --l1.node.url value                  Node URL of L1 peer (default: "127.0.0.1:8545") [$OP_MONITORISM_L1_NODE_URL]
   --nickname value                     Nickname of chain being monitored [$OP_MONITORISM_NICKNAME]
   --notion.database.id value           Notion database ID containing Safe records [$OP_MONITORISM_NOTION_DATABASE_ID]
   --notion.token value                 Notion integration token (API key) [$OP_MONITORISM_NOTION_TOKEN]
   --webhook.url value                  Webhook URL for sending alerts (optional) [$OP_MONITORISM_WEBHOOK_URL]
   --high.value.threshold.usd value     USD threshold for high-value Safe validation (default: 1000000) [$OP_MONITORISM_HIGH_VALUE_THRESHOLD_USD]
   --log.level value                    The lowest log level that will be output (default: INFO) [$OP_MONITORISM_LOG_LEVEL]
   --log.format value                   Format the log output. Supported formats: 'text', 'terminal', 'logfmt', 'json', 'json-pretty' (default: text) [$OP_MONITORISM_LOG_FORMAT]
   --log.color                          Color the log output if in terminal mode (default: false) [$OP_MONITORISM_LOG_COLOR]
   --help, -h                          show help
```


For additional support, check the logs with `--log.level=DEBUG` for detailed troubleshooting information.
