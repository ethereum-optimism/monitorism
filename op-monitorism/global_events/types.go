package global_events

import (
	"fmt"
	"gopkg.in/yaml.v3"
	"os"
)

type EventTopic struct {
	Index  int      `yaml:"index"`
	Values []string `yaml:"values"`
}

type Event struct {
	_4bytes   string // That is the 4 bytes of the event signature, will be generated from the Event.Signature function just below this.
	Signature string `yaml:"signature"` // That is the name of the function like "Transfer(address,address,uint256)"
	// Topics    []EventTopic `yaml:"topics,omitempty"`
}

type Configuration struct {
	Version   string   `yaml:"version"`
	Name      string   `yaml:"name"`
	Priority  string   `yaml:"priority"`
	Addresses []string `yaml:"addresses"`
	Events    []Event  `yaml:"events"`
}

type MonitoringAddress struct {
	Address string  `yaml:"addresses"`
	Events  []Event `yaml:"events"`
}

func ReadYamlFile(filename string) Configuration {
	var config Configuration
	data, err := os.ReadFile(filename)
	if err != nil {
		fmt.Println("Error reading YAML file:", err)
		panic("Error reading YAML")
	}
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		fmt.Println("Error reading YAML file:", err)
		panic("Error reading YAML")

	}
	return config
}

// fromConfigurationToAddress take the configuration yaml and create the `monitoringAddresses` array
// that will contains all the addresses that will monitore.
func fromConfigurationToAddress(config Configuration) []MonitoringAddress {
	var monitoringAddresses []MonitoringAddress
	for _, address := range config.Addresses {
		event_with_4bytes := make([]Event, len(config.Events))
		for _, event := range config.Events {
			event._4bytes = string(FormatAndHash(event.Signature))
			event_with_4bytes = append(event_with_4bytes, event)
		}
		monitoringAddresses = append(monitoringAddresses, MonitoringAddress{Address: address, Events: event_with_4bytes})
	}
	return monitoringAddresses
}
func DisplayMonitorAddresses(monitoringAddresses []MonitoringAddress) {
	println("Monitoring addresses")
	fmt.Printf("Number of addresses: %d\n", len(monitoringAddresses)) //need to put also the number of events
	fmt.Printf("Number of events: %d\n", len(monitoringAddresses[0].Events))
	for _, address := range monitoringAddresses {
		fmt.Println("Address:", address.Address)
		for _, event := range address.Events {
			fmt.Println("Event:", event.Signature, "4bytes:", event._4bytes)
		}
	}
}

// if _, err := toml.DecodeFile("config.toml", &config); err != nil {
//         fmt.Println("Error loading TOML data:", err)
//         os.Exit(1)
//     }
