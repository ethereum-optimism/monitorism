# Makefile
# List of contracts to generate
CONTRACTS_TO_GENERATE := dispute:DisputeGameFactory dispute:FaultDisputeGame l1:OptimismPortal2 l2:L2ToL1MessagePasser


# Default action
build_bindings: 
	rm -rf ./../bindings/*	
	CONTRACTS_TO_GENERATE="$(CONTRACTS_TO_GENERATE)" ./generate_bindings.sh

# Clean up action
clean:
	rm -rf ./tmp

# Set build_bindings as the default action
.DEFAULT_GOAL := build_bindings
