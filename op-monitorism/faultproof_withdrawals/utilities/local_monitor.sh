#!/bin/bash
SCRITP_DIR=$(cd $(dirname $0); pwd)
MAKE_DIR=$(cd $SCRITP_DIR/../..; pwd)
cd $MAKE_DIR

l1_geth_port=8080
l2_node_port=8081
l2_geth_port=8082

#check if I should sniff and save the traffic or not from the command line 
if [ "$1" == "sniff" ]; then
    export SNIFF="true"
else
    export SNIFF="false"
fi

#read the environment variables from the .env file
export $(cat "${SCRITP_DIR}/.env.op.sepolia" | xargs)

#kill the mitmdump process if it is running
ps aux | grep mitmdump | grep -v grep | awk '{print $2}' | xargs kill


if [ "$SNIFF" == "true" ]; then
    #if sniffing is enabled, then we need to start the mitmdump process and set properly the environment variables
    l1_geth=${FAULTPROOF_WITHDRAWAL_MON_L1_GETH_URL}
    l2_node=${FAULTPROOF_WITHDRAWAL_MON_L2_OP_NODE_URL}
    l2_geth=${FAULTPROOF_WITHDRAWAL_MON_L2_OP_GETH_URL}
    export FAULTPROOF_WITHDRAWAL_MON_L1_GETH_URL="http://localhost:${l1_geth_port}"
    export FAULTPROOF_WITHDRAWAL_MON_L2_OP_NODE_URL="http://localhost:${l2_node_port}"
    export FAULTPROOF_WITHDRAWAL_MON_L2_OP_GETH_URL="http://localhost:${l2_geth_port}"


    mitmdump -s  "${SCRITP_DIR}/monitor.py" --listen-port ${l1_geth_port} --set ssl_insecure=true --set target_url=$l1_geth --set log_file_name="${SCRITP_DIR}/logs/l1_geth.log" -q &
    MITMDUMP_PID=$!
    echo "mitmdump pid: $MITMDUMP_PID"

    mitmdump -s  "${SCRITP_DIR}/monitor.py" --listen-port ${l2_node_port} --set ssl_insecure=true --set target_url=$l2_node --set log_file_name="${SCRITP_DIR}/logs/l2_node.log" -q &
    MITMDUMP_PID=$!
    echo "mitmdump pid: $MITMDUMP_PID"

    mitmdump -s  "${SCRITP_DIR}/monitor.py" --listen-port ${l2_geth_port} --set ssl_insecure=true --set target_url=$l2_geth --set log_file_name="${SCRITP_DIR}/logs/l2_geth.log" -q &
    MITMDUMP_PID=$!
    echo "mitmdump pid: $MITMDUMP_PID"
fi

make run ARGS="faultproof_withdrawals"
# make run ARGS="withdrawals"


