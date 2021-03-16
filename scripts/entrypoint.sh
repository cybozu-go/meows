#!/bin/sh

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

cd /runner
mkdir -p _work
./config.sh --unattended --replace --name "${RUNNER_NAME}" --url "https://github.com/${RUNNER_ORG}/${RUNNER_REPO}" --token "${RUNNER_TOKEN}" --work /runner/_work

# TODO: run placemat

./bin/runsvc.sh

/usr/local/bin/wait.sh
