#!/bin/sh

set -e

if [ -z "${POD_NAME}" ]; then
  echo "POD_NAME must be set" 1>&2
  exit 1
fi

if [ -z "${POD_NAMESPACE}" ]; then
  echo "POD_NAMESPACE must be set" 1>&2
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

# TODO: run startup script

cd /runner
mkdir -p _work
./config.sh --unattended --replace --name "${POD_NAME}" --url "https://github.com/${RUNNER_ORG}/${RUNNER_REPO}" --token "${RUNNER_TOKEN}" --work /runner/_work
./bin/runsvc.sh

if [ -z "${EXTEND_DURATION}" ]; then
  EXTEND_DURATION="20m"
fi

if [ -f /tmp/extend ]; then
  echo "Annotate pods with the time ${EXTEND_DURATION} later"
  deltime-annotate ${POD_NAME} -n ${POD_NAMESPACE} -a ${EXTEND_DURATION}
  EXTEND=true
else
  echo "Annotate pods with current time"
  deltime-annotate ${POD_NAME} -n ${POD_NAMESPACE}
  EXTEND=false
fi

if [ -f /tmp/failure ]; then
  JOB_RESULT=failure
elif [ -f /tmp/cancelled ]; then
  JOB_RESULT=cancelled
elif [ -f /tmp/success ]; then
  JOB_RESULT=success
else
  JOB_RESULT=unknown
fi

if [ -n "${SLACK_AGENT_SERVICE_NAME}" ]; then
  echo "Send an notification to slack"
  slack-agent-client ${POD_NAME} ${JOB_RESULT} \
    -n ${POD_NAMESPACE} \
    -s "http://${SLACK_AGENT_SERVICE_NAME}" \
    -e=${EXTEND}
else
  echo "Skip sending an notification to slack because SLACK_AGENT_SERVICE_NAME is blank"
fi

sleep infinity
