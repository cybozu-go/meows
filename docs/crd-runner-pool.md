RunnerPool
----------

`RunnerPool` is a custom resource definition (CRD) that represents a pool of
GitHub Actions self-hosted runners.

| Field        | Type                                  | Description                                        |
|--------------|---------------------------------------|----------------------------------------------------|
| `apiVersion` | string                                | APIVersion.                                        |
| `kind`       | string                                | Kind.                                              |
| `metadata`   | [ObjectMeta][]                        | Metadata.                                          |
| `spec`       | [RunnerPoolSpec](#RunnerPoolSpec)     | Specification of desired behavior of `RunnerPool`. |
| `status`     | [RunnerPoolStatus](#RunnerPoolStatus) | Most recently observed status of `RunnerPool`.     | ]

RunnerPoolSpec
--------------

| Field                   | Type                   | Description                                              |
|-------------------------|------------------------|----------------------------------------------------------|
| `repositoryName`        | string                 | Repository Name to register Pods as self-hosted runners. |
| `slackAgentServiceName` | string                 | Service name of Slack agent.                             |
| `replicas`              | int32                  | Number of desired Pods.                                  |
| `selector`              | [LabelSelector][]      | Label selector for pods.                                 |
| `template`              | [PodTemplateSpec][]    | Pod manifest Template.                                   |
| `strategy`              | [DeploymentStrategy][] | Strategy to replace existing Pods with new ones.         |

RunnerPoolStatus
----------------

| Field   | Type    | Description                 |
|---------|---------|-----------------------------|
| `bound` | boolean | Deployment is bound or not. |

[ObjectMeta]: https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.20/#objectmeta-v1-meta
[PodTemplateSpec]: https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.20/#podtemplatespec-v1-core
[LabelSelector]: https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.20/#labelselector-v1-meta
[DeploymentStrategy]: https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.20/#deploymentstrategy-v1-apps
