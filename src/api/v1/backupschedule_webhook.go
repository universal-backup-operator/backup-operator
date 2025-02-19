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
	"time"

	"github.com/robfig/cron/v3"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

func (r *BackupSchedule) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

//+kubebuilder:webhook:path=/validate-backup-operator-io-v1-backupschedule,mutating=false,failurePolicy=fail,sideEffects=None,groups=backup-operator.io,resources=backupschedules,verbs=create;update;delete,versions=v1,name=vbackupschedule.kb.io,admissionReviewVersions=v1

var _ webhook.CustomValidator = &BackupSchedule{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *BackupSchedule) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	return r.validate(ctx, obj)
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *BackupSchedule) ValidateUpdate(ctx context.Context, _, obj runtime.Object) (admission.Warnings, error) {
	return r.validate(ctx, obj)
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *BackupSchedule) ValidateDelete(_ context.Context, _ runtime.Object) (admission.Warnings, error) {
	return nil, nil
}

// validate checks the overall BackupSchedule for correctness by validating its name,
// run spec, and schedule format. It calls validateName to validate the name,
// validateRunSpec to validate the run spec, and validateCron to
// validate the schedule format. Any validation errors are aggregated into a field.ErrorList.
// If there are no validation errors, it returns nil. Otherwise, it returns an apierrors.Invalid
// error containing the aggregated field.ErrorList.
func (r *BackupSchedule) validate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	log := log.FromContext(ctx)
	schedule, ok := obj.(*BackupSchedule)
	if !ok {
		return nil, fmt.Errorf("expected a BackupSchedule but got a %T", obj)
	}
	log.V(1).Info("Validating BackupSchedule")

	var allErrs field.ErrorList
	if err := schedule.validateName(); err != nil {
		allErrs = append(allErrs, err)
	}
	if err := schedule.validateRunSpec(); err != nil {
		allErrs = append(allErrs, err)
	}
	if err := schedule.validateCron(); err != nil {
		allErrs = append(allErrs, err)
	}
	if len(allErrs) == 0 {
		return nil, nil
	}
	return nil, apierrors.NewInvalid(
		schema.GroupKind{
			Group: schedule.GroupVersionKind().Group,
			Kind:  schedule.Kind,
		}, schedule.Name, allErrs)
}

// validateRunSpec validates the BackupRun spec for a BackupSchedule.
// It creates a new BackupRun object based on the BackupSchedule and its template spec, and
// then validates the BackupRun spec using validateBackupRunSpec.
// Returns nil if the spec is valid, otherwise returns a field.Error indicating the validation error.
func (r *BackupSchedule) validateRunSpec() (err *field.Error) {
	run := &BackupRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-%d", r.Name, time.Now().Unix()),
			Namespace: r.Namespace,
		},
		Spec: *r.Spec.Template.Spec.DeepCopy(),
	}
	err = run.validateSpec()
	return
}

// validateName checks the length of the BackupSchedule name to ensure it meets the
// requirements for creating corresponding jobs. Kubernetes object names must fit in a DNS subdomain
// and have a maximum length of 63 characters. The controller appends an 11-character suffix
// (`-$TIMESTAMP`) to the job name when creating a job. Therefore, BackupSchedule names must have a
// length <= 52 characters to account for the additional suffix. If the name is too long, the function
// returns a field.Error indicating the validation error, otherwise it returns nil.
func (r *BackupSchedule) validateName() (err *field.Error) {
	if len(r.ObjectMeta.Name) > validation.DNS1035LabelMaxLength-11 {
		// The job name length is 63 character like all Kubernetes objects
		// (which must fit in a DNS subdomain). The controller appends
		// a 11-character suffix to the run (`-$TIMESTAMP`) when creating
		// a job. The job name length limit is 63 characters. Therefore schedule
		// names must have length <= 63-11=52. If we don't validate this here,
		// then job creation will fail later.
		fld := field.NewPath("metadata").Child("name")
		msg := "must be no more than 52 characters"
		err = field.Invalid(fld, r.Name, msg)
	}
	return
}

// validateSchedule checks the schedule format of the BackupSchedule.
// It uses the cron.ParseStandard function to parse the schedule string. If the schedule
// string is unparseable, the function returns a field.Error indicating the validation
// error, otherwise it returns nil.
func (r *BackupSchedule) validateCron() (err *field.Error) {
	if _, err := cron.ParseStandard(r.Spec.Schedule); err != nil {
		fld := field.NewPath("spec").Child("schedule")
		msg := fmt.Sprintf("unparseable schedule: %s", err.Error())
		return field.Invalid(fld, r.Name, msg)
	}
	return
}
