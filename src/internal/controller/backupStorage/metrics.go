/*
Copyright 2023.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package backupstorage

import (
	backupoperatoriov1 "backup-operator.io/api/v1"
	"backup-operator.io/internal/monitoring"
	"github.com/prometheus/client_golang/prometheus"
)

func UpdateMetric(storage *backupoperatoriov1.BackupStorage) {
	DeleteMetric(storage)
	monitoring.BackupOperatorStorageStatus.WithLabelValues(
		storage.Name, string(storage.Spec.Type),
	).SetToCurrentTime()
}

func DeleteMetric(storage *backupoperatoriov1.BackupStorage) {
	monitoring.BackupOperatorStorageStatus.DeletePartialMatch(prometheus.Labels{
		"name": storage.Name,
	})
}
