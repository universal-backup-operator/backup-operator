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
	"sync"

	"github.com/creasty/defaults"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/retry"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"

	backupoperatoriov1 "backup-operator.io/api/v1"
	backupstorage "backup-operator.io/internal/controller/backupStorage"
	backupstorageproviders "backup-operator.io/internal/controller/backupStorage/providers"
	utils "backup-operator.io/internal/controller/utils"
	"backup-operator.io/internal/monitoring"
)

// BackupStorageReconciler reconciles a BackupStorage object
type BackupStorageReconciler struct {
	client.Client
	Config   *rest.Config
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

//+kubebuilder:rbac:groups=backup-operator.io,resources=backupstorages,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=backup-operator.io,resources=backupstorages/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=backup-operator.io,resources=backupstorages/finalizers,verbs=update
//+kubebuilder:rbac:groups=backup-operator.io,resources=backupschedules,verbs=get;list;watch
//+kubebuilder:rbac:groups=backup-operator.io,resources=backupruns,verbs=get;list;watch
//+kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch
//+kubebuilder:rbac:groups="",resources=events,verbs=create;get;list;patch;update;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// (user): Modify the Reconcile function to compare the state specified by
// the BackupStorage object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.14.1/pkg/reconcile
func (r *BackupStorageReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	monitoring.RegisterAlerts(ctx, r.Client)
	return utils.ManageLifecycle(ctx, &utils.ManagedLifecycleReconcile{
		Client:   r.Client,
		Config:   r.Config,
		Scheme:   r.Scheme,
		Recorder: r.Recorder,
		Request:  req,
		Object:   &backupoperatoriov1.BackupStorage{},
	}, &backupStorageLifecycle{})
}

// Implements ManagedLifecycleObject interface
type backupStorageLifecycle struct{}

// Map with hashes of configured providers. If hash does not match, hence,
// we have to call provider.Constructor. We have both parameters and credentials content.
var storageProvidersConfigurationHashes = &sync.Map{}

// ┌─┐┌─┐┌┐┐┐─┐┌┐┐┬─┐┬ ┐┌┐┐┌─┐┬─┐
// │  │ ││││└─┐ │ │┬┘│ │ │ │ ││┬┘
// └─┘┘─┘┘└┘──┘ ┘ ┘└┘┘─┘ ┘ ┘─┘┘└┘

func (b *backupStorageLifecycle) Constructor(ctx context.Context, r *utils.ManagedLifecycleReconcile) (result ctrl.Result, err error) {
	storage := r.Object.(*backupoperatoriov1.BackupStorage)
	log := log.FromContext(ctx)
	if err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
		if err = r.Client.Get(ctx, client.ObjectKeyFromObject(storage), storage); err != nil {
			utils.Log(r, log, err, storage, "FailedGet", "could not get the storage")
			return err
		}
		storage.Status.Conditions = *utils.AddOrUpdateConditions(storage.Status.Conditions,
			metav1.Condition{
				Type:               backupoperatoriov1.ConditionTypeReady,
				Status:             metav1.ConditionFalse,
				Reason:             utils.EventReasonInitializing,
				Message:            utils.EventReasonInitializing,
				LastTransitionTime: metav1.Now(),
				ObservedGeneration: storage.Generation,
			},
		)
		return r.Client.Status().Update(ctx, storage)
	}); err != nil {
		return
	}
	return
}

// ┬─┐┬─┐┐─┐┌┐┐┬─┐┬ ┐┌─┐┌┐┐┌─┐┬─┐
// │ │├─ └─┐ │ │┬┘│ ││   │ │ ││┬┘
// ┘─┘┴─┘──┘ ┘ ┘└┘┘─┘└─┘ ┘ ┘─┘┘└┘

func (b *backupStorageLifecycle) Destructor(ctx context.Context, r *utils.ManagedLifecycleReconcile) (result ctrl.Result, err error) {
	storage := r.Object.(*backupoperatoriov1.BackupStorage)
	log := log.FromContext(ctx)
	if err = r.Client.Get(ctx, client.ObjectKeyFromObject(storage), storage); err != nil {
		utils.Log(r, log, err, storage, "FailedGet", "could not get the storage")
		return
	}
	// Annotation backupoperatoriov1.AnnotationDeletionProtection is honored in validation deletion webhook
	// Destruct provider
	if provider, ok := backupstorage.GetBackupStorageProvider(storage.Name); ok {
		if err = provider.Destructor(); err != nil {
			utils.Log(r, log, err, storage, "FailedDeletion", "failed to delete the storage")
		}
		backupstorage.RemoveBackupStorageProvider(storage.Name)
	}
	// Forget its configuration hash
	storageProvidersConfigurationHashes.Delete(storage.GetUID())
	// Delete metric
	backupstorage.DeleteMetric(storage)
	return
}

// ┬─┐┬─┐┌─┐┌─┐┬─┐┐─┐┐─┐┌─┐┬─┐
// │─┘│┬┘│ ││  ├─ └─┐└─┐│ ││┬┘
// ┘  ┘└┘┘─┘└─┘┴─┘──┘──┘┘─┘┘└┘

func (b *backupStorageLifecycle) Processor(ctx context.Context, r *utils.ManagedLifecycleReconcile) (result ctrl.Result, err error) {
	storage := r.Object.(*backupoperatoriov1.BackupStorage)
	log := log.FromContext(ctx)
	if err = r.Client.Get(ctx, client.ObjectKeyFromObject(storage), storage); err != nil {
		utils.Log(r, log, err, storage, "FailedGet", "could not get the storage")
		return
	}
	credentials := make(map[string]string)
	// Construct storage object
	if storage.Spec.Credentials != nil {
		// Prepare Secret object to read credentials
		secret := &corev1.Secret{}
		secret.Name = storage.Spec.Credentials.Name
		secret.Namespace = storage.Spec.Credentials.Namespace
		// Read credentials secret
		if err = r.Client.Get(ctx, client.ObjectKeyFromObject(secret), secret); err != nil {
			err = fmt.Errorf("could not get credentials secret: %s", err.Error())
			utils.Log(r, log, err, storage, "FailedGetCredentials", "")
			backupstorage.ChangeStorageReadiness(ctx, r.Client, storage, false, err.Error())
			return
		}
		credentials = utils.DecodeSecretData(secret)
	}
	// Creating respective provider
	provider, providerExists := backupstorage.GetBackupStorageProvider(storage.Name)
	switch storage.Spec.Type {
	case "s3":
		// If provider is not listed in BackupStorageProviders map - create new one
		if !providerExists {
			provider = &backupstorageproviders.S3Storage{}
			if err = defaults.Set(provider); err != nil {
				err = fmt.Errorf("could not set default provider settings: %s", err.Error())
				utils.Log(r, log, err, storage, "FailedSetDefaults", "")
				backupstorage.ChangeStorageReadiness(ctx, r.Client, storage, false, err.Error())
				return
			}
		}
	default:
		// Error if type is unknown
		err = fmt.Errorf("unknown storage type: %s", storage.Spec.Type)
		utils.Log(r, log, err, storage, "UnknownType", "")
		backupstorage.ChangeStorageReadiness(ctx, r.Client, storage, false, err.Error())
		return
	}
	// Configure provider
	hash := utils.Hash(storage.Spec.Parameters, credentials)
	// If has differs or not found...
	_, found := backupstorage.GetBackupStorageProvider(storage.Name)
	var oldHash any
	if oldHash, _ = storageProvidersConfigurationHashes.Load(storage.UID); !found || hash != oldHash {
		// ...and construct provider (for the first time or again, does not matter)
		if err = provider.Constructor(storage, storage.Spec.Parameters, credentials); err != nil {
			err = fmt.Errorf("could not configure the provider %s: %s", storage.Spec.Type, err.Error())
			utils.Log(r, log, err, storage, "FailedConfigure", "")
			backupstorage.ChangeStorageReadiness(ctx, r.Client, storage, false, err.Error())
			return
		}
		utils.Log(r, log, err, storage, "Reconciled", "(Re)configured successfully")
		// ...save hash
		storageProvidersConfigurationHashes.Store(storage.UID, hash)
		// Add backup storage provider to memory
		backupstorage.AddBackupStorageProvider(storage.Name, provider)
	}
	// Count child schedules
	if err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
		if err = r.Client.Get(ctx, client.ObjectKeyFromObject(storage), storage); err != nil {
			utils.Log(r, log, err, storage, "FailedGet", "could not get the storage")
			return err
		}
		// List child schedules
		childSchedules := &backupoperatoriov1.BackupScheduleList{}
		if err = r.List(ctx, childSchedules, client.MatchingFields{".spec.template.spec.storage.name": storage.Name}); err != nil {
			utils.Log(r, log, err, storage, "FailedGet", "could not list child schedules")
			return err
		}
		// List child runs
		childRuns := &backupoperatoriov1.BackupRunList{}
		if err = r.List(ctx, childRuns, client.MatchingFields{".spec.storage.name": storage.Name}); err != nil {
			utils.Log(r, log, err, storage, "FailedGet", "could not list child runs")
			return err
		}
		var childRunsSizeInBytes uint = 0
		for _, run := range childRuns.Items {
			if run.Status.SizeInBytes != nil {
				childRunsSizeInBytes += *run.Status.SizeInBytes
			}
		}
		storage.Status.Schedules = ptr.To(uint16(len(childSchedules.Items)))
		storage.Status.Runs = ptr.To(uint16(len(childRuns.Items)))
		storage.Status.SizeInBytes = ptr.To(childRunsSizeInBytes)
		storage.Status.Size = ptr.To(utils.ConvertBytesToHumanReadable(childRunsSizeInBytes))
		backupstorage.ChangeStorageReadiness(ctx, r.Client, storage, true, "Provider object is reconfigured and ready to work")
		return r.Client.Status().Update(ctx, storage)
	}); err != nil {
		backupstorage.ChangeStorageReadiness(ctx, r.Client, storage, false, err.Error())
		return
	}
	// Update metrics
	backupstorage.UpdateMetric(storage)
	return
}

var backupStorageIndexers = map[string]client.IndexerFunc{
	".metadata.controller": func(o client.Object) []string {
		owner := metav1.GetControllerOf(o)
		if owner == nil {
			return nil
		}
		return []string{string(owner.UID)}
	},
}

// SetupWithManager sets up the controller with the Manager.
func (r *BackupStorageReconciler) SetupWithManager(mgr ctrl.Manager) error {
	for path, function := range backupStorageIndexers {
		if err := mgr.GetFieldIndexer().IndexField(context.Background(),
			&backupoperatoriov1.BackupStorage{}, path, function); err != nil {
			return err
		}
	}
	return ctrl.NewControllerManagedBy(mgr).
		For(&backupoperatoriov1.BackupStorage{}).
		Watches(&backupoperatoriov1.BackupSchedule{}, handler.EnqueueRequestsFromMapFunc(
			func(ctx context.Context, o client.Object) []reconcile.Request {
				schedule := o.(*backupoperatoriov1.BackupSchedule)
				return []reconcile.Request{
					{
						NamespacedName: types.NamespacedName{
							Name: schedule.Spec.Template.Spec.Storage.Name,
						},
					},
				}
			}),
		).
		Watches(&backupoperatoriov1.BackupRun{}, handler.EnqueueRequestsFromMapFunc(
			func(ctx context.Context, o client.Object) []reconcile.Request {
				run := o.(*backupoperatoriov1.BackupRun)
				return []reconcile.Request{
					{
						NamespacedName: types.NamespacedName{
							Name: run.Spec.Storage.Name,
						},
					},
				}
			}),
		).
		WithOptions(controller.Options{MaxConcurrentReconciles: 10}).
		Complete(r)
}
