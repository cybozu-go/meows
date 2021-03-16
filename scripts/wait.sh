#!/bin/sh

set -e

if [ -z "${RUNNER_NAME}" ]; then
  echo "RUNNER_NAME must be set" 1>&2
  exit 1
fi

if [ -z "${EXTEND_MINUTES}" ]; then
  EXTEND_MINUTES=20
fi

if [ -f /tmp/failed ]; then
    echo "Label pods with current time + ${EXTEND_MINUTES}m"
    kubectl annotate pods ${RUNNER_NAME} --overwrite actions.cybozu.com/deletedAt=$(date -Iseconds -u -d "${EXTEND_MINUTES} minutes")
else
    echo "Label pods with current time"
    kubectl annotate pods ${RUNNER_NAME} --overwrite actions.cybozu.com/deletedAt=$(date -Iseconds -u)
fi
sleep infinity

