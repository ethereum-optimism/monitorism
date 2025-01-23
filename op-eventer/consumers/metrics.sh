#!/bin/bash

set -o pipefail -o errexit -o nounset
source ~/op-eventer.env

function log {
  echo $(date "+%Y-%m-%d %H:%M:%S") $*
}

function metric {
   local payload="raffaele_metrics,source=test"

  payload="${payload},event=2 event=1"

  curl \
  	-X POST \
  	-u $METRICS_USER \
  	-H "Content-Type: text/plain" \
  	"$METRICS_URL" \
  	-d "$payload"
}

metric