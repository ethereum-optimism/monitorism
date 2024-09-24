# Global Events Monitoring

This monitoring modules is made for to taking YAML rules as configuration.
![df2b94999628ce8eee98fb60f45667e54be9b13db82add6aa77888f355137329](https://github.com/ethereum-optimism/monitorism/assets/23560242/b8d36a0f-8a17-4e22-be5a-3e9f3586b3ab)

Once the Yaml rules is configured correctly, we can listen to an event choosen to send the data through prometheus.

## CLI and Docs:

### CLI Args

```bash
NAME:
   Monitorism global_events - Monitors global events with YAML configuration

USAGE:
   Monitorism global_events [command options] [arguments...]

DESCRIPTION:
   Monitors global events with YAML configuration

OPTIONS:
   --l1.node.url value         Node URL of L1 peer (default: "http://127.0.0.1:8545") [$GLOBAL_EVENT_MON_L1_NODE_URL]
   --nickname value            Nickname of the chain being monitored [$GLOBAL_EVENT_MON_NICKNAME]
   --PathYamlRules value       Path to the directory containing the yaml files with the events to monitor [$GLOBAL_EVENT_MON_PATH_YAML]
   --log.level value           The lowest log level that will be output (default: INFO) [$MONITORISM_LOG_LEVEL]
   --log.format value          Format the log output. Supported formats: 'text', 'terminal', 'logfmt', 'json', 'json-pretty', (default: text) [$MONITORISM_LOG_FORMAT]
   --log.color                 Color the log output if in terminal mode (default: false) [$MONITORISM_LOG_COLOR]
   --metrics.enabled           Enable the metrics server (default: false) [$MONITORISM_METRICS_ENABLED]
   --metrics.addr value        Metrics listening address (default: "0.0.0.0") [$MONITORISM_METRICS_ADDR]
   --metrics.port value        Metrics listening port (default: 7300) [$MONITORISM_METRICS_PORT]
   --loop.interval.msec value  Loop interval of the monitor in milliseconds (default: 60000) [$MONITORISM_LOOP_INTERVAL_MSEC]
   --help, -h                  show help

```

### Yaml rules

The rules are located here: `op-monitorism/global_events/rules/`. Then we have multiple folders depending on the networks you want to monitor (`mainnet` or `sepolia`) for now.

```yaml
# This is a TEMPLATE file please copy this one
# This watches all contacts for OP, Mode, and Base mainnets for two logs.
version: 1.0
name: Template SafeExecution Events (Success/Failure) L1 # Please put the L1 or L2 at the end of the name.
priority: P5 # This is a test, so it is a P5
#If addresses are empty like below, it will watch all addresses; otherwise, you can address specific addresses.
addresses:
  # - 0xbEb5Fc579115071764c7423A4f12eDde41f106Ed # Specific Addresses /!\ We are not supporting EIP 3770 yet, if the address is not starting by 0x, this will panic by safety measure.
events:
  - signature: ExecutionFailure(bytes32,uint256) # List of the events to watch for the addresses.
  - signature: ExecutionSuccess(bytes32,uint256) # List of the events to watch for the addresses.
```

### Execution

To run it:

```bash

go run ../cmd/monitorism global_events --nickname MySuperNickName --l1.node.url https://localhost:8545 --PathYamlRules /tmp/Monitorism/op-monitorism/global_events/rules/rules_mainnet_L1 --loop.interval.msec 12000

```
