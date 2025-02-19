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

package backuprun

import (
	backupoperatoriov1 "backup-operator.io/api/v1"
	"backup-operator.io/internal/monitoring"
	"github.com/prometheus/client_golang/prometheus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// UpdateMetric updates metric value
func UpdateMetric(run *backupoperatoriov1.BackupRun) {
	ownerName := ""
	if owner := metav1.GetControllerOf(run); owner != nil {
		ownerName = owner.Name
	}
	DeleteMetric(run)
	monitoring.BackupOperatorRunStatus.WithLabelValues(
		run.Namespace, run.Name, *run.Status.State, ownerName, run.Spec.Storage.Name, run.Spec.Storage.Path,
	).Set(float64(run.CreationTimestamp.Time.Unix()))
	if run.Status.SizeInBytes != nil {
		monitoring.BackupOperatorRunBackupSizeBytes.WithLabelValues(
			run.Namespace, run.Name,
		).Set(float64(*run.Status.SizeInBytes))
	}
}

// DeleteMetric deletes metric from the operator
func DeleteMetric(run *backupoperatoriov1.BackupRun) {
	monitoring.BackupOperatorRunStatus.DeletePartialMatch(prometheus.Labels{
		"namespace": run.Namespace,
		"name":      run.Name,
	})
}
