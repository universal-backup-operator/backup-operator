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
	"sort"

	"sigs.k8s.io/controller-runtime/pkg/client"

	backupoperatoriov1 "backup-operator.io/api/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Function to deletes runs above the limit and return count of deleted runs
func DeleteRunsAboveTheLimit(ctx context.Context, c client.Client, schedule *backupoperatoriov1.BackupSchedule,
	ct backupoperatoriov1.BackupRunConditionType, limit uint16) (count uint16, err error) {
	count = 0
	// Get all child runs...
	childRuns := &backupoperatoriov1.BackupRunList{}
	if err = c.List(ctx, childRuns, client.InNamespace(schedule.Namespace),
		client.MatchingFields{".metadata.controller": string(schedule.UID)}); err != nil {
		return
	}
	// ...and filter only ones that match the condition
	var runs []backupoperatoriov1.BackupRun
	for _, run := range childRuns.Items {
		for _, cnd := range run.Status.Conditions {
			if _, keep := run.GetAnnotations()[backupoperatoriov1.AnnotationKeepBackupRun]; cnd.Type == string(ct) &&
				cnd.Status == metav1.ConditionTrue && !keep {
				runs = append(runs, run)
			}
		}
	}
	// Initial length
	l := len(runs)
	if l > 0 && uint16(l) >= limit {
		// Sort runs by time
		sort.Slice(runs, func(i, j int) bool {
			return (runs)[i].GetCreationTimestamp().UTC().Before((runs)[j].GetCreationTimestamp().UTC())
		})
		// Delete N oldest
		for i, run := range runs {
			if i >= l-int(limit) {
				break
			}
			if err = c.Delete(ctx, &run, client.PropagationPolicy(metav1.DeletePropagationBackground)); client.IgnoreNotFound(err) != nil {
				err = fmt.Errorf("unable to delete the run: %s", run.Name)
			}
			count++
		}
	}
	return
}
