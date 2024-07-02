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
	if err = r.Client.Get(ctx, client.ObjectKeyFromObject(run), run); err != nil {
		log.V(1).Info("could not fetch an object")
		return
	}
	// Setting up different flag conditions
	if err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
		if err = r.Client.Get(ctx, client.ObjectKeyFromObject(run), run); err != nil {
			return err
		}
		// Analyse the run
		state := backuprun.AnalyzeRunConditions(run)
		// Condition status and message variables
		var restorableMsg, encryptedMsg, compressedMsg string
		// Check encryption
		if state.Encrypted {
			encryptedMsg = run.Spec.Encryption.Recipients[0]
		} else {
			encryptedMsg = "Not encrypted"
		}
		// Check compression
		if state.Compressed {
			compressedMsg = string(run.Spec.Compression.Algorithm)
		} else {
			compressedMsg = "Not compressed"
		}
		// Check restoration
		if state.Restorable {
			restorableMsg = "Restorable"
		} else if state.Encrypted && run.Spec.Restore != nil {
			restorableMsg = "No decryption key provided"
		}
		// Create respective conditions
		run.Status.Conditions = *utils.AddConditions(run.Status.Conditions,
			metav1.Condition{
				Type:               string(backupoperatoriov1.BackupRunConditionTypeNeverRun),
				Status:             utils.ToConditionStatus(&state.NeverRun),
				Reason:             utils.EventReasonCreated,
				Message:            utils.EventReasonCreated,
				LastTransitionTime: metav1.Now(),
				ObservedGeneration: run.Generation,
			},
			metav1.Condition{
				Type:               string(backupoperatoriov1.BackupRunConditionTypeInProgress),
				Status:             utils.ToConditionStatus(&state.InProgress),
				Reason:             utils.EventReasonCreated,
				Message:            utils.EventReasonCreated,
				LastTransitionTime: metav1.Now(),
				ObservedGeneration: run.Generation,
			},
			metav1.Condition{
				Type:               string(backupoperatoriov1.BackupRunConditionTypeFailed),
				Status:             utils.ToConditionStatus(&state.Failed),
				Reason:             utils.EventReasonCreated,
				Message:            utils.EventReasonCreated,
				LastTransitionTime: metav1.Now(),
				ObservedGeneration: run.Generation,
			},
			metav1.Condition{
				Type:               string(backupoperatoriov1.BackupRunConditionTypeSuccessful),
				Status:             utils.ToConditionStatus(&state.Successful),
				Reason:             utils.EventReasonCreated,
				Message:            utils.EventReasonCreated,
				LastTransitionTime: metav1.Now(),
				ObservedGeneration: run.Generation,
			},
			metav1.Condition{
				Type:               string(backupoperatoriov1.BackupRunConditionTypeRestorable),
				Status:             utils.ToConditionStatus(&state.Restorable),
				Reason:             utils.EventReasonCreated,
				Message:            restorableMsg,
				LastTransitionTime: metav1.Now(),
				ObservedGeneration: run.Generation,
			},
			metav1.Condition{
				Type:               string(backupoperatoriov1.BackupRunConditionTypeEncrypted),
				Status:             utils.ToConditionStatus(&state.Encrypted),
				Reason:             utils.EventReasonCreated,
				Message:            encryptedMsg,
				LastTransitionTime: metav1.Now(),
				ObservedGeneration: run.Generation,
			},
			metav1.Condition{
				Type:               string(backupoperatoriov1.BackupRunConditionTypeCompressed),
				Status:             utils.ToConditionStatus(&state.Compressed),
				Reason:             utils.EventReasonCreated,
				Message:            compressedMsg,
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
	r.Recorder.Eventf(run, corev1.EventTypeNormal, "Reconciled", "Successfully reconciled")
	// Update metrics
	backuprun.UpdateMetricsStatus(run)
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
		log.V(1).Info("could not fetch an object")
		return
	}
	log.V(1).Info("starting run deletion")
	// Update metrics
	run.Status.State = ptr.To("Deleted")
	backuprun.UpdateMetricsStatus(run)
	// Delete backup from storage...
	if *run.Spec.RetainPolicy == backupoperatoriov1.BackupRetainDelete {
		// ...if retain is set to Delete
		log.V(1).Info("trying to delete backup from storage according to spec.retainPolicy=Delete")
		var storage backupstorage.BackupStorageProvider
		if storage, ok = backupstorage.GetBackupStorageProvider(run.Spec.Storage.Name); !ok {
			log.V(1).Info("failed to get backupStorageProvider", "storageName", run.Spec.Storage.Name)
			// Deletion is best effort, if it fails - no need to reschedule
			return result, nil
		}
		log = log.WithValues("storageName", run.Spec.Storage.Name, "backupPath", run.Spec.Storage.Path)
		log.V(1).Info("storage provider has been found, making deletion at path")
		if err := storage.Delete(ctx, run.Spec.Storage.Path); err != nil {
			log.V(1).Error(err, "failed to delete the backup")
			// Deletion is best effort, if it fails - no need to reschedule
			return result, nil
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
		log.V(1).Info("could not fetch an object")
		return
	}
	log.V(1).Info("run processing")
	// Trying to get backup storage provider...
	var storage backupstorage.BackupStorageProvider
	if storage, ok = backupstorage.GetBackupStorageProvider(run.Spec.Storage.Name); !ok {
		// ...if fail - reschedule
		result.RequeueAfter = time.Second * 20
		return
	}
	// Analyze run conditions
	state := backuprun.AnalyzeRunConditions(run)
	if state.Interrupted {
		r.Recorder.Eventf(run, corev1.EventTypeWarning, "Failure", "Run has been interrupted")
		backuprun.ChangeRunState(ctx, r.Client, run, backupoperatoriov1.BackupRunConditionTypeFailed, state)
		return
	}
	// Exit if we do not have to run
	if !state.HaveToBackup && !state.HaveToRestore {
		return
	}
	// Set InProgress to true
	if err = backuprun.ChangeRunState(ctx, r.Client, run, backupoperatoriov1.BackupRunConditionTypeInProgress, state); err != nil {
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
	log.V(1).Info("run preparing new pod")
	if err = backuprun.CreatePodFromRun(ctx, r.Client, r.Scheme, r.Config, run, pod); err != nil {
		log.V(1).Error(err, "failed to create run pod")
		// Fail the run
		backuprun.ChangeRunState(ctx, r.Client, run, backupoperatoriov1.BackupRunConditionTypeFailed, state)
		return
	}
	// Schedule pod deletion at the end of function
	defer func() {
		log.V(1).Info("deleting run pod")
		if err = r.Client.Delete(ctx, pod, &client.DeleteOptions{
			PropagationPolicy: ptr.To(metav1.DeletePropagationForeground),
		}); err != nil {
			log.Error(err, "failed to delete backup run pod at the very end")
			return
		}
	}()
	// Backup or Restore
	switch {
	case state.HaveToBackup:
		log.V(1).Info("run calling backup action")
		if err = backuprun.Backup(ctx, r.Client, r.Scheme, r.Config, run, pod, storage); err != nil {
			backuprun.ChangeRunState(ctx, r.Client, run, backupoperatoriov1.BackupRunConditionTypeFailed, state)
			return
		}
		if err = backuprun.SetBackupSizeInStatus(ctx, r.Client, run, storage); err != nil {
			log.V(1).Error(err, "failed to set backup size")
			// No need to fail, that is not critical
		}
	case state.HaveToRestore:
		log.V(1).Info("run calling restore action")
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
			r.Recorder.Eventf(run, corev1.EventTypeWarning, "Restoration", "Restoration has failed")
			return
		}
		// Make successful restoration event and update annotations
		r.Recorder.Eventf(run, corev1.EventTypeNormal, "Restoration", "Restoration has been completed successfully")
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
	log.V(1).Info("finished backup run")
	return
}

// SetupWithManager sets up the controller with the Manager.
func (r *BackupRunReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&backupoperatoriov1.BackupRun{}).
		Owns(&corev1.Pod{}).
		WithOptions(controller.Options{MaxConcurrentReconciles: 10}).
		WithEventFilter(utils.IgnoreOutOfOrder()).
		WithEventFilter(utils.IgnoreDeletionPredicate()).
		Complete(r)
}
