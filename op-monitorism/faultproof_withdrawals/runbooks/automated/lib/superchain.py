import requests
import toml
from typing import Dict

def get_superchain_file(l1_name: str, l2_superchain_name: str) -> Dict[str, str]:
    """
    Fetches the superchain file from the given l1_name and l2_superchain_name.

    Args:
        l1_name (str): The name of the L1 chain.
        l2_superchain_name (str): The name of the L2 superchain.

    Returns:
        dict: The parsed superchain file as a dictionary.

    Raises:
        requests.exceptions.RequestException: If there is an error making the HTTP request.
        toml.TomlDecodeError: If there is an error parsing the TOML file.
    """
    superchain_file = f"https://raw.githubusercontent.com/ethereum-optimism/superchain-registry/main/superchain/configs/{l1_name}/{l2_superchain_name}.toml"
    try:
        response = requests.get(superchain_file)
        response.raise_for_status()
        superchain = toml.loads(response.text)
        return superchain
    except requests.exceptions.RequestException as e:
        raise e
    except toml.TomlDecodeError as e:
        raise e