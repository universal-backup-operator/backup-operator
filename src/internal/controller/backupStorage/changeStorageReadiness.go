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
	"context"

	"backup-operator.io/internal/controller/utils"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"

	backupoperatoriov1 "backup-operator.io/api/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Change run conditions and status according to the new state
func ChangeStorageReadiness(ctx context.Context, c client.Client,
	storage *backupoperatoriov1.BackupStorage, ready bool, reason string) (err error) {
	oldReady, err := utils.GetConditionByType(&storage.Status.Conditions, backupoperatoriov1.ConditionTypeReady)
	if err == nil && oldReady.Status != utils.ToConditionStatus(&ready) {
		return retry.RetryOnConflict(retry.DefaultRetry, func() error {
			if err = c.Get(ctx, client.ObjectKeyFromObject(storage), storage); err != nil {
				return err
			}
			if !ready {
				RemoveBackupStorageProvider(storage.Name)
			}
			storage.Status.Conditions = *utils.AddOrUpdateConditions(storage.Status.Conditions,
				metav1.Condition{
					Type:               backupoperatoriov1.ConditionTypeReady,
					Status:             utils.ToConditionStatus(&ready),
					Reason:             utils.EventReasonReconciled,
					Message:            reason,
					LastTransitionTime: metav1.Now(),
					ObservedGeneration: storage.Generation,
				},
			)
			return c.Status().Update(ctx, storage)
		})
	}
	return
}
