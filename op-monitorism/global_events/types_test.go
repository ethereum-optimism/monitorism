package global_events

import (
	"gopkg.in/yaml.v3"
	"testing"
)

func TestYamlToConfiguration(t *testing.T) {
	data := `
configuration:
  - version: "1.0"
    name: "Safe Watcher"
    priority: "P0"
    addresses: "We are not supporting EIP 3770 yet, if the address is not starting by '0x', this will panic by safety measure."
    events:
      - signature: "ExecutionFailure(bytes32,uint256)"
      - signature: "ExecutionSuccess(bytes32,uint256)"
  - version: "1.0"
    name: "Safe Watcher"
    priority: "P0"
    addresses: "We are not supporting EIP 3770 yet, if the address is not starting by '0x', this will panic by safety measure."
    events:
      - signature: "ExecutionFailure(bytes32,uint256)"
      - signature: "ExecutionSuccess(bytes32,uint256)"
`

	var config GlobalConfiguration
	err := yaml.Unmarshal([]byte(data), &config)
	if err != nil {
		t.Errorf("error: %v", err)
	}

	// Print the config to see if it's correct
	t.Logf("%+v\n", config)
}
