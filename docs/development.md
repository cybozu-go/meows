# Development guide

## Testing

There are 2 kinds of test included in this repository.

- [envtest](https://github.com/kubernetes-sigs/controller-runtime/tree/master/pkg/envtest):
  Test against a real API server without container runtime.
- kindtest: Test on a real Kubernetes cluster with [kind](https://kind.sigs.k8s.io/docs/user/quick-start/).

### kindtest

Kindtest is normally used for the end-to-end testing purpose, but this controller is
difficult to test in some parts and some parts of the code are not tested intentionally.

What kindtest covers is:

- Runner `Pod`s are registered to GitHub Actions on a test repository.
  - At present, the test repository is a fixed one (`github.com/neco-test/meows-ci`).
- GitHub Actions workflows run on the `Pod`s.
- Runner `Pod`s send messages to Slack agent.
- Slack agent sends messages to Slack.
- The controller can delete runner `Pod`s with deletion time exposed by API.
- The controller can delete runner registrations of unexisting `Pod`s from GitHub Actions.

What kindtest does not cover is:

- Slack agent extends a runner `Pod`'s lifetime.

So, you might need to test the Slack agent behavior manually if you make a change on Slack agent.

In order to run the kindtest, you need to prepare as follows.

1. Create GitHub App and Slack App. Please follow [user manual](./user-manual.md).
2. Get the write access permission to the test repository.
    - In the kindtest, test branch and workflow files are generated and pushed dynamically.
3. Set the [remove](../kindtest/workflows/remove.yaml) workflow to the test repository.

You can run the kindtest as following.

1. Create secret files for kindtest.

    ```console
    $ vi .secret.private-key.pem
    # Save your GitHub App private key in this file.

    $ vi .secret.env.sh
    # Save env variables as following.
    #
    # export GITHUB_APP_ID=<your GitHub App ID>
    # export GITHUB_APP_INSTALLATION_ID=<your GitHub App Installation ID>
    # export SLACK_CHANNEL=#<your Slack Channel>
    # export SLACK_APP_TOKEN=<your Slack App Token>
    # export SLACK_BOT_TOKEN=<your Slack Bot Token>
    ```

2. Install tools.

    ```bash
    make setup
    ```

3. Run kindtest.

    ```bash
    # Start kind cluster.
    make -C kindtest start

    # Run test on kind.
    make -C kindtest test

    # Stop kind cluster.
    make -C kindtest stop
    ```

### Run slack agent manually

Then, run a server with the following commands:

```bash
# Run server process
export SLACK_CHANNEL=#<your Slack Channel>
export SLACK_APP_TOKEN=<your Slack App Token>
export SLACK_BOT_TOKEN=<your Slack Bot Token>
go run ./cmd/slack-agent -d
```

You can test both the failure and success messages by actually sending them:

```bash
# client
cat <<EOF > /tmp/github.env
{
  "actor": "user",
  "git_ref": "branch-name",
  "job_id": "job",
  "pull_request_number": 123,
  "repository": "owner/repo",
  "run_id": 999999,
  "run_number": 987,
  "workflow_name": "Work flow"
}
EOF

# success
go run ./cmd/meows slackagent send pod success -f /tmp/github.env

# failure
go run ./cmd/meows slackagent send pod failure --extend -f /tmp/github.env
```

Then, click the button on the Slack message, and check if a receiving log appears
on the terminal.

## How to run meows for development

If you need to run the controller on your local environment, this is the easiest way to do that.
You can reuse the token for the test repository, which is prepared for CI.
But please be careful that your local environment steals the job that is expected to run on a node created in CI and might cause a failure on CI.

```bash
# Create secret files for kindtest.
vi .secret.private-key.pem
vi .secret.env.sh

make -C kindtest start
make -C kindtest bootstrap
```
