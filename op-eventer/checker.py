import yaml
from string import Template
import sys
import os
from pprint import pprint
from google.cloud import bigquery
import subprocess
import json

home = os.path.expanduser("~")
credentials_path = f"{
    home}/.config/gcloud/application_default_credentials.json"

# Set credentials environment variable
os.environ['GOOGLE_APPLICATION_CREDENTIALS'] = credentials_path


def run_query(query):
    client = bigquery.Client()

    query_job = client.query(query)
    results = query_job.result()

    return results


def process_queries(file_path):
    with open(file_path, 'r') as file:
        config = yaml.safe_load(file)

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
            query_description = queries[query_name]['Description']
            query_parameters_name = queries[query_name]['Parameters']
            query_template_name = queries[query_name]['Query']

            query_parameters = Parameters[query_parameters_name]
            query_template = Template(QueriesTemplate[query_template_name])
            parsed_query = query_template.safe_substitute(query_parameters)
            processed_queries.append((query_name, parsed_query))
            results = run_query(parsed_query)

            for row in results:
                field_string = " ".join(
                    f"{field}={getattr(row, field)}" for field in row.keys())

                print(f"\n{field_string} \n")
                subprocess.run(f"bash {script_path} '{
                               field_string}'", shell=True)

    return processed_queries


if __name__ == "__main__":
    for name, query in process_queries(sys.argv[1]):
        print(f"\n=== {name} ===")
        print(query)
