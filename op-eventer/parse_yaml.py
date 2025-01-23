import yaml
from string import Template
import sys
from pprint import pprint


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
        path = check['Path']
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

    return processed_queries


if __name__ == "__main__":
    for name, query in process_queries(sys.argv[1]):
        print(f"\n=== {name} ===")
        print(query)
