# RunnerPool

`RunnerPool` is a custom resource definition (CRD) that represents a pool of
GitHub Actions self-hosted runners.

| Field        | Type                                  | Description                                        |
| ------------ | ------------------------------------- | -------------------------------------------------- |
| `apiVersion` | string                                | APIVersion.                                        |
| `kind`       | string                                | Kind.                                              |
| `metadata`   | [ObjectMeta][]                        | Metadata.                                          |
| `spec`       | [RunnerPoolSpec](#RunnerPoolSpec)     | Specification of desired behavior of `RunnerPool`. |
| `status`     | [RunnerPoolStatus](#RunnerPoolStatus) | Most recently observed status of `RunnerPool`.     |

## RunnerPoolSpec

| Field                  | Type                                                | Description                                                                                                                                                                |
| ---------------------- | --------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `repository`           | string                                              | Repository name. If this field is specified, meows registers pods as repository-level runners.                                                                             |
| `organization`         | string                                              | Organization name. If this field is specified, meows registers pods as organization-level runners.                                                                         |
| `credentialSecretName` | string                                              | Secret name that contains a GitHub Credential. If this field is omitted or the empty string (`""`) is specified, meows uses the default secret name (`meows-github-cred`). |
| `replicas`             | int32                                               | Number of desired runner pods to accept a new job. Defaults to `1`.                                                                                                        |
| `maxRunnerPods`        | int32                                               | Number of desired runner pods to keep. Defaults to `0`. If this field is `0`, it will keep the number of pods specified in `replicas`.                                     |
| `setupCommand`         | []string                                            | Command that runs when the runner pods will be created.                                                                                                                    |
| `slackNotification`    | [SlackNotificationConfig](#SlackNotificationConfig) | Configuration of the Slack notification.                                                                                                                                   |
| `recreateDeadline`     | string                                              | Deadline for the Pod to be recreated. Default value is `24h`. This value should be parseable with `time.ParseDuration`.                                                    |
| `template`             | [RunnerPodTemplateSpec](#RunnerPodTemplateSpec)     | Pod manifest Template.                                                                                                                                                     |

**NOTE**: `maxRunnerPods` is equal-to or greater than `replicas`.

## SlackNotificationConfig

| Field                   | Type   | Description                                                                                                                                                                    |
| ----------------------- | ------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| `enable`                | string | Flag to toggle Slack notifications sends or not.                                                                                                                               |
| `channel`               | string | Slack channel which the job results are reported. If this field is omitted, the default channel specified in the `--channel`(`-c`) option of slack-agent command will be used. |
| `extendDuration`        | string | Extension time. If this field is omitted, users cannot extend the runner pods.                                                                                                 |
| `slackAgentServiceName` | string | Service name of Slack agent. If this field is omitted, the default name (`slack-agent.meows.svc`) will be used.                                                                |

## RunnerPodTemplateSpec

| Field                          | Type                                | Description                                                                                                        |
| ------------------------------ | ----------------------------------- | ------------------------------------------------------------------------------------------------------------------ |
| `image`                        | string                              | Docker image name for the runner container.                                                                        |
| `imagePullPolicy`              | string                              | Image pull policy for the runner container.                                                                        |
| `imagePullSecrets`             | \[\][corev1.LocalObjectReference][] | List of secret names in the same namespace to use for pulling any of the images.                                   |
| `securityContext`              | [corev1.SecurityContext][]          | Security options for the runner container.                                                                         |
| `env`                          | \[\][corev1.EnvVar][]               | List of environment variables to set in the runner container.                                                      |
| `resources`                    | [corev1.ResourceRequirements][]     | Compute Resources required by the runner container.                                                                |
| `workVolume`                   | [corev1.VolumeSource][]             | The volume source for the working directory.                                                                       |
| `volumeMounts`                 | \[\][corev1.VolumeMount][]          | Pod volumes to mount into the runner container's filesystem.                                                       |
| `volumes`                      | \[\][corev1.Volume][]               | List of volumes that can be mounted by containers belonging to the pod.                                            |
| `ServiceAccountName`           | string                              | Name of the service account that the Pod use. (default value is "default")                                         |
| `automountServiceAccountToken` | *bool                               | AutomountServiceAccountToken indicates whether a service account token should be automatically mounted to the pod. |

## RunnerPoolStatus

| Field   | Type    | Description                 |
| ------- | ------- | --------------------------- |
| `bound` | boolean | Deployment is bound or not. |

[ObjectMeta]: https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.20/#objectmeta-v1-meta
[corev1.LocalObjectReference]: https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.20/#localobjectreference-v1-core
[corev1.SecurityContext]: https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.20/#securitycontext-v1-core
[corev1.EnvVar]: https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.20/#envvar-v1-core
[corev1.ResourceRequirements]: https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.20/#resourcerequirements-v1-core
[corev1.VolumeSource]: https://pkg.go.dev/k8s.io/api/core/v1#VolumeSource
[corev1.VolumeMount]: https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.20/#volumemount-v1-core
[corev1.Volume]: https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.20/#volume-v1-core
