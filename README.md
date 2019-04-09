# SqlDB Operator

**This is not an officially supported Google product**

SqlDB operator is a basic Kubernetes stateful operator that automates creation, backup, and restore workflows of SQL instances. This operator leverages [CustomResourceDefinitions](https://kubernetes.io/docs/tasks/access-kubernetes-api/custom-resources/custom-resource-definitions/) to represent each workflow declaratively:
1. `SqlDB`: for **instance creation** and **restore** workflows
2. `SqlBackup`: for **backup** workflow

## Recommended Versions
`kubernetes`: `1.11.0` and above  
`kubectl`: `1.11.0` and above

## Example Workflows: PostgreSQL

All example manifests are located under `config/samples`.

Before starting any of the workflows below, run following commands:
1. Run the operator:
`make run`  
_Note: You can also build an image of the operator and deploy it using a pod._

2. Run a NFS server (for storing backup file):
`kubectl apply -f config/samples/nfs.yaml`

3. Run an `alpine` pod and get a shell to its container:  
`kubectl run --generator=run-pod/v1 --image=alpine:latest -it alpine -- /bin/sh`

4. Install the container with `psql` terminal:  
`apk add --update postgresql-client`

To Detech from alpine-shell:
CTRL-P, CTRL-Q

To reattach to an existing alpine session:
kubectl attach -it alpine-shell

##### Instance Creation

1. Create a `SqlDB` resource named `db1` to bring up a PostgreSQL cluster:  
`kubectl apply -f config/samples/db1.yaml`

2. Wait for the instance to become ready:  
`kubectl get sqldb/db1`  
Verify `STATUS` field becomes `ServerReady` eventually.

3. Within the `alpine` container, connect to the PostgreSQL instance using `psql` terminal via Kubernetes Service named `sqldb-db1-svc` (username is `john` and password is `abc`):  
`psql -h sqldb-db1-svc -U john`

4. In the `psql` terminal, execute following commands:  
```
CREATE TABLE account(user_id serial PRIMARY KEY);
INSERT INTO account(user_id) VALUES (1234);
SELECT * FROM account;
```

5. Verify following output is shown:  
```
 user_id 
---------
    1234
(1 row)
```

##### Backup

1. Create a `SqlBackup` resource named `db1-backup`:  
`kubectl apply -f config/samples/db1-backup.yaml`

2. Wait for the backup to become ready:  
`kubectl get sqlbackup/db1-backup`  
Verify `STATUS` field becomes `BackupSucceeded` eventually.

##### Restore
1. Create another `SqlDB` resource named `db2` to bring up `another` PostgreSQL cluster:  
`kubectl apply -f config/samples/db2.yaml`  
_Note: `.spec.backupName` field has been specified to differentiate between instance creation and restore._

2. Wait for the restore to complete:  
`kubectl get sqldb/db2`  
Verify `STATUS` field becomes `ServerRestored` eventually.

3. Within the `alpine` container, connect to the instance using `psql` terminal via Kubernetes Service named `sqldb-db2-svc` (again, username is `john` and password is `abc`):  
`psql -h sqldb-db2-svc -U john`

4. In the `psql` terminal, execute following command:  
`SELECT * FROM account;`  
Verify that user ID `1234` is found.
