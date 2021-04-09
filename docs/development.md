Development guide
=================

Testing
-------

There are 2 kinds of test included in this repository.

- [envtest](https://github.com/kubernetes-sigs/controller-runtime/tree/master/pkg/envtest):
  Test against a real API server without container runtime.
- kindtest: Test on a real Kubernetes cluster with [kind](https://kind.sigs.k8s.io/docs/user/quick-start/).

### kindtest

Kindtest is normally used for the end-to-end testing purpose, but this controller is
difficult to test in some parts and some parts of the code are not tested intentionally.

What kindtest covers is:

- Runner `Pod`s are registered to GitHub Actions on [`github.com/neco-test/github-actions-controller-ci`](https://github.com/neco-test/github-actions-controller-ci).
- GitHub Actions workflows run on the `Pod`s.
- Runner `Pod`s send messages to Slack agent.
- The controller can delete runner `Pod`s with deletion time annotations.
- The controller can delete runner registrations of unexisting `Pod`s from GitHub
  Actions.

Note that the GitHub App needs an **Actions Write** permission in addition to
[this](./user-manual.md#how-to-create-github-app) to trigger workflow dispatch events.

What kindtest does not cover is:

- Slack agent `notifier` sends messages to Slack.
  - In kindtest, `notifier` does receive requests from runner `Pod`s, but it does
    not send messages to Slack.
- Slack agent `extender` runs WebSocket and extends a runner `Pod`'s lifetime.

So, you might need to test the Slack agent behavior manually if you make a change
on Slack agent.

### Test `notifier` manually

First, create Slack App following [user manual](./user-manual.md#how-to-create-slack-app).

Then, run a server with the following commands:

```bash
## notifier server
$ export SLACK_WEBHOOK_URL=<your Slack Webhook URL>
$ go run ./cmd/slack-agent/ notifier
```

You can test both the failure and success messages by actually sending them:

```bash
## notifier client
# failure
$ go run ./cmd/slack-agent/ client \
    -n sample-ns sample-pod \
    -w Build -i 123 \
    -o cybozu-go -r github-actions-controller \
    -b sample-branch \
    --failed

# success
$ go run ./cmd/slack-agent/ client \
    -n sample-ns sample-pod \
    -w Build -i 123 \
    -o cybozu-go -r github-actions-controller \
    -b sample-branch
```

### Test `extender` manually

After sending a failure message with `notifier`, run a `extender` server:

```bash
## notifier server
$ export SLACK_APP_TOKEN=<your Slack App Token>
$ export SLACK_BOT_TOKEN=<your Slack Bot Token>
$ export SLACK_WEBHOOK_URL=<your Slack Webhook URL>
$ go run ./cmd/slack-agent/ extender -d
```

Then, click the button on the Slack message, and check if a receiving log appears
on the terminal.

How to run GitHub Actions controller for development
----------------------------------------------------

If you need to run the controller on your local environment, this is the easiest
way to do that.

```bash
$ export GITHUB_APP_ID=<your GitHub App ID>
$ export GITHUB_APP_INSTALLATION_ID=<your GitHub installation ID>
$ export GITHUB_APP_PRIVATE_KEY_PATH=<path to your .pem file>
$ export SLACK_APP_TOKEN=dummy
$ export SLACK_BOT_TOKEN=dummy
$ export SLACK_WEBHOOK_URL=dummy

$ make start-kind
$ make images load
$ make prepare
```
