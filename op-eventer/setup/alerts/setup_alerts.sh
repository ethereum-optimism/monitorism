set -o pipefail -o errexit -o nounset

source ~/alerts_cred.env
mimirtool rules diff --namespaces=op-eventer ./op-eventer.yaml
mimirtool rules sync --namespaces=op-eventer ./op-eventer.yaml
