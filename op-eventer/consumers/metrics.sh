#!/bin/bash

set -o pipefail -o errexit -o nounset
source ~/op-eventer.env

function log {
  echo $(date "+%Y-%m-%d %H:%M:%S") $*
}

function metric {
  metric_name=$1
  metric_source=$2
  metric_value=$3
  payload="${metric_name},${metric_source},${metric_value}"

 echo "Sending metric: $payload"

  curl \
  	-X POST \
  	-u $METRICS_USER \
  	-H "Content-Type: text/plain" \
  	"$METRICS_URL" \
  	-d "$payload"
}
value=$1
metric "raffaele_metrics" "source=test" "${value} metric=1"
