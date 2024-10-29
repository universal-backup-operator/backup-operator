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
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

func (r *BackupStorage) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

//+kubebuilder:webhook:path=/mutate-backup-operator-io-v1-backupstorage,mutating=true,failurePolicy=fail,sideEffects=None,groups=backup-operator.io,resources=backupstorages,verbs=create,versions=v1,name=mbackupstorage.kb.io,admissionReviewVersions=v1

var _ webhook.Defaulter = &BackupStorage{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
// Launched on CREATE only
func (r *BackupStorage) Default() {
	a := r.GetAnnotations()
	if a == nil {
		a = make(map[string]string)
	}
	a[AnnotationDeletionProtection] = "true"
	r.SetAnnotations(a)
}

//+kubebuilder:webhook:path=/validate-backup-operator-io-v1-backupstorage,mutating=false,failurePolicy=fail,sideEffects=None,groups=backup-operator.io,resources=backupstorages,verbs=create;update;delete,versions=v1,name=vbackupstorage.kb.io,admissionReviewVersions=v1

var _ webhook.Validator = &BackupStorage{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *BackupStorage) ValidateCreate() (admission.Warnings, error) {
	return nil, nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *BackupStorage) ValidateUpdate(old runtime.Object) (admission.Warnings, error) {
	return nil, nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *BackupStorage) ValidateDelete() (admission.Warnings, error) {
	return nil, r.validateBackupStorageDeletion()
}

// validateBackupStorageDeletion checks the BackupStorage object for deletion correctness by
// validating its deletion protection. It calls validateBackupStorageDeletionProtection to check
// if the object is protected from deletion using an annotation. If the object is protected,
// it returns a field.Error indicating the validation error, otherwise it returns nil.
func (r *BackupStorage) validateBackupStorageDeletion() error {
	var allErrs field.ErrorList
	if err := r.validateBackupStorageDeletionProtection(); err != nil {
		allErrs = append(allErrs, err)
	}
	if len(allErrs) == 0 {
		return nil
	}
	return apierrors.NewInvalid(
		schema.GroupKind{
			Group: r.GroupVersionKind().Group,
			Kind:  r.Kind,
		}, r.Name, allErrs)
}

// validateBackupStorageDeletionProtection validates the deletion protection of the BackupStorage object.
// It checks if the object is protected from deletion using an annotation. If the object is protected,
// it returns a field.Error indicating the validation error, otherwise it returns nil.
func (r *BackupStorage) validateBackupStorageDeletionProtection() (err *field.Error) {
	if _, protected := r.GetAnnotations()[AnnotationDeletionProtection]; protected {
		fld := field.NewPath("metadata").Child("annotations").Child(AnnotationDeletionProtection)
		msg := "object is protected from deletion with annotation, remove it if you know what you do"
		err = field.Invalid(fld, r.Name, msg)
	}
	return
}
