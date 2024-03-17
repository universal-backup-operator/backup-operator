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

package v1

import (
	"encoding/json"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

/* BackupScheduleSpec defines the desired state of BackupSchedule. */
type BackupScheduleSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// The schedule in Cron format, see https://en.wikipedia.org/wiki/Cron.
	//+kubebuilder:validation:MinLength=1
	Schedule string `json:"schedule" protobuf:"bytes,1,req,name=schedule"`

	// The time zone name for the given schedule, see https://en.wikipedia.org/wiki/List_of_tz_database_time_zones.
	// If not specified, this will default to the time zone of the kube-controller-manager process.
	// The set of valid time zone names and the time zone offset is loaded from the system-wide time zone
	// database by the API server during CronJob validation and the controller manager during execution.
	// If no system-wide time zone database can be found a bundled version of the database is used instead.
	// If the time zone name becomes invalid during the lifetime of a CronJob or due to a change in host
	// configuration, the controller will stop creating new new Jobs and will create a system event with the
	// reason UnknownTimeZone.
	// More information can be found in https://kubernetes.io/docs/concepts/workloads/controllers/cron-jobs/#time-zones
	//+kubebuilder:validation:MinLength=1
	//+kubebuilder:validation:Optional
	TimeZone *string `json:"timeZone,omitempty" protobuf:"bytes,2,opt,name=timeZone"`

	// Optional deadline in seconds for starting the job if it misses scheduled
	// time for any reason.  Missed jobs executions will be counted as failed ones.
	//+kubebuilder:validation:Minimum=1
	//+kubebuilder:validation:Optional
	StartingDeadlineSeconds *uint64 `json:"startingDeadlineSeconds,omitempty" protobuf:"varint,3,opt,name=startingDeadlineSeconds"`

	// Specifies how to treat concurrent executions.
	// Valid values are:
	//
	// - "Allow" (default): allows CronJobs to run concurrently;
	// - "Forbid": forbids concurrent runs, skipping next run if previous run hasn't finished yet;
	// - "Replace": cancels currently running job and replaces it with a new one
	//+kubebuilder:default=Replace
	//+kubebuilder:validation:Optional
	ConcurrencyPolicy *batchv1.ConcurrencyPolicy `json:"concurrencyPolicy,omitempty" protobuf:"bytes,4,opt,name=concurrencyPolicy,casttype=ConcurrencyPolicy"`

	// This flag tells the controller to suspend subsequent executions, it does
	// not apply to already started executions.  Defaults to false.
	//+kubebuilder:default=false
	//+kubebuilder:validation:Optional
	Suspend *bool `json:"suspend,omitempty" protobuf:"varint,5,opt,name=suspend"`

	// BackupRun template configuration to make backup with.
	// All scheduled backups will be created with this exact configuration.
	Template *backupRunTemplate `json:"template" protobuf:"bytes,6,opt,name=template"`

	// The number of successful finished runs to retain. Value must be non-negative integer.
	// In order to keep BackupRun forever - annotate it with backup-operator.io/keep (value does not matter)
	//
	// Defaults to 3.
	//+kubebuilder:default=3
	//+kubebuilder:validation:Minimum=1
	//+kubebuilder:validation:Optional
	SuccessfulRunsHistoryLimit *uint16 `json:"successfulRunsHistoryLimit,omitempty" protobuf:"varint,7,opt,name=successfulRunsHistoryLimit"`

	// The number of failed finished runs to retain. Value must be non-negative integer.
	// In order to keep BackupRun forever - annotate it with backup-operator.io/keep (value does not matter)
	//
	// Defaults to 1.
	//+kubebuilder:default=1
	//+kubebuilder:validation:Minimum=1
	//+kubebuilder:validation:Optional
	FailedRunsHistoryLimit *uint16 `json:"failedRunsHistoryLimit,omitempty" protobuf:"varint,8,opt,name=failedRunsHistoryLimit"`
}

/* Backup Run definition with metadata and spec. */
type backupRunTemplate struct {
	/* Backup Run custom metadata. */
	//+kubebuilder:validation:Optional
	Metadata *LabelsAndAnnotationsMetadata `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	/* Backup Run custom specification. */
	Spec BackupRunSpec `json:"spec" protobuf:"bytes,2,req,name=spec"`
}

/* BackupScheduleStatus defines the observed state of BackupSchedule. */
type BackupScheduleStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Conditions store
	// +operator-sdk:csv:customresourcedefinitions:type=status
	// +patchMergeKey=type
	// +patchStrategy=merge
	// +listType=map
	// +listMapKey=type
	//+kubebuilder:validation:Optional
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,1,rep,name=conditions"`

	/* Total count of backups. */
	//+kubebuilder:default=0
	//+kubebuilder:validation:Minimum=0
	//+kubebuilder:validation:Optional
	Total *uint16 `json:"total,omitempty" protobuf:"varint,2,opt,name=total"`

	/* Count of successful backups. */
	//+kubebuilder:default=0
	//+kubebuilder:validation:Minimum=0
	//+kubebuilder:validation:Optional
	Successful *uint16 `json:"successful,omitempty" protobuf:"varint,3,opt,name=successful"`

	/* Count of failed backups. */
	//+kubebuilder:default=0
	//+kubebuilder:validation:Minimum=0
	//+kubebuilder:validation:Optional
	Failed *uint16 `json:"failed,omitempty" protobuf:"varint,4,opt,name=failed"`

	/* Count of active backups. */
	//+kubebuilder:default=0
	//+kubebuilder:validation:Minimum=0
	//+kubebuilder:validation:Optional
	InProgress *uint16 `json:"inProgress,omitempty" protobuf:"varint,5,opt,name=inProgress"`

	/* A list of pointers to currently running runs. */
	//+listType=atomic
	//+kubebuilder:validation:Optional
	Active []corev1.ObjectReference `json:"active,omitempty" protobuf:"bytes,6,rep,name=active"`

	/* Information when was the last time the job was successfully scheduled. */
	//+kubebuilder:validation:Optional
	LastScheduleTime *metav1.Time `json:"lastScheduleTime,omitempty" protobuf:"bytes,7,opt,name=lastScheduleTime"`

	/* Information when was the last time the job successfully completed. */
	//+kubebuilder:validation:Optional
	LastSuccessfulTime *metav1.Time `json:"lastSuccessfulTime,omitempty" protobuf:"bytes,8,opt,name=lastSuccessfulTime"`
}

func (s *BackupScheduleStatus) String() string {
	var j []byte
	var err error
	if j, err = json.Marshal(*s); err != nil {
		panic(err)
	}
	return string(j)
}

/*
Backups of MySQL, PostgreSQL, MongoDB, etc.
Supports backups encryption and compression.
Here you configure connection string, schedule and rotation.
It creates BackupRun objects on schedule.
*/
//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:shortName=bacsc;backsc;backupsc
//+kubebuilder:printcolumn:name="Schedule",type=string,JSONPath=`.spec.schedule`,description="CRON schedule"
//+kubebuilder:printcolumn:name="Storage",type=string,JSONPath=`.spec.template.spec.storage.name`,description="Backup storage"
//+kubebuilder:printcolumn:name="Total",type=integer,JSONPath=`.status.total`,description="Total backups count"
//+kubebuilder:printcolumn:name="In Progress",type=integer,JSONPath=`.status.inProgress`,description="Total backups count",priority=1
//+kubebuilder:printcolumn:name="Successful",type=integer,JSONPath=`.status.successful`,description="Successful backups count",priority=1
//+kubebuilder:printcolumn:name="Failed",type=integer,JSONPath=`.status.failed`,description="Failed backups count",priority=1
//+kubebuilder:printcolumn:name="Last Scheduled",type=date,format=date-time,JSONPath=`.status.lastScheduleTime`,description="Last schedule time"
//+kubebuilder:printcolumn:name="Last Successful",type=date,format=date-time,JSONPath=`.status.lastSuccessfulTime`,description="Last successful run"
type BackupSchedule struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata" protobuf:"bytes,3,req,name=metadata"`

	Spec   BackupScheduleSpec   `json:"spec" protobuf:"bytes,4,req,name=metadata"`
	Status BackupScheduleStatus `json:"status,omitempty" protobuf:"bytes,5,opt,name=metadata"`
}

/* BackupScheduleList contains a list of BackupSchedule. */
//+kubebuilder:object:root=true
type BackupScheduleList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty" protobuf:"bytes,3,opt,name=metadata"`
	Items           []BackupSchedule `json:"items" protobuf:"bytes,4,req,name=items"`
}

func init() {
	SchemeBuilder.Register(&BackupSchedule{}, &BackupScheduleList{})
}
