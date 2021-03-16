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

if [ -z "${RUNNER_ORG}" ]; then
  echo "RUNNER_ORG must be set" 1>&2
  exit 1
fi

if [ -z "${RUNNER_REPO}" ]; then
  echo "RUNNER_REPO must be set" 1>&2
  exit 1
fi

echo "https://github.com/${RUNNER_ORG}/${RUNNER_REPO}"

cd /runner
mkdir -p _work
./config.sh --unattended --replace --name "${RUNNER_NAME}" --url "https://github.com/${RUNNER_ORG}/${RUNNER_REPO}" --token "${RUNNER_TOKEN}" --work /runner/_work

# TODO: run placemat

./bin/runsvc.sh --once

if [ -f /tmp/failed ]; then
    echo "Label pods with current time + 20m"
    kubectl annotate pods ${RUNNER_NAME} --overwrite delete-at=$(date -d "20 minutes")
else
    echo "Label pods with current time"
    kubectl annotate pods ${RUNNER_NAME} --overwrite delete-at=$(date)
fi
sleep infinity

