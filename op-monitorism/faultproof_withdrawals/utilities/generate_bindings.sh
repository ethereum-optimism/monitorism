#!/bin/bash
SCRIPT_DIR=$(cd $(dirname $0); pwd)

# Variables
REPO_URL="https://github.com/ethereum-optimism/optimism.git"
SRC_DIR="packages/contracts-bedrock/src/L1"
TEMP_DIR="$SCRIPT_DIR/tmp/"
OUTPUT_DIR="$SCRIPT_DIR/../bindings"
NOTE_FILE="$OUTPUT_DIR/binding.md"
ARTIFACTS_DIR="$TEMP_DIR/packages/contracts-bedrock/snapshots/abi"
CONTRACTS_ARRAY=($CONTRACTS_TO_GENERATE)
SRC_DIR="$TEMP_DIR/packages/contracts-bedrock/src"

mkdir -p "$TEMP_DIR"

# Ensure abigen is installed
if ! command -v abigen &> /dev/null; then
    echo "abigen could not be found. Installing it..."
    go install github.com/ethereum/go-ethereum/cmd/abigen@latest
fi

# Ensure forge (foundry tool) is installed
if ! command -v forge &> /dev/null; then
    echo "Foundry not installed. Installing foundry..."
    curl -L https://foundry.paradigm.xyz | bash
    foundryup
fi

# Clone the repository to a temporary directory (shallow clone)
echo "Cloning the repository into a temporary directory..."
git clone --depth 1 "$REPO_URL" "$TEMP_DIR"
GIT_COMMIT=$(git rev-parse HEAD)
echo 

echo "## Info" > "$NOTE_FILE"
echo "The bindings in this folder are taken from" >> "$NOTE_FILE"
echo "Repository $REPO_URL cloned at commit $GIT_COMMIT." > "$NOTE_FILE"
echo "" >> "$NOTE_FILE"
echo "This tool is compatible with these bindings. Future binding may break compatibility for those files." >> "$NOTE_FILE"

# Navigate to the cloned repository
cd "$SRC_DIR" || exit

# Compile all contracts using Forge
echo "Compiling all contracts using Forge..."
forge build

# Check if any contracts were compiled
if [ ! -d "$ARTIFACTS_DIR" ]; then
    echo "Foundry compilation failed or no artifacts were found."
    exit 1
fi

# Create the output directory for Go bindings
mkdir -p "$OUTPUT_DIR"

# Loop through specified contracts and generate Go bindings
echo "Generating Go bindings for the specified contracts..."
for directory_contract_name in "${CONTRACTS_ARRAY[@]}"; do

    IFS=":" read -r directory contract_name <<< "$directory_contract_name"
    json_file="$ARTIFACTS_DIR/$contract_name.json"

    if [ ! -s "$json_file" ]; then
        echo "JSON for $json_file is empty or malformed. Skipping."
        continue
    fi

    final_dir="$OUTPUT_DIR/$directory"
    mkdir -p "$final_dir"
    output_file="$final_dir/$contract_name.go"

    # Generate Go bindings using abigen and redirect output to the file
    abigen --abi "$json_file" --pkg ${directory} --type $contract_name --out "$output_file"

    if [ $? -ne 0 ]; then
        echo "Failed to generate Go bindings for $contract_name."
    else
        echo "Generated Go bindings for $contract_name in $output_file."
    fi
done
