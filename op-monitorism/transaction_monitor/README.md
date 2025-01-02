# Transaction Monitor

A service that monitors Ethereum transactions for specific addresses and raises alerts based on configurable rules.

## Features

- Monitor transactions to specific addresses
- Configurable address filtering with multiple check types:
  - Exact match checking for allowlisted addresses
  - Dynamic allowlisting through dispute game factory events
- Threshold monitoring for transaction values
- Prometheus metrics for monitoring and alerting
- Flexible YAML configuration

## Configuration

Configuration is provided via YAML file. Example:

```yaml
node_url: "http://localhost:8545"
start_block: 0
watch_configs:
  - address: "0xAE0b5DF2dFaaCD6EB6c1c56Cc710f529F31C6C44"
    filters:
      - type: exact_match
        params:
          match: "0x1234567890123456789012345678901234567890"
      - type: dispute_game
        params:
          disputeGameFactory: "0x9876543210987654321098765432109876543210"
    thresholds:
      "0x1234567890123456789012345678901234567890": "1000000000000000000"
```

* A `start_block` set to `0` indicates the latest block. 

## Metrics

The service exports the following Prometheus metrics:

- `tx_mon_transactions_total`: Total number of transactions processed
- `tx_mon_unauthorized_transactions_total`: Number of transactions from unauthorized addresses
- `tx_mon_threshold_exceeded_transactions_total`: Number of transactions exceeding allowed threshold
- `tx_mon_eth_spent_total`: Cumulative ETH spent by address
- `tx_mon_unexpected_rpc_errors_total`: Number of unexpected RPC errors

## Usage

```bash
monitorism \
  --node.url=http://localhost:8545 \
  --config.file=config.yaml
```

