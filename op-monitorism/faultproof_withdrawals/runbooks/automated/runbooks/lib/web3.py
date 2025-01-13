from web3 import Web3
from typing import List, Any
import json
from datetime import datetime, timezone
import urllib3
# Disable warnings for insecure HTTPS requests
urllib3.disable_warnings(urllib3.exceptions.InsecureRequestWarning)

# Custom request settings to ignore SSL certificate verification
request_kwargs = {
    'verify': False  # Disable SSL verification
}

class Web3Utility:

    def __init__(self, l1_geth_url: str,l2_op_geth_url: str,l2_op_node_url: str,OptimismPortal_abi_path:str, OptimismPortalProxy:str,ignore_certificate: bool=False):
        self.OptimismPortal_abi_path=OptimismPortal_abi_path
        self.OptimismPortalProxy=OptimismPortalProxy
        self.l2_op_geth_url=l2_op_geth_url
        self.l2_op_node_url=l2_op_node_url
        self.l1_geth_url=l1_geth_url

        if ignore_certificate:
            self.web3 = Web3(Web3.HTTPProvider(l1_geth_url,request_kwargs=request_kwargs))
        else:
            self.web3 = Web3(Web3.HTTPProvider(l1_geth_url))
        if not self.web3.is_connected():
            print("Failed to connect to Web3 provider.")

        contract_abi = None
        with open( self.OptimismPortal_abi_path, 'r') as file:
            contract_abi = json.load(file)
        self.contract_abi = contract_abi

        self.contract = self.web3.eth.contract(address=self.OptimismPortalProxy, abi=contract_abi)


    def find_latest_withdrawal_event(self, batch_size: int = 1000) -> List[Any]:
        """
        Fetches the latest event from the OptimismPortal contract by searching in increments of `batch_size` blocks.

        Args:
            abi_path (str): The path to the contract ABI.
            contract_address (str): The address of the OptimismPortal contract.
            batch_size (int, optional): Number of blocks to search at a time. Defaults to 1000.

        Returns:
            Dict: A dictionary containing the latest event log and its timestamp.

        Raises:
            Exception: If there is an error fetching the events or if no events are found.
        """

        contract=self.contract
        latest_block = self.web3.eth.block_number
        current_block = latest_block

        # Search in batches of `batch_size` blocks
        while current_block > 0:
            from_block = max(0, current_block - batch_size)
            try:
                logs = contract.events.WithdrawalProvenExtension1().get_logs(from_block=from_block, to_block=current_block)
                if logs:
                    # Return the latest event found along with its timestamp
                    last_log = logs[-1]
                    block_number = last_log["blockNumber"]
                    timestamp_formatted = self.get_block_timestamp(block_number)
                    return {"log": last_log, "timestamp": timestamp_formatted}
            except Exception as e:
                print(f"Error fetching logs between blocks {from_block} and {current_block}: {str(e)}")
            
            # Move the search window to the previous `batch_size` block range
            current_block = from_block

        raise Exception("No WithdrawalProven event found within the searched block range.")


    def get_block_timestamp(self, blockNumber: int):
            """
            Fetches the timestamp of a block.

            Args:
                web3 (Web3): An instance of the Web3 class.
                blockNumber (int): The block number.

            Returns:
                dict: A dictionary containing the block number, timestamp, time since the last withdrawal, and formatted timestamp.
            """

            block=self.web3.eth.get_block(blockNumber)
            timestamp=block["timestamp"]

            ret = {
                "blockNumber": blockNumber,
                "timestamp": timestamp,
                "time_since_last_withdrawal": f"{datetime.now(timezone.utc) - datetime.fromtimestamp(timestamp, tz=timezone.utc)}",
                "formatted_timestamp": f"{datetime.fromtimestamp(timestamp, tz=timezone.utc).strftime('%Y-%m-%d %H:%M:%S')}",
            }    
            return ret
