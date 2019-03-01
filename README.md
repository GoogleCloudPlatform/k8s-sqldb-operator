# SqlDB Operator

SqlDB operator is a basic Kubernetes stateful operator that automates creation, backup, and restore workflows of SQL instances. This operator leverages [CustomResourceDefinitions](https://kubernetes.io/docs/tasks/access-kubernetes-api/custom-resources/custom-resource-definitions/) to represent each workflow declaratively:
1. `SqlDB`: for creation and restore workflows
2. `SqlBackup`: for backup workflow