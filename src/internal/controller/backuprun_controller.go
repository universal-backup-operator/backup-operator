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
	"errors"
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/retry"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/log"

	backupoperatoriov1 "backup-operator.io/api/v1"
	backuprun "backup-operator.io/internal/controller/backupRun"
	backupstorage "backup-operator.io/internal/controller/backupStorage"
	"backup-operator.io/internal/controller/utils"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

// BackupRunReconciler reconciles a BackupRun object
type BackupRunReconciler struct {
	client.Client
	Config   *rest.Config
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

//+kubebuilder:rbac:groups=backup-operator.io,resources=backupruns,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=backup-operator.io,resources=backupruns/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=backup-operator.io,resources=backupruns/finalizers,verbs=update
//+kubebuilder:rbac:groups=backup-operator.io,resources=backupschedules,verbs=get
//+kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=pods/exec,verbs=create
//+kubebuilder:rbac:groups="",resources=events,verbs=create;get;list;patch;update;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// (user): Modify the Reconcile function to compare the state specified by
// the BackupRun object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.14.1/pkg/reconcile
func (r *BackupRunReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	return utils.ManageLifecycle(ctx, &utils.ManagedLifecycleReconcile{
		Client:   r.Client,
		Config:   r.Config,
		Scheme:   r.Scheme,
		Recorder: r.Recorder,
		Request:  req,
		Object:   &backupoperatoriov1.BackupRun{},
	}, &backupRunLifecycle{})
}

// Implements ManagedLifecycleObject interface
type backupRunLifecycle struct{}

// ┌─┐┌─┐┌┐┐┐─┐┌┐┐┬─┐┬ ┐┌┐┐┌─┐┬─┐
// │  │ ││││└─┐ │ │┬┘│ │ │ │ ││┬┘
// └─┘┘─┘┘└┘──┘ ┘ ┘└┘┘─┘ ┘ ┘─┘┘└┘

func (b *backupRunLifecycle) Constructor(ctx context.Context, r *utils.ManagedLifecycleReconcile) (result ctrl.Result, err error) {
	run := r.Object.(*backupoperatoriov1.BackupRun)
	log := log.FromContext(ctx)
	// Setting up different flag conditions
	if err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
		if err = r.Client.Get(ctx, client.ObjectKeyFromObject(run), run); err != nil {
			utils.Log(r, log, err, run, "FailedGet", "could not get the run")
			return err
		}
		// Analyse the run
		state := backuprun.AnalyzeRunConditions(run)
		// Check encryption
		if state.Encrypted {
			r.Recorder.Eventf(run, corev1.EventTypeNormal, "Encryption",
				fmt.Sprintf("recipients count %d", len(run.Spec.Encryption.Recipients)),
			)
		}
		// Check compression
		if state.Compressed {
			r.Recorder.Eventf(run, corev1.EventTypeNormal, "Compression",
				fmt.Sprintf("%s with level %d", run.Spec.Compression.Algorithm, run.Spec.Compression.Level),
			)
		}
		// Create respective conditions
		run.Status.Conditions = *utils.AddConditions(run.Status.Conditions,
			metav1.Condition{
				Type:               backupoperatoriov1.ConditionTypeReady,
				Status:             utils.ToConditionStatus(&state.Ready),
				Reason:             utils.EventReasonInitializing,
				Message:            utils.EventReasonInitializing,
				LastTransitionTime: metav1.Now(),
				ObservedGeneration: run.Generation,
			},
			metav1.Condition{
				Type:               string(backupoperatoriov1.BackupRunConditionTypeNeverRun),
				Status:             utils.ToConditionStatus(&state.NeverRun),
				Reason:             utils.EventReasonInitializing,
				Message:            utils.EventReasonInitializing,
				LastTransitionTime: metav1.Now(),
				ObservedGeneration: run.Generation,
			},
			metav1.Condition{
				Type:               string(backupoperatoriov1.BackupRunConditionTypeInProgress),
				Status:             utils.ToConditionStatus(&state.InProgress),
				Reason:             utils.EventReasonInitializing,
				Message:            utils.EventReasonInitializing,
				LastTransitionTime: metav1.Now(),
				ObservedGeneration: run.Generation,
			},
			metav1.Condition{
				Type:               string(backupoperatoriov1.BackupRunConditionTypeFailed),
				Status:             utils.ToConditionStatus(&state.Failed),
				Reason:             utils.EventReasonInitializing,
				Message:            utils.EventReasonInitializing,
				LastTransitionTime: metav1.Now(),
				ObservedGeneration: run.Generation,
			},
			metav1.Condition{
				Type:               string(backupoperatoriov1.BackupRunConditionTypeSuccessful),
				Status:             utils.ToConditionStatus(&state.Successful),
				Reason:             utils.EventReasonInitializing,
				Message:            utils.EventReasonInitializing,
				LastTransitionTime: metav1.Now(),
				ObservedGeneration: run.Generation,
			},
			metav1.Condition{
				Type:               string(backupoperatoriov1.BackupRunConditionTypeRestorable),
				Status:             utils.ToConditionStatus(&state.Restorable),
				Reason:             utils.EventReasonInitializing,
				Message:            utils.EventReasonInitializing,
				LastTransitionTime: metav1.Now(),
				ObservedGeneration: run.Generation,
			},
			metav1.Condition{
				Type:               string(backupoperatoriov1.BackupRunConditionTypeEncrypted),
				Status:             utils.ToConditionStatus(&state.Encrypted),
				Reason:             utils.EventReasonInitializing,
				Message:            utils.EventReasonInitializing,
				LastTransitionTime: metav1.Now(),
				ObservedGeneration: run.Generation,
			},
			metav1.Condition{
				Type:               string(backupoperatoriov1.BackupRunConditionTypeCompressed),
				Status:             utils.ToConditionStatus(&state.Compressed),
				Reason:             utils.EventReasonInitializing,
				Message:            utils.EventReasonInitializing,
				LastTransitionTime: metav1.Now(),
				ObservedGeneration: run.Generation,
			},
		)
		// Update status subresource
		return r.Client.Status().Update(ctx, run)
	}); err != nil {
		return
	}
	// Finish initialization
	utils.Log(r, log, err, run, "Reconciled", "initialized the object after operator (re)start")
	return
}

// ┬─┐┬─┐┐─┐┌┐┐┬─┐┬ ┐┌─┐┌┐┐┌─┐┬─┐
// │ │├─ └─┐ │ │┬┘│ ││   │ │ ││┬┘
// ┘─┘┴─┘──┘ ┘ ┘└┘┘─┘└─┘ ┘ ┘─┘┘└┘

func (b *backupRunLifecycle) Destructor(ctx context.Context, r *utils.ManagedLifecycleReconcile) (result ctrl.Result, err error) {
	run := r.Object.(*backupoperatoriov1.BackupRun)
	var ok bool
	log := log.FromContext(ctx)
	if err = r.Client.Get(ctx, client.ObjectKeyFromObject(run), run); err != nil {
		utils.Log(r, log, err, run, "FailedGet", "could not get the run")
		return
	}
	utils.Log(r, log, err, run, "StartingDeletion", "starting deletion of the run")
	// Deleting metric
	backuprun.DeleteMetric(run)
	// Analyse the run
	state := backuprun.AnalyzeRunConditions(run)
	// Delete backup from storage...
	if *run.Spec.RetainPolicy == backupoperatoriov1.BackupRetainDelete && state.Completed {
		// ...if retain is set to Delete
		var storage backupstorage.BackupStorageProvider
		if storage, ok = backupstorage.GetBackupStorageProvider(run.Spec.Storage.Name); !ok {
			utils.Log(r, log, errors.New("FailedFindStorage"), run, "FailedFindStorage",
				fmt.Sprintf("no storage provider with name %s found, ignore if operator has restarted recently", run.Spec.Storage.Name),
			)
			result.RequeueAfter = time.Second * 5
			return
		}
		log = log.WithValues("storageName", run.Spec.Storage.Name, "backupPath", run.Spec.Storage.Path)
		utils.Log(r, log, err, run, "DeletingFromStorage", fmt.Sprintf("deleting backup at %s", run.Spec.Storage.Path))
		if err = storage.Delete(ctx, run.Spec.Storage.Path); err != nil {
			utils.Log(r, log, err, run, "FailedDeletion", "failed to delete the backup from storage")
			result.RequeueAfter = time.Minute
			return
		}
	}
	return
}

// ┬─┐┬─┐┌─┐┌─┐┬─┐┐─┐┐─┐┌─┐┬─┐
// │─┘│┬┘│ ││  ├─ └─┐└─┐│ ││┬┘
// ┘  ┘└┘┘─┘└─┘┴─┘──┘──┘┘─┘┘└┘

func (b *backupRunLifecycle) Processor(ctx context.Context, r *utils.ManagedLifecycleReconcile) (result ctrl.Result, err error) {
	run := r.Object.(*backupoperatoriov1.BackupRun)
	var ok bool
	log := log.FromContext(ctx)
	if err = r.Client.Get(ctx, client.ObjectKeyFromObject(run), run); err != nil {
		utils.Log(r, log, err, run, "FailedGet", "could not get the run")
		return
	}
	// Analyze run conditions
	state := backuprun.AnalyzeRunConditions(run)
	// Check interruption
	if state.Interrupted {
		utils.Log(r, log, errors.New("InterruptedRun"), run, "InterruptedRun", "run has been interrupted by some reason")
		backuprun.ChangeRunState(ctx, r.Client, run, backupoperatoriov1.BackupRunConditionTypeFailed, state)
		return
	}
	// Exit if we do not have to run
	if !state.HaveToBackup && !state.HaveToRestore {
		// Update metrics
		backuprun.UpdateMetric(run)
		return
	}
	// Trying to get backup storage provider...
	var storage backupstorage.BackupStorageProvider
	if storage, ok = backupstorage.GetBackupStorageProvider(run.Spec.Storage.Name); !ok {
		// ...if fail - reschedule
		utils.Log(r, log, errors.New("FailedFindStorage"), run, "FailedFindStorage",
			fmt.Sprintf("no storage provider with name %s found, ignore if operator has restarted recently", run.Spec.Storage.Name),
		)
		backuprun.ChangeRunState(ctx, r.Client, run, backupoperatoriov1.BackupRunConditionTypeFailed, state)
		result.RequeueAfter = time.Second * 20
		return
	}
	// Set InProgress to true
	utils.Log(r, log, err, run, "InProgress", "run is in progress")
	if err = backuprun.ChangeRunState(ctx, r.Client, run, backupoperatoriov1.BackupRunConditionTypeInProgress, state); err != nil {
		utils.Log(r, log, err, run, "FailedChangeState", "failed to change the state")
		return
	}
	// Create Pod
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      run.Name,
			Namespace: run.Namespace,
		},
		Spec: *run.Spec.Template.Spec.DeepCopy(),
	}
	log = log.WithValues("pod", pod.Name)
	utils.Log(r, log, err, run, "CreatingPod", fmt.Sprintf("creating pod %s", pod.Name))
	if err = backuprun.CreatePodFromRun(ctx, r.Client, r.Scheme, r.Config, run, pod); err != nil {
		utils.Log(r, log, err, run, "FailedCreatePod", "failed to create the pod")
		// Fail the run
		backuprun.ChangeRunState(ctx, r.Client, run, backupoperatoriov1.BackupRunConditionTypeFailed, state)
		return
	}
	// Schedule pod deletion at the end of function
	defer func() {
		utils.Log(r, log, err, run, "DeletingPod", fmt.Sprintf("deleting pod %s", pod.Name))
		if err = r.Client.Delete(ctx, pod, &client.DeleteOptions{
			PropagationPolicy: ptr.To(metav1.DeletePropagationForeground),
		}); err != nil {
			utils.Log(r, log, err, run, "FailedDeletePod", "failed to delete the pod")
			return
		}
	}()
	// Backup or Restore
	switch {
	case state.HaveToBackup:
		utils.Log(r, log, err, run, "MakingBackup", "creating a new backup")
		if err = backuprun.Backup(ctx, r.Client, r.Scheme, r.Config, run, pod, storage); err != nil {
			utils.Log(r, log, err, run, "FailedBackup", "failed to make a backup")
			backuprun.ChangeRunState(ctx, r.Client, run, backupoperatoriov1.BackupRunConditionTypeFailed, state)
			return
		}
		if err = backuprun.SetBackupSizeInStatus(ctx, r.Client, run, storage); err != nil {
			utils.Log(r, log, err, run, "FailedSetBackupSize", "failed to set backup size in status")
			// No need to fail, that is not critical
		}
	case state.HaveToRestore:
		utils.Log(r, log, err, run, "RestoringBackup", "restoring a backup")
		// First clean restore annotation
		utils.SetAnnotations(ctx, r.Client, run, func() (a map[string]string) {
			a = make(map[string]string)
			for k, v := range run.GetAnnotations() {
				if k != backupoperatoriov1.AnnotationRestore && k != backupoperatoriov1.AnnotationRestoredAt {
					a[k] = v
				}
			}
			return
		}())
		// Start restoration
		if err = backuprun.Restore(ctx, r.Client, r.Scheme, r.Config, run, pod, storage); err != nil {
			backuprun.ChangeRunState(ctx, r.Client, run, backupoperatoriov1.BackupRunConditionTypeFailed, state)
			// Make failure restoration event
			utils.Log(r, log, err, run, "FailedRestore", "failed to restore a backup")
			return
		}
		// Make successful restoration event and update annotations
		utils.Log(r, log, err, run, "RestorationCompleted", "restoration has been completed successfully")
		utils.SetAnnotations(ctx, r.Client, run, func() (a map[string]string) {
			a = run.GetAnnotations()
			if a == nil {
				a = make(map[string]string)
			}
			a[backupoperatoriov1.AnnotationRestoredAt] = metav1.Now().String()
			return
		}())
	}
	// Finished successfully
	if err = backuprun.ChangeRunState(ctx, r.Client, run, backupoperatoriov1.BackupRunConditionTypeSuccessful, state); err != nil {
		return
	}
	return
}

var backupRunIndexers = map[string]client.IndexerFunc{
	".metadata.controller": func(o client.Object) []string {
		owner := metav1.GetControllerOf(o)
		if owner == nil {
			return nil
		}
		return []string{string(owner.UID)}
	},
	".spec.storage.name": func(o client.Object) []string {
		run := o.(*backupoperatoriov1.BackupRun)
		return []string{run.Spec.Storage.Name}
	},
}

// SetupWithManager sets up the controller with the Manager.
func (r *BackupRunReconciler) SetupWithManager(mgr ctrl.Manager) error {
	for path, function := range backupRunIndexers {
		if err := mgr.GetFieldIndexer().IndexField(context.Background(),
			&backupoperatoriov1.BackupRun{}, path, function); err != nil {
			return err
		}
	}
	return ctrl.NewControllerManagedBy(mgr).
		For(&backupoperatoriov1.BackupRun{}).
		Owns(&corev1.Pod{}).
		WithOptions(controller.Options{MaxConcurrentReconciles: 10}).
		Complete(r)
}
