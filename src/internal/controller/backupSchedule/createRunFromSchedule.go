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

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/util/retry"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	backupoperatoriov1 "backup-operator.io/api/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

func CreateRunFromSchedule(ctx context.Context, c client.Client, s *runtime.Scheme, schedule *backupoperatoriov1.BackupSchedule, runName string) (err error) {
	// Create run...
	run := &backupoperatoriov1.BackupRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:      runName,
			Namespace: schedule.Namespace,
		},
		Spec: *schedule.Spec.Template.Spec.DeepCopy(),
	}
	// ...set labels and annotations it any...
	if schedule.Spec.Template.Metadata != nil {
		run.SetLabels(schedule.Spec.Template.Metadata.Labels)
		run.SetAnnotations(schedule.Spec.Template.Metadata.Annotations)
	}
	// ...do not forget to set ownership...
	if err = ctrl.SetControllerReference(schedule, run, s); err != nil {
		return
	}
	// ...and create the run
	err = c.Create(ctx, run)
	// Update last scheduled time
	err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
		if err = c.Get(ctx, client.ObjectKeyFromObject(schedule), schedule); err != nil {
			return err
		}
		schedule.Status.LastScheduleTime = ptr.To(run.CreationTimestamp)
		return c.Status().Update(ctx, schedule)
	})
	return
}
