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

package backupschedule

import (
	"context"
	"fmt"

	"k8s.io/client-go/util/retry"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	backupoperatoriov1 "backup-operator.io/api/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// UpdateScheduleStatus updates counters and active runs in schedule.status
func UpdateScheduleStatus(ctx context.Context, c client.Client, schedule *backupoperatoriov1.BackupSchedule) (err error) {
	// Prepare and update the status...
	old := schedule.DeepCopy()
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		if err = c.Get(ctx, client.ObjectKeyFromObject(schedule), schedule); err != nil {
			return err
		}
		// Get all child runs...
		childRuns := &backupoperatoriov1.BackupRunList{}
		if err = c.List(ctx, childRuns, client.InNamespace(schedule.Namespace),
			client.MatchingFields{".metadata.controller": string(schedule.UID)}); err != nil {
			return err
		}
		// ...and counting them by state...
		inProgressRuns := []*backupoperatoriov1.BackupRun{}
		successfulRuns := []*backupoperatoriov1.BackupRun{}
		failedRuns := []*backupoperatoriov1.BackupRun{}
		var mostRecentSuccessful, mostResentFailed *metav1.Time
		// Also, we locate...
		for i, run := range childRuns.Items {
			// Get current run phase...
			var phase *backupoperatoriov1.BackupRunConditionType
			var notInitialized bool
			if notInitialized, _, phase = GetRunPhase(run); notInitialized {
				// Run is not processed by controller yet, skipping it
				continue
			} else if phase == nil {
				return fmt.Errorf("failed to get run phase: %s", run.Name)
			}
			createdAt := run.GetCreationTimestamp()
			switch *phase {
			// Add it to the respective list and update most recent var
			case backupoperatoriov1.BackupRunConditionTypeInProgress:
				inProgressRuns = append(inProgressRuns, &childRuns.Items[i])
			case backupoperatoriov1.BackupRunConditionTypeFailed:
				failedRuns = append(failedRuns, &childRuns.Items[i])
				if mostResentFailed == nil || mostResentFailed.Before(&createdAt) {
					mostResentFailed = ptr.To[metav1.Time](createdAt)
				}
			case backupoperatoriov1.BackupRunConditionTypeSuccessful:
				successfulRuns = append(successfulRuns, &childRuns.Items[i])
				if mostRecentSuccessful == nil || mostRecentSuccessful.Before(&createdAt) {
					mostRecentSuccessful = ptr.To[metav1.Time](createdAt)
				}
			}
		}
		// Set last successful time
		schedule.Status.LastSuccessfulTime = mostRecentSuccessful
		// Referencing all inProgress runs
		schedule.Status.Active = []corev1.ObjectReference{}
		for _, run := range inProgressRuns {
			schedule.Status.Active = append(schedule.Status.Active, corev1.ObjectReference{
				APIVersion: run.APIVersion,
				Kind:       run.Kind,
				Name:       run.Name,
				Namespace:  run.Namespace,
				UID:        run.UID,
			})
		}
		// ...and count runs after deletion cleaning
		schedule.Status.InProgress = ptr.To(uint16(len(inProgressRuns)))
		schedule.Status.Failed = ptr.To(uint16(len(failedRuns)))
		schedule.Status.Successful = ptr.To(uint16(len(successfulRuns)))
		schedule.Status.Total = ptr.To(*schedule.Status.InProgress + *schedule.Status.Failed + *schedule.Status.Successful)
		if old.Status.String() != schedule.Status.String() {
			// Make apply only in case of changes to the status
			return c.Status().Update(ctx, schedule)
		}
		return nil
	})
}
