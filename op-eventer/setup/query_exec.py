import yaml
from string import Template
import sys
import os
from pprint import pprint
from google.cloud import bigquery
import subprocess
import json
import signal
import time

home = os.path.expanduser("~")
credentials_path = f"{
    home}/.config/gcloud/application_default_credentials.json"

# Set credentials environment variable
os.environ['GOOGLE_APPLICATION_CREDENTIALS'] = credentials_path


def signal_handler(sig, frame):
    print('\nExiting gracefully...')
    sys.exit(0)


def run_query(query):
    client = bigquery.Client()

    query_job = client.query(query)
    results = query_job.result()

    return results


def send_alert(script_path, source, metric_name, priority, fields):
    subprocess.run(f"bash {script_path} '{source}' 'Query={metric_name},Priority={
                   priority},{fields} metric=1'", shell=True)


def process_queries(config):

    # Get all Parameters
    Parameters = config['Parameters']
    # Get all QueryTemplate
    QueriesTemplate = config['QueriesTemplate']
    # Process Checks
    Checks = config['Checks']

    processed_queries = []
    for check in Checks:
        script_path = check['Path']
        queries = check['Queries']
        for query in queries:
            query_name = query
            query_parameters_name = queries[query_name]['Parameters']
            query_template_name = queries[query_name]['Query']
            priority = queries[query_name]['Priority']
            source = queries[query_name]['Source']

            query_parameters = Parameters[query_parameters_name]
            query_template = Template(QueriesTemplate[query_template_name])
            parsed_query = query_template.safe_substitute(query_parameters)
            processed_queries.append((query_name, parsed_query))

            results = run_query(parsed_query)

            for row in results:
                field_string = " ".join(
                    f"{field}={getattr(row, field)}" for field in row.keys())

                print(f"\n{field_string} \n")
                send_alert(script_path, source, query_name,
                           priority, field_string)

    return processed_queries


if __name__ == "__main__":
    signal.signal(signal.SIGINT, signal_handler)
    file_path = sys.argv[1]

    while True:
        with open(file_path, 'r') as file:
            config = yaml.safe_load(file)

        for name, query in process_queries(config):
            print(f"\nProcessing {name} ")
            print(query)

        time.sleep(10)
