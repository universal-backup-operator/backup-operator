MySQL
===

Schedule below will do the following.

1. Backup MySQL and compress it with GZIP
2. Drop database before the restoration
3. Restore the database
4. Print info and warnings to Pod main log streams, so you can debug it if you want
5. Will keep last 7 backups
6. Limit backup and restoration to 5 minutes

```yaml
apiVersion: backup-operator.io/v1
kind: BackupSchedule
metadata:
  name: backup-mysql
spec: # (1)
  schedule: "0 0 * * *"
  concurrencyPolicy: Replace
  successfulRunsHistoryLimit: 7
  failedRunsHistoryLimit: 1
  template:
    spec:
      retainPolicy: Retain # (2)
      storage:
        name: minio
        path: "/${CLUSTER}/mysql-backend-{{ now | date \"20060102-150405\" }}.sql.gz" # (3)
      compression:
        algorithm: gzip
        level: 5
      backup:
        deadlineSeconds: 300
        container: mysql
        command: ["/bin/bash", "-o", "pipefail", "-ec"]
        args: # (4)
        - |-
          backup() {
            mysqldump \
              --host "${HOSTNAME}" \
              --port "${PORT}" \
              --user "${USERNAME}" \
              --password="${PASSWORD}" \
              --no-create-db \
              --all-tablespaces \
              --add-drop-database \
              --quote-names \
              --routines \
              --triggers \
              --events \
              --set-gtid-purged=OFF \
              --max_allowed_packet=512M \
              "${DATABASE}"
          }
          main() {
            echo "info: starting" 1>/proc/1/fd/1
            backup
            echo "info: finished" 1>/proc/1/fd/1
            sleep 5
          }
          main 2>/proc/1/fd/2
      restore:
        deadlineSeconds: 300
        container: mysql
        command: ["/bin/bash", "-o", "pipefail", "-ec"]
        args: # (5)
        - |-
          drop() {
            echo "
            DROP DATABASE IF EXISTS ${DATABASE};
            CREATE DATABASE ${DATABASE};
            " | mysql \
            --host "${HOSTNAME}" --port "${PORT}" \
            --user "${USERNAME}" --password="${PASSWORD}" \
            --no-auto-rehash
            echo "info: drop is completed"
          }
          restore() {
            mysql \
              --host "${HOSTNAME}" --port "${PORT}" \
              --user "${USERNAME}" --password="${PASSWORD}" \
              --no-auto-rehash "${DATABASE}"
            echo "info: restore is completed"
          }
          main() {
            echo "info: starting"
            drop
            restore
            echo "info: finished"
          }
          main 1>/proc/1/fd/1 2>/proc/1/fd/2 || (cat >dev/null; exit 1)
      template:
        spec:
          restartPolicy: Never
          containers:
          - name: mysql
            image: docker.io/mysql:8.4.0
            imagePullPolicy: IfNotPresent
            command: ["sleep", "-c"] # (6)
            args: ["while sleep 1; do true; done"]
            securityContext:
              capabilities:
                drop: [ALL]
              privileged: false
              allowPrivilegeEscalation: false
              runAsNonRoot: true
              runAsUser: 1000
              seccompProfile:
                type: RuntimeDefault
            resources:
              limits:
                memory: 2Gi
              requests:
                cpu: 100m
                memory: 200Mi
            env:
            - name: HOSTNAME
              value: mysql.default.svc.cluster.local
            - name: PORT
              value: "3306"
            - name: DATABASE
              value: backend
            - name: USERNAME
              value: backend
            - name: PASSWORD
              valueFrom:
                secretKeyRef:
                  key: mysql-password
                  name: mysql-backend-passwords # (7)
```

1. Same fields as for a vanilla CronJob
2. Impact on retain or removal of backup from storage on BackupRun deletion
3. Templated with [Go Template](https://pkg.go.dev/text/template) and [Sprig](https://masterminds.github.io/sprig/) functions
4. While stdout of the command is considered as a backup data, we use `/proc/1/fd/*` to redirect some info and errors to container main logs streams so you can debug it
5. There is a `cat >/dev/null` at the end of main function call. It is required in case if restore command fails and noone consumes STDIN. Kubernetes fails to finish exec command and we have to trash that data somehow.
6. We do not use a simple `command: [sleep, 1d]`, because killing the Pod after backup is done is faster, you have a loop with small sleeps
7. Do not forget to create the respective secret with MySQL credentials
