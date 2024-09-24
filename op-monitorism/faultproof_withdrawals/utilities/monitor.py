from mitmproxy import http, ctx
import datetime
import json
from pprint import pprint
import urllib.parse
        
def load(loader):
    # Add an option for the target URL
    ctx.options.add_option(
        name="target_url",  # The name of the option
        typespec=str,       # The type of the option
        default="",         # Default value
        help="Target URL to forward the requests"
    )
    ctx.options.add_option(
        name="log_file_name",  # The name of the option
        typespec=str,       # The type of the option
        default="",         # Default value
        help="Name"
    )
    

def response(flow: http.HTTPFlow) -> None:

    with open(ctx.options.log_file_name, "a") as log_file:
        body= json.loads(flow.response.get_text())
        body["target_url"]=flow.request.headers["Host"]
        log_file.write(f"{body}\n")



def request(flow: http.HTTPFlow) -> None:
    # Retrieve the target URL from the options
    target_url = ctx.options.target_url

    if target_url:
        # Parse the target URL to extract the host, scheme, and port
        parsed_url = urllib.parse.urlparse(target_url)
        
        flow.request.host = parsed_url.hostname

        flow.request.scheme = parsed_url.scheme
        flow.request.port = parsed_url.port or (443 if parsed_url.scheme == "https" else 80)

        # Optionally modify the Host header
        flow.request.headers["Host"] = parsed_url.hostname

        # Convert timestamp to a human-readable format
        timestamp = datetime.datetime.fromtimestamp(flow.request.timestamp_start).strftime('%Y-%m-%d %H:%M:%S')

        with open(ctx.options.log_file_name, "a") as log_file:
            body= json.loads(flow.request.get_text())
            body["target_url"]=parsed_url
            log_file.write(f"{body}\n")

