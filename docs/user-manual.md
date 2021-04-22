User Manual
===========

How to create GitHub App
------------------------

GitHub Actions controller is responsible for the registering/deregistering `Pod`s
to/from a GitHub Actions runner and the controller requires to have a GitHub App
secret.

Please create a GitHub App first on the GitHub page following [the official documentation](https://docs.github.com/en/developers/apps/creating-a-github-app).
Here are the minimal changes from the default setting on the registration page:

- Fill **GitHub Apps Name**
- Fill **Homepage URL**
- Uncheck `Active` under **Webhook** section
- Set **Administration** `Read & Write` permission to the repository scope

Then, you are redirected to the **General** page and what you should do is:

1. Click `Generate a private key` and downloads the generated private key.
1. Keep `App ID` shown on the top of the page somewhere.

Next, you should proceed to the Install App page from the sidebar and click the
`Install` button. You are asked to give the permission to `All repositories`
or `Only select repositories`. `Only select repositories` is recommended because
the permission is very strong. You should decide the scope wide enough depending on
how you use this controller.

Finally, you should get the installation ID from the URL of which page you are
redicted to. The URL should look like `https://github.com/organizations/cybozu-go/settings/installations/12345`
and `12345` is your installation ID.

How to deploy GitHub Actions controller
---------------------------------------

You should deploy the controller `Deployment` with a `Secret` resource which
contains the GitHub App private key you created. The command below creates the
`Secret` resource.

```bash
$ GITHUB_APP_ID=<your GitHub App ID>
$ GITHUB_APP_INSTALLATION_ID=<your GitHub App Installation ID>
$ GITHUB_APP_PRIVATE_KEY_PATH=<Path of a GitHub App private key file>

$ kubectl create secret generic github-app-secret \
    -n actions-system \
    --from-literal=app-id=${GITHUB_APP_ID} \
    --from-literal=app-installation-id=${GITHUB_APP_INSTALLATION_ID} \
    --from-file=app-private-key=${GITHUB_APP_PRIVATE_KEY_PATH}
```

In addition to this, the admission webhook requires a TLS certificate.
You should use [`github.com/jetstack/cert-manager`](https://github.com/jetstack/cert-manager)
or create a certificate by yourself.

This document does not give you a comprehensive list of what you shold deploy.
Please refer to the manifests under `config/` for further information.

How to create Slack App
-----------------------

Slack agent notifies users whether CI succeeds or not and receives messages to
extend a `Pod` lifetime.
So, users have to prepare a Slack App to send messages to and run a WebSocket client
to watch button events.

Here's a procedure for how to configure the Slack App.

1. Go to [this](https://api.slack.com/apps) page.
1. Click the **Create New App** button.
   - Choose **From scratch**.
   - Fill the application name field and choose a Slack workspace.
1. Go to **Socket Mode** from the sidebar.
   - Enable **Enable Socket Mode**.
   - Create App-Level Token on the windows coming up and keep the generated App Token.
1. Go to **OAuth & Permissions** from the sidebar.
   - Add the `chat:write` permission under **Bot Token Scopes**.
   - Click **Install(Reinstall) to Workspace** and (re)install the bot in your desired channel.
   - Keep **Bot User OAuth Token**.
1. Go to **Beta features** from the sidebar.
   - Enable **Time picker element**.
1. Open your Slack desktop app and go to your desired channel.
   - Click the `i` button on the top right corner.
   - Click **more** and then **Add apps**.
   - Add the created Slack App to the channel.

How to deploy Slack agent
-------------------------

You should deploy the Slack agent `Deployment` with a `Secret` resource which
contains the Slack App tokens you created. The command below creates the `Secret`
resource.

```bash
$ RUNNER_NAMESPACE=<Runner Namespace>
$ SLACK_CHANNEL=#<your Slack Channel>
$ SLACK_APP_TOKEN=<your Slack App Token>
$ SLACK_BOT_TOKEN=<your Slack Bot Token>

$ kubectl create secret generic slack-app-secret \
    -n ${RUNNER_NAMESPACE} \
    --from-literal=SLACK_CHANNEL=${SLACK_CHANNEL} \
    --from-literal=SLACK_APP_TOKEN=${SLACK_APP_TOKEN} \
    --from-literal=SLACK_BOT_TOKEN=${SLACK_BOT_TOKEN}
```

Please refer to the manifest `config/agent/manifests.yaml` for detail.

RunnerPool Custom Resource Example
----------------------------------

This is an example of the `RunnerPool` custom resource.

```yaml
apiVersion: actions.cybozu.com/v1alpha1
kind: RunnerPool
metadata:
  name: runnerpool-sample
spec:
  repositoryName: repository-sample
  slackAgentServiceName: slack-agent
  replicas: 3
  selector:
    matchLabels:
      app: actions-runner
  template:
    metadata:
      labels:
        app: actions-runner
    spec:
      containers:
      - name: runner
        image: quay.io/cybozu/actions-runner:latest
      serviceAccountName: runner
```

The controller creates a `Deployment` based on `template` in `RunnerPool`.
It modifies `template` just to satisfy the minimal requirement to run GitHub Actions.

The controller is mainly responsible for:

- Add the GitHub organization name specified with the controller CLI's option and
  the repository name defined in the `RunnerPool` manifest to `metadata.labels`.
- Add environment variables needed to run `entrypoing.sh`.
  `entrypoing.sh` is a default command for the runner container and contains
  [`github.com/actions/runner`](https://github.com/actions/runner).

You are responsible for:

- A `Pod` created with `RunnerPool` annotates itself with what time to delete the
  `Pod` when CI finishes and needs the `patch` role for `Pod`. The corresponding
  `ServiceAccount` is not created by the controller for now, so please create it
  by yourself.  Please refer to the manifest `config/samples/manifests.yaml` for
  detail.
- `slackAgentServiceName` is a `Service` name that can be resolved inside a
  Kubernetes cluster, so a `Service` name is acceptable. If `slackAgentServiceName`
  is omitted, the `Pod`s do not send notifications to Slack.

After running `Pod`s, you can check if the runners are actually registered to
GitHub on the **Actions** page under each repository's **Settings**
(e.g. https://github.com/cybozu-go/github-actions-controller/settings/actions).

How to use self-hosted runners
------------------------------

GitHub Actions controller provides the following commands, you have to execute these commands in your workflow.

- `job-started`
    - Notify a runner pod that a workflow has been started.
      You need to call this command at the start of the workflow.

- `job-success`, `job-cancelled`, `job-failure`
    - Notify a runner pod that the result of a workflow.
      You need to call these commands at the end of the workflow with [Job status check functions](https://docs.github.com/en/actions/reference/context-and-expression-syntax-for-github-actions#job-status-check-functions).

Here is an example of a workflow definition.

```yaml
name: Main
on:
  pull_request:
  push:
    branches:
      - 'main'

jobs:
  build:
    name: build
    runs-on: self-hosted
    steps:
      - run: job-started

      - run: ...
      - run: ...
      - run: ...

      - if: success()
        run: job-success
      - if: cancelled()
        run: job-cancelled
      - if: failure()
        run: job-failure
```

How to extend GitHub Actions jobs
---------------------------------

![failure message](./images/slack_failure.png)

Choose the time in UTC you want to extend the `Pod` by and click `Extend`.
This button can be clicked multiple times if the `Pod` still exists.

How to recreate Pod when update
-------------------------------

Runners download new runners when a new release is out, but we would still face a
situation that we have to update the runner `Pod` image.

Here are some small technique to decrease the downtime in your CI.

- Restart all the `Pod`s at the same time by setting update strategy `Recreate`.
- Dare use `:latest` image and let the `Pod`s upgrade by themeself after a job
  is scheduled and executed.
  The official document says that the `:latest` tag should be avoided in production
  because it's harder to track which version is running, but self-hosted runners
  do not run in production and the runners upgrade itself in the first place.

You are still forced to kill `Pod`s to update them, but hopefully these help you
decrease such opportunities.
