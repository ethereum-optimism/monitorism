from web3 import Web3
from typing import List, Any
import json
from datetime import datetime, timezone
import urllib3
import os
import requests
# Disable warnings for insecure HTTPS requests
urllib3.disable_warnings(urllib3.exceptions.InsecureRequestWarning)

# Custom request settings to ignore SSL certificate verification
request_kwargs = {
    'verify': False  # Disable SSL verification
}

class Web3Utility:

    def __init__(self, l1_geth_url: str,l2_op_geth_url: str, l2_op_node_url: str,abi_folder_path:str, OptimismPortalProxyAddress:str,ignore_certificate: bool=False):
        self.OptimismPortal_abi_path=os.path.join(abi_folder_path,"OptimismPortal2.json")
        self.FaulDisputeGame_abi_path=os.path.join(abi_folder_path,"FaultDisputeGame.json")
        self.L2ToL1MessagePasser_abi_path=os.path.join(abi_folder_path,"L2ToL1MessagePasser.json")

        self.OptimismPortalProxyAddress=OptimismPortalProxyAddress
        self.l2_op_node_url = l2_op_node_url
        self.ignore_certificate=ignore_certificate

        if ignore_certificate:
            self.l1_geth = Web3(Web3.HTTPProvider(l1_geth_url,request_kwargs=request_kwargs))
            self.l2_op_geth = Web3(Web3.HTTPProvider(l2_op_geth_url,request_kwargs=request_kwargs))
        else:
            self.l1_geth = Web3(Web3.HTTPProvider(l1_geth_url))
            self.l2_op_geth = Web3(Web3.HTTPProvider(l2_op_geth_url))
    
        if not self.l1_geth.is_connected():
            print(f"Failed to connect to Web3 l1_geth_url {l1_geth_url} provider.")
        if not self.l2_op_geth.is_connected():
            print(f"Failed to connect to Web3 l2_op_geth_url {l2_op_geth_url} provider.")

      
        OptimismPortal_contract_abi = None
        with open( self.OptimismPortal_abi_path, 'r') as file:
            OptimismPortal_contract_abi = json.load(file)
        self.OptimismPortal_contract_abi = OptimismPortal_contract_abi

        self.OptimismPortal2 = self.l1_geth.eth.contract(address=self.OptimismPortalProxyAddress, abi=OptimismPortal_contract_abi)

        L2ToL1MessagePasser_contract_abi = None
        with open( self.L2ToL1MessagePasser_abi_path, 'r') as file:
            L2ToL1MessagePasser_contract_abi = json.load(file)
        self.L2ToL1MessagePasser_contract_abi = L2ToL1MessagePasser_contract_abi

        self.L2ToL1MessagePasser = self.l2_op_geth.eth.contract(address="0x4200000000000000000000000000000000000016",abi=L2ToL1MessagePasser_contract_abi)

    def get_fault_dispute_game(self, gameProxyAddress:str):
        contract_abi = None
        with open( self.FaulDisputeGame_abi_path, 'r') as file:
            contract_abi = json.load(file)
        FaulDisputeGame = self.l1_geth.eth.contract(address=gameProxyAddress, abi=contract_abi)
        return FaulDisputeGame

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

        contract=self.OptimismPortal2
        latest_block = self.l1_geth.eth.block_number
        current_block = latest_block

        # Search in batches of `batch_size` blocks
        while current_block > 0:
            from_block = max(0, current_block - batch_size)
            try:
                to_block = current_block
                logs = contract.events.WithdrawalProvenExtension1().get_logs(from_block=from_block, to_block=to_block)
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

    def get_withdrawal_proven_extension_1(self,txHash:str):
       
        try:
            txReceipt=self.l1_geth.eth.get_transaction_receipt(txHash)
            logs = self.OptimismPortal2.events.WithdrawalProvenExtension1().process_receipt(txReceipt)
            return logs
        except Exception as e:
            print(f"Error: {str(e)}")



    def get_block_timestamp(self, blockNumber: int):
            """
            Fetches the timestamp of a block.

            Args:
                web3 (Web3): An instance of the Web3 class.
                blockNumber (int): The block number.

            Returns:
                dict: A dictionary containing the block number, timestamp, time since the last withdrawal, and formatted timestamp.
            """

            block=self.l1_geth.eth.get_block(blockNumber)
            timestamp=block["timestamp"]

            ret = {
                "blockNumber": blockNumber,
                "timestamp": timestamp,
                "formatted_timestamp": f"{datetime.fromtimestamp(timestamp, tz=timezone.utc).strftime('%Y-%m-%d %H:%M:%S')}",
            }    
            return ret

    def get_game_data(self,withDrawalHash:str ,proofSubmitter:str):
        if type(withDrawalHash) is str:
            withDrawalHash = bytes.fromhex(withDrawalHash)
        gameProxyAddress,timestamp=self.OptimismPortal2.functions.provenWithdrawals(withDrawalHash,proofSubmitter).call()
        game=self.get_fault_dispute_game(gameProxyAddress)
        l2BlockNumber=game.functions.l2BlockNumber().call()
        rootClaim=game.functions.rootClaim().call()  
     
        sentMessages=self.L2ToL1MessagePasser.functions.sentMessages(withDrawalHash).call()
     
        optimism_outputAtBlock=self.optimism_output_at_block(l2BlockNumber)
     
        return {"gameProxyAddress":gameProxyAddress,"timestamp":timestamp,"l2BlockNumber":l2BlockNumber,"rootClaim":f"0x{rootClaim.hex()}","sentMessages":sentMessages,"optimism_outputAtBlock":optimism_outputAtBlock}
    

    def optimism_output_at_block(self,blockNumber:int):
        # we need to do the equivalent of the following command

        url = self.l2_op_node_url
        block_number_hex = hex(blockNumber)
        headers = {
            "Content-Type": "application/json"
        }
        data = {
            "jsonrpc": "2.0",
            "method": "optimism_outputAtBlock",
            "params": [block_number_hex],
            "id": 1
        }

        # Send the POST request
        response = requests.post(url, headers=headers, data=json.dumps(data),verify=not self.ignore_certificate)

        # Check if the request was successful
        if response.status_code == 200:
            return response.json()["result"]["outputRoot"]
        else:
            raise Exception(f"Error: {response.status_code} - {response.text}")