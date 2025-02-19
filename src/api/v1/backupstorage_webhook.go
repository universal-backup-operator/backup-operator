/*
Copyright 2024.

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

package v1

import (
	"context"
	"fmt"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

func (r *BackupStorage) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

//+kubebuilder:webhook:path=/mutate-backup-operator-io-v1-backupstorage,mutating=true,failurePolicy=fail,sideEffects=None,groups=backup-operator.io,resources=backupstorages,verbs=create,versions=v1,name=mbackupstorage.kb.io,admissionReviewVersions=v1

var _ webhook.CustomDefaulter = &BackupStorage{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
// Launched on CREATE only
func (r *BackupStorage) Default(ctx context.Context, obj runtime.Object) (err error) {
	log := log.FromContext(ctx)
	storage, ok := obj.(*BackupStorage)
	if !ok {
		return fmt.Errorf("expected a BackupStorage but got a %T", obj)
	}
	log.V(1).Info("Validating BackupStorage")
	a := storage.GetAnnotations()
	if a == nil {
		a = make(map[string]string)
	}
	a[AnnotationDeletionProtection] = "true"
	storage.SetAnnotations(a)
	return
}

//+kubebuilder:webhook:path=/validate-backup-operator-io-v1-backupstorage,mutating=false,failurePolicy=fail,sideEffects=None,groups=backup-operator.io,resources=backupstorages,verbs=create;update;delete,versions=v1,name=vbackupstorage.kb.io,admissionReviewVersions=v1

var _ webhook.CustomValidator = &BackupStorage{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *BackupStorage) ValidateCreate(_ context.Context, _ runtime.Object) (admission.Warnings, error) {
	return nil, nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *BackupStorage) ValidateUpdate(_ context.Context, _, _ runtime.Object) (admission.Warnings, error) {
	return nil, nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *BackupStorage) ValidateDelete(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	return r.validateBackupStorageDeletion(ctx, obj)
}

// validateBackupStorageDeletion checks the BackupStorage object for deletion correctness by
// validating its deletion protection. It calls validateDeletionProtection to check
// if the object is protected from deletion using an annotation. If the object is protected,
// it returns a field.Error indicating the validation error, otherwise it returns nil.
func (r *BackupStorage) validateBackupStorageDeletion(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	log := log.FromContext(ctx)
	storage, ok := obj.(*BackupStorage)
	if !ok {
		return nil, fmt.Errorf("expected a BackupStorage but got a %T", obj)
	}
	log.V(1).Info("Validating BackupStorage")

	var allErrs field.ErrorList
	if err := storage.validateDeletionProtection(); err != nil {
		allErrs = append(allErrs, err)
	}
	if len(allErrs) == 0 {
		return nil, nil
	}
	return nil, apierrors.NewInvalid(
		schema.GroupKind{
			Group: storage.GroupVersionKind().Group,
			Kind:  storage.Kind,
		}, storage.Name, allErrs)
}

// validateDeletionProtection validates the deletion protection of the BackupStorage object.
// It checks if the object is protected from deletion using an annotation. If the object is protected,
// it returns a field.Error indicating the validation error, otherwise it returns nil.
func (r *BackupStorage) validateDeletionProtection() (err *field.Error) {
	if _, protected := r.GetAnnotations()[AnnotationDeletionProtection]; protected {
		fld := field.NewPath("metadata").Child("annotations").Child(AnnotationDeletionProtection)
		msg := "object is protected from deletion with annotation, remove it if you know what you do"
		err = field.Invalid(fld, r.Name, msg)
	}
	return
}
