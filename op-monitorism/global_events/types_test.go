package global_events

import (
	"github.com/ethereum/go-ethereum/common"
	"gopkg.in/yaml.v3"
	"testing"
)

const data = `
configuration:
  - version: "1.0"
    name: "BuildLand"
    priority: "P0"
    addresses: 
      - 0x95222290DD7278Aa3Ddd389Cc1E1d165CC4BAfe5
    events:
      - signature: "ExecutionFailure(bytes32,uint256)"
      - signature: "ExecutionSuccess(bytes32,uint256)"
  - version: "1.0"
    name: "NightLand"
    priority: "P2"
    addresses: # We are not supporting EIP 3770 yet, if the address is not starting by '0x', this will panic by safety measure."
    events:
      - signature: "ExecutionFailure(bytes32,uint256)"
      - signature: "ExecutionSuccess(bytes32,uint256)"
`

//	func (G GlobalConfiguration) ReturnEventsMonitoredForAnAddress(target_address common.Address) []Event {
//		for _, config := range G.Configuration {
//			for _, address := range config.Addresses {
//				if address == target_address {
//					return config.Events
//				}
//			}
//		}
//		return []Event{} // no events monitored for this address
//
// Print the config to see if it's correct
func TestReturnEventsMonitoredForAnAddress(t *testing.T) {
	var config GlobalConfiguration
	err := yaml.Unmarshal([]byte(data), &config)
	if err != nil {
		t.Errorf("error: %v", err)
	}
	config.ReturnEventsMonitoredForAnAddress(common.HexToAddress("0x41"))
}

func TestDisplayMonitorAddresses(t *testing.T) {
	var config GlobalConfiguration
	err := yaml.Unmarshal([]byte(data), &config)
	if err != nil {
		t.Errorf("error: %v", err)
	}
	config.DisplayMonitorAddresses()
}

func TestYamlToConfiguration(t *testing.T) {

	var config GlobalConfiguration
	err := yaml.Unmarshal([]byte(data), &config)
	if err != nil {
		t.Errorf("error: %v", err)
	}
}
