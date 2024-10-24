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

package controller

import (
	"context"
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/log"

	backupoperatoriov1 "backup-operator.io/api/v1"
	backupschedule "backup-operator.io/internal/controller/backupSchedule"
	"backup-operator.io/internal/controller/utils"
	batchv1 "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

// BackupScheduleReconciler reconciles a BackupSchedule object
type BackupScheduleReconciler struct {
	client.Client
	Config   *rest.Config
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

//+kubebuilder:rbac:groups=backup-operator.io,resources=backupschedules,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=backup-operator.io,resources=backupschedules/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=backup-operator.io,resources=backupschedules/finalizers,verbs=update
//+kubebuilder:rbac:groups=backup-operator.io,resources=backupschedules,verbs=get;list;watch
//+kubebuilder:rbac:groups=backup-operator.io,resources=backupruns,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=events,verbs=create;get;list;patch;update;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// (user): Modify the Reconcile function to compare the state specified by
// the BackupSchedule object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-sctime@v0.14.1/pkg/reconcile
func (r *BackupScheduleReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	return utils.ManageLifecycle(ctx, &utils.ManagedLifecycleReconcile{
		Client:   r.Client,
		Config:   r.Config,
		Scheme:   r.Scheme,
		Recorder: r.Recorder,
		Request:  req,
		Object:   &backupoperatoriov1.BackupSchedule{},
	}, &backupScheduleLifecycle{})
}

// Implements ManagedLifecycleObject interface
type backupScheduleLifecycle struct{}

// ┌─┐┌─┐┌┐┐┐─┐┌┐┐┬─┐┬ ┐┌┐┐┌─┐┬─┐
// │  │ ││││└─┐ │ │┬┘│ │ │ │ ││┬┘
// └─┘┘─┘┘└┘──┘ ┘ ┘└┘┘─┘ ┘ ┘─┘┘└┘

func (b *backupScheduleLifecycle) Constructor(ctx context.Context, r *utils.ManagedLifecycleReconcile) (result ctrl.Result, err error) {
	return
}

// ┬─┐┬─┐┐─┐┌┐┐┬─┐┬ ┐┌─┐┌┐┐┌─┐┬─┐
// │ │├─ └─┐ │ │┬┘│ ││   │ │ ││┬┘
// ┘─┘┴─┘──┘ ┘ ┘└┘┘─┘└─┘ ┘ ┘─┘┘└┘

func (b *backupScheduleLifecycle) Destructor(ctx context.Context, r *utils.ManagedLifecycleReconcile) (result ctrl.Result, err error) {
	schedule := r.Object.(*backupoperatoriov1.BackupSchedule)
	log := log.FromContext(ctx)
	if err = r.Client.Get(ctx, client.ObjectKeyFromObject(schedule), schedule); err != nil {
		utils.Log(r, log, err, schedule, "FailedGet", "could not get the schedule")
		return
	}
	backupschedule.DeleteMetric(schedule)
	return
}

// ┬─┐┬─┐┌─┐┌─┐┬─┐┐─┐┐─┐┌─┐┬─┐
// │─┘│┬┘│ ││  ├─ └─┐└─┐│ ││┬┘
// ┘  ┘└┘┘─┘└─┘┴─┘──┘──┘┘─┘┘└┘

func (b *backupScheduleLifecycle) Processor(ctx context.Context, r *utils.ManagedLifecycleReconcile) (result ctrl.Result, err error) {
	schedule := r.Object.(*backupoperatoriov1.BackupSchedule)
	log := log.FromContext(ctx)
	if err = r.Client.Get(ctx, client.ObjectKeyFromObject(schedule), schedule); err != nil {
		utils.Log(r, log, err, schedule, "FailedGet", "could not get the schedule")
		return
	}
	// Update the status
	if err = backupschedule.UpdateScheduleStatus(ctx, r.Client, schedule); err != nil {
		utils.Log(r, log, err, schedule, "FailedUpdateStatus", "failed to update the status")
		return
	}
	// Update metrics
	backupschedule.UpdateMetric(schedule)
	// Check if we suspended
	if backupschedule.CheckScheduleSuspended(schedule) {
		utils.Log(r, log, err, schedule, "Skipping", "schedule is suspended")
		return result, nil
	}
	// Delete old runs if limit is set...
	if schedule.Spec.FailedRunsHistoryLimit != nil {
		var count uint16
		if count, err = backupschedule.DeleteRunsAboveTheLimit(ctx, r.Client, schedule,
			backupoperatoriov1.BackupRunConditionTypeFailed, *schedule.Spec.FailedRunsHistoryLimit); err != nil {
			return
		}
		if count > 0 {
			utils.Log(r, log, err, schedule, "DeletedFailedRuns", fmt.Sprintf("deleted %d failed runs", count))
		}
	}
	if schedule.Spec.SuccessfulRunsHistoryLimit != nil {
		var count uint16
		if count, err = backupschedule.DeleteRunsAboveTheLimit(ctx, r.Client, schedule,
			backupoperatoriov1.BackupRunConditionTypeSuccessful, *schedule.Spec.SuccessfulRunsHistoryLimit); err != nil {
			return
		}
		if count > 0 {
			utils.Log(r, log, err, schedule, "DeletedSuccessfulRuns", fmt.Sprintf("deleted %d successful runs", count))
		}
	}
	// Figure out the next times that we need to create
	// runs at (or anything we missed).
	var missedRun, nextRun time.Time
	if missedRun, nextRun, err = backupschedule.ParseScheduleCron(schedule, time.Now()); err != nil {
		utils.Log(r, log, err, schedule, "FailedScheduleCalc", "unable to figure out the next scheduled run time")
		// We don't really care about requeuing until we get an update that
		// Fixes the schedule, so don't return an error
		return result, nil
	}
	// Remove trigger annotation if present
	if _, ok := schedule.Annotations[backupoperatoriov1.AnnotationTriggerSchedule]; ok {
		if err = utils.SetAnnotations(ctx, r.Client, schedule, func() (a map[string]string) {
			a = make(map[string]string)
			for k, v := range schedule.GetAnnotations() {
				if k != backupoperatoriov1.AnnotationTriggerSchedule {
					a[k] = v
				}
			}
			return
		}()); err != nil {
			return
		}
	}
	// Set requeue to the next run
	result.RequeueAfter = time.Until(nextRun)
	log = log.WithValues("now", time.Now(), "nextRun", nextRun)
	// Current run must appear as missed run, because we are reconciled instantly after the schedule
	// If missed run is empty, then we are launched for the first time or something bad has happened
	if missedRun.IsZero() {
		return result, nil
	} else if schedule.Status.LastScheduleTime != nil {
		log = log.WithValues("missedRun", missedRun, "lastScheduleTime", schedule.Status.LastScheduleTime.String())
		// if time.Since(schedule.Status.LastScheduleTime.Time) < time.Minute {
		// 	// If we have scheduled less than a minute ago - exit without an error
		// 	return result, nil
		// }
	}
	// Make sure we're not too late to start the run
	tooLate := false
	if schedule.Spec.StartingDeadlineSeconds != nil {
		tooLate = missedRun.
			Add(time.Duration(*schedule.Spec.StartingDeadlineSeconds) * time.Second).
			Before(time.Now())
	}
	if tooLate {
		utils.Log(r, log, err, schedule, "MissedTime",
			"missed the starting deadline for the last run, waiting until the next scheduled time",
		)
		return result, nil
	}
	// Figure out how to run -- concurrency policy might forbid us from running multiple backups...
	switch *schedule.Spec.ConcurrencyPolicy {
	case batchv1.ForbidConcurrent:
		var count uint
		if count, err = backupschedule.CountInProgressRuns(ctx, r.Client, schedule); err != nil {
			return
		}
		if count > 0 {
			utils.Log(r, log, err, schedule, "NoConcurrency",
				fmt.Sprintf("concurrency policy blocks concurrent runs, skipping because there are %d in progress runs", count),
			)
			return
		}
	case batchv1.ReplaceConcurrent:
		utils.Log(r, log, err, schedule, "DeletingOldInProgressRuns",
			"deleting in progress runs according to .spec.concurrencyPolicy",
		)
		if err = backupschedule.DeleteInProgressAndStaleRuns(ctx, r.Client, schedule); err != nil {
			return
		}
	}
	// Create new run
	runName := fmt.Sprintf("%s-%d", schedule.Name, missedRun.Unix())
	utils.Log(r, log, err, schedule, "CreatingRun", fmt.Sprintf("creating BackupRun %s", runName))
	if err = backupschedule.CreateRunFromSchedule(ctx, r.Client, r.Scheme, schedule, runName); err != nil {
		// No need to fail, we will be rescheduled anyway
		utils.Log(r, log, err, schedule, "FailedCreateRun", "failed to create the BackupRun")
	}
	// Update the status
	if err = backupschedule.UpdateScheduleStatus(ctx, r.Client, schedule); err != nil {
		return
	}
	// Update metrics
	backupschedule.UpdateMetric(schedule)
	return
}

var backupScheduleIndexers = map[string]client.IndexerFunc{
	".metadata.controller": func(o client.Object) []string {
		owner := metav1.GetControllerOf(o)
		if owner == nil {
			return nil
		}
		return []string{string(owner.UID)}
	},
	".spec.template.spec.storage.name": func(o client.Object) []string {
		schedule := o.(*backupoperatoriov1.BackupSchedule)
		return []string{string(schedule.Spec.Template.Spec.Storage.Name)}
	},
}

// SetupWithManager sets up the controller with the Manager.
func (r *BackupScheduleReconciler) SetupWithManager(mgr ctrl.Manager) error {
	for path, function := range backupScheduleIndexers {
		if err := mgr.GetFieldIndexer().IndexField(context.Background(),
			&backupoperatoriov1.BackupSchedule{}, path, function); err != nil {
			return err
		}
	}
	return ctrl.NewControllerManagedBy(mgr).
		For(&backupoperatoriov1.BackupSchedule{}).
		Owns(&backupoperatoriov1.BackupRun{}).
		WithOptions(controller.Options{MaxConcurrentReconciles: 10}).
		Complete(r)
}
