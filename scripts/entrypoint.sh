#!/bin/bash

set -e

if [ -z "${RUNNER_NAME}" ]; then
  echo "RUNNER_NAME must be set" 1>&2
  exit 1
fi

if [ -z "${RUNNER_TOKEN}" ]; then
  echo "RUNNER_TOKEN must be set" 1>&2
  exit 1
fi

if [ -n "${RUNNER_LABELS}" ]; then
  LABEL_ARG="--labels ${RUNNER_LABELS}"
fi

cd /runner
mkdir -p _work
./config.sh --unattended --replace --name "${RUNNER_NAME}" --url "https://github.com/${RUNNER_ORG}" --token "${RUNNER_TOKEN}" --work /runner/_work ${LABEL_ARG}

# TODO: run placemat

./bin/runsvc.sh --once

if [ ! kubectl get node >/dev/null 2>&1 ]; then
    echo "not in kubernetes cluster, so exit"
    exit 0
fi

# delete your self
if [ -f /tmp/failed ]; then
    echo "Label pods with current time + 1m"
    kubectl label pod ${RUNNER_NAME} delete-at=$(date -d "1 minutes" +%Y%m%d%H%M%S)
else
    echo "Label pods with current time"
    kubectl label pod ${RUNNER_NAME} delete-at=$(date +%Y%m%d%H%M%S)
fi

echo "Wait until delete-at"
while true
do
    DELETE_AT=$(kubectl get pod ${RUNNER_NAME} -o jsonpath='{.metadata.labels.delete-at}')
    NOW=$(date +%Y%m%d%H%M%S)
    if [ -n "${DELETE_AT}" ] && [ ${NOW} -gt ${DELETE_AT} ]; then
        echo "Delete ${RUNNER_NAME}"
        kubectl delete pod ${RUNNER_NAME}
    fi
    echo "sleeping..."
    sleep 30
done
