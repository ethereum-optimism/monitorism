#!/bin/bash -eo pipefail

# Get script's absolute path 
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
VENV_DIR="$SCRIPT_DIR/../../.venv"
MONITORING_DIR="$SCRIPT_DIR/../monitoring"
query_exec="$SCRIPT_DIR/query_exec.py"

# Activate virtual environment
source "$VENV_DIR/bin/activate"

export GOOGLE_CLOUD_PROJECT=oplabs-dev-security
# Check where credentials actually are
CRED_FILE=$(gcloud info --format="get(config.paths.global_config_dir)")/application_default_credentials.json

if [ ! -f "$CRED_FILE" ]; then
    echo "No application default credentials found. Running gcloud auth..."
    gcloud auth application-default login
else
    if ! GOOGLE_APPLICATION_CREDENTIALS="$CRED_FILE" gcloud auth application-default print-access-token &>/dev/null; then
        echo "Invalid credentials. Running gcloud auth..."
        gcloud auth application-default login
    fi
fi

python $query_exec $MONITORING_DIR/monitoring.yaml 
