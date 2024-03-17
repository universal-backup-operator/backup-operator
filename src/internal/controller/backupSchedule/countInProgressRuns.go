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

	"sigs.k8s.io/controller-runtime/pkg/client"

	backupoperatoriov1 "backup-operator.io/api/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Count all InProgress Runs
func CountInProgressRuns(ctx context.Context, c client.Client, schedule *backupoperatoriov1.BackupSchedule) (count uint, err error) {
	count = 0
	// Get all child runs...
	childRuns := &backupoperatoriov1.BackupRunList{}
	if err = c.List(ctx, childRuns, client.InNamespace(schedule.Namespace),
		client.MatchingFields{".metadata.controller": string(schedule.UID)}); err != nil {
		return
	}
	for _, run := range childRuns.Items {
		// ...find ones...
		for _, condition := range run.Status.Conditions {
			switch condition.Type {
			case string(backupoperatoriov1.BackupRunConditionTypeInProgress):
				// ...that are in progress...
				if condition.Status == metav1.ConditionTrue {
					// ...and count them
					count++
				}
			}
		}
	}
	return
}
