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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// +kubebuilder:validation:Enum=s3
type storageType string

const (
	// S3 storage type name
	S3 storageType = "s3"
)

/* BackupStorageSpec defines the desired state of BackupStorage. */
type BackupStorageSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	/* Type of the storage. At the moment, only S3 is supported.
	Valid values: s3
	Default: s3 */
	//+kubebuilder:default="s3"
	//+kubebuilder:example="s3"
	Type storageType `json:"type" protobuf:"bytes,1,req,name=type"`

	/* Extra provisioner configuration options if any. */
	//+kubebuilder:validation:Optional
	Parameters map[string]string `json:"parameters,omitempty" protobuf:"bytes,2,rep,name=parameters"`

	/* Credentials to use for connection. You can select exact keys adding overrides in parameters. */
	//+kubebuilder:validation:Optional
	Credentials *secretReferenceRequireNamespace `json:"credentials,omitempty" protobuf:"bytes,3,opt,name=credentials"`
}

/* BackupStorageStatus defines the observed state of BackupStorage. */
type BackupStorageStatus struct {
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

	/* Total count of schedules. */
	//+kubebuilder:default=0
	//+kubebuilder:validation:Minimum=0
	//+kubebuilder:validation:Optional
	Schedules *uint16 `json:"schedules,omitempty" protobuf:"varint,2,opt,name=schedules"`

	/* Total count of runs. */
	//+kubebuilder:default=0
	//+kubebuilder:validation:Minimum=0
	//+kubebuilder:validation:Optional
	Runs *uint16 `json:"runs,omitempty" protobuf:"varint,3,opt,name=runs"`

	/* Total occupied size by child BackupRuns. */
	//+kubebuilder:validation:Optional
	Size *string `json:"size,omitempty" protobuf:"bytes,4,opt,name=size"`

	/* Same as size, but in bytes. */
	//+kubebuilder:default=0
	//+kubebuilder:validation:Minimum=0
	//+kubebuilder:validation:Optional
	SizeInBytes *uint `json:"sizeInBytes,omitempty" protobuf:"varint,5,opt,name=sizeInBytes"`
}

/*
BackupStorage points to some remote storage, like S3, NFS, etc.
It depends on what is implemented in the controller. BackupRun objects make backups.
and upload backups to the place defined in these BackupStorages for long time storage.
*/
//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:scope=Cluster,shortName=bt
//+kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`,description="Readiness"
//+kubebuilder:printcolumn:name="Type",type=string,JSONPath=`.spec.type`,description="Storage type"
//+kubebuilder:printcolumn:name="Schedules",type=integer,JSONPath=`.status.schedules`,description="Count of child schedules"
//+kubebuilder:printcolumn:name="Runs",type=integer,JSONPath=`.status.runs`,description="Count of child runs"
//+kubebuilder:printcolumn:name="Size",type=string,JSONPath=`.status.size`,description="Total occupied storage size",priority=1
type BackupStorage struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,3,req,name=metadata"`

	Spec   BackupStorageSpec   `json:"spec,omitempty" protobuf:"bytes,4,req,name=metadata"`
	Status BackupStorageStatus `json:"status,omitempty" protobuf:"bytes,5,opt,name=metadata"`
}

/* BackupStorageList contains a list of BackupStorage. */
//+kubebuilder:object:root=true
type BackupStorageList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty" protobuf:"bytes,3,opt,name=metadata"`
	Items           []BackupStorage `json:"items" protobuf:"bytes,4,req,name=items"`
}

func init() {
	SchemeBuilder.Register(&BackupStorage{}, &BackupStorageList{})
}
