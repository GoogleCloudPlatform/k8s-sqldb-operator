# SqlDB Operator

**This is not an officially supported Google product.**

SqlDB operator is a basic Kubernetes stateful operator that automates creation, backup, and restore workflows of SQL instances. This operator leverages [CustomResourceDefinitions](https://kubernetes.io/docs/tasks/access-kubernetes-api/custom-resources/custom-resource-definitions/) to represent each workflow declaratively:
1. `SqlDB`: for **server deployment** and **restore** workflows
2. `SqlBackup`: for **backup** workflow

## Example Workflows: PostgreSQL

##### Server Deployment

1. `kubectl apply -f config/samples/operator_v1alpha1_sqldb.yaml` (don't specify `.spec.backupName` field)
2. `kubectl describe sqldb/sqldb-sample` (wait until `.status.phase` field becomes `ServerReady`)
3. `kubectl exec -it sqldb-sample-statefulset-0 /bin/bash`

After executing into pod `sqldb-sample-statefulset-0` (name is fixed), run following `psql` commands:
```
psql -c 'create role root with superuser;' -U postgres
psql -c 'alter user postgres login;' -U postgres
psql -c 'CREATE TABLE account(user_id serial PRIMARY KEY);' -U postgres
psql -c 'INSERT INTO account(user_id) VALUES (1234)' -U postgres
psql -c 'SELECT * FROM account;' -U postgres
```

You should see following output:
```
 user_id 
---------
    1234
(1 row)
```

##### Backup

1. `kubectl apply -f config/samples/operator_v1alpha1_sqlbackup.yaml`
2. `kubectl describe sqlbackup/sqlbackup-sample` (wait until `.status.phase` field becomes `BackupSucceeded`)

##### Restore
1. `kubectl delete -f config/samples/operator_v1alpha1_sqldb.yaml` (tear down existing PostgreSQL servers)
2. `kubectl apply -f config/samples/operator_v1alpha1_sqldb.yaml` (specify `.spec.backupName` field to "sqlbackup-sample" to trigger restore)
3. `kubectl describe sqldb/sqldb-sample` (wait until `.status.phase` field becomes `ServerRestored`)
4. `kubectl exec -it sqldb-sample-statefulset-0 /bin/bash`
5. `psql -c 'SELECT * FROM account;' -U postgres`

Verify that the `1234` user ID data is restored.