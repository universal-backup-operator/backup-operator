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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// +kubebuilder:validation:Enum=Delete;Retain
type BackupRetainPolicy string

const (
	BackupRetainDelete BackupRetainPolicy = "Delete"
	BackupRetainRetain BackupRetainPolicy = "Retain"
)

/* BackupRunSpec defines the desired state of BackupRun. */
type BackupRunSpec struct {
	/*
		Controls whether backup file will be removed from BackupStorage after deletion of BackupRun object.

		Possible values:
		- Delete;
		- Retain;
	*/
	//+kubebuilder:default="Retain"
	RetainPolicy *BackupRetainPolicy `json:"retainPolicy" protobuf:"bytes,1,opt,name=retainPolicy"`

	/* Backup action configuration. */
	//+kubebuilder:validation:Optional
	Backup *BackupRunAction `json:"backup,omitempty" protobuf:"bytes,3,opt,name=backup"`

	/* Restoration action configuration. May be omitted if not needed. */
	//+kubebuilder:validation:Optional
	Restore *BackupRunAction `json:"restore,omitempty" protobuf:"bytes,4,opt,name=restore"`

	/* Destination where backup must be copied. */
	Storage *backupStorage `json:"storage" protobuf:"bytes,5,opt,name=storage"`

	/* Compression configuration. */
	//+kubebuilder:validation:Optional
	Compression *backupCompression `json:"compression,omitempty" protobuf:"bytes,6,opt,name=compression"`

	/* Encryption configuration. */
	//+kubebuilder:validation:Optional
	Encryption *backupEncryption `json:"encryption,omitempty" protobuf:"bytes,7,opt,name=encryption"`

	/*
		Backup Pod template definition with metadata and spec like in Pod.
		Make sure that container for executing backup action will have 'sleep 1d' command set or similar.
		Just make sure it will stay alive long enough.
	*/
	Template *pod `json:"template" protobuf:"bytes,7,opt,name=template"`
}

/* Backup creation or restoration command to execute. */
type BackupRunAction struct {
	/* Name of Pod container to execute command in. */
	//+kubebuilder:validation:MinLength=1
	Container string `json:"container" protobuf:"bytes,1,req,name=container"`

	/* Command to execute in container. It is like Pod.spec.containers.command.
	Command must stream backup data directly to stdout. */
	//+kubebuilder:validation:MinItems=1
	Command []string `json:"command" protobuf:"bytes,2,rep,name=command"`

	/* Arguments to pass to command. It is like Pod.spec.containers.args. */
	//+kubebuilder:validation:MinItems=1
	//+kubebuilder:validation:Optional
	Args []string `json:"args,omitempty" protobuf:"bytes,3,rep,name=args"`

	/* Optional deadline in seconds for action to complete. */
	//+kubebuilder:validation:Minimum=1
	//+kubebuilder:validation:Optional
	DeadlineSeconds *uint `json:"deadlineSeconds,omitempty" protobuf:"varint,4,opt,name=deadlineSeconds"`
}

/* Storage configuration for particular backup. */
type backupStorage struct {
	/* Name of BackupStorage target object. */
	//+kubebuilder:validation:MinLength=1
	Name string `json:"name" protobuf:"bytes,1,req,name=name"`

	/* Backup file full path template. Templated with Sprig.
	See http://masterminds.github.io/sprig/ for details.
	It is on you to add compression/encryption extensions to make file name clear (e.g. .gz, .gz.age)

	Example: {{ now | date "20060102-150405" | printf "/mysql/%s.sql.gz" }}
	Result: /mysql/20231231-245959.sql.gz
	Default: {{ now | date "20060102-150405" | printf "/%s.backup" }} */
	//+kubebuilder:validation:MinLength=1
	//+kubebuilder:default=`{{ now | date "20060102-150405" | printf "/%s.backup" }}`
	//+kubebuilder:example=`/my-backups/{{ now | date "2006.01.02-15:04:05" }}.tgz`
	Path string `json:"path" protobuf:"bytes,2,req,name=path"`
}

// +kubebuilder:validation:Enum=gzip
type compressionAlgorithm string

const (
	GZIP compressionAlgorithm = "gzip"
)

/* Backup compression options. */
type backupCompression struct {
	/* Check https://pkg.go.dev/compress for available algorithms.
	Valid values: gzip
	Example: gzip
	Default: gzip */
	//+kubebuilder:default="gzip"
	//+kubebuilder:example="gzip"
	Algorithm compressionAlgorithm `json:"algorithm" protobuf:"bytes,1,req,name=algorithm"`

	/* Compression level according to https://pkg.go.dev/compress/flate#pkg-constants values.
	May be equal to any number between BestSpeed and BestCompression.
	Minimum: -2
	Maximum: 9
	Default: 0 */
	//+kubebuilder:default=0
	//+kubebuilder:validation:Minimum=-2
	//+kubebuilder:validation:Maximum=9
	Level int8 `json:"level" protobuf:"varint,2,req,name=level"`
}

/* Backup encryption options */
type backupEncryption struct {
	/* Recipients list to encrypt with.
	We use Age for encryption https://github.com/FiloSottile/age.
	Pattern: ^age1-.+|ssh-.+ */
	//+kubebuilder:validation:MinItems=1
	Recipients []string `json:"recipients" protobuf:"bytes,1,rep,name=recipients"`

	/* Decryption key if you need automatic restoration. May be omitted. */
	//+kubebuilder:validation:Optional
	DecryptionKey *secretKeyReference `json:"decryptionKey,omitempty" protobuf:"bytes,2,opt,name=decryptionKey"`
}

/* Backup Pod definition with metadata and spec. */
type pod struct {
	/* Backup Pod custom metadata. */
	//+kubebuilder:validation:Optional
	Metadata *TemplateMetadata `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	/* Backup Pod custom specification. */
	Spec corev1.PodSpec `json:"spec" protobuf:"bytes,2,req,name=spec"`
}

// +kubebuilder:validation:Enum=Idle;InProgress;Successful;Failed
type BackupRunConditionType string

const (
	// Backup or restoration has been never run
	BackupRunConditionTypeNeverRun BackupRunConditionType = "NeverRun"
	// Backup is in progress
	BackupRunConditionTypeInProgress BackupRunConditionType = "InProgress"
	// Backup has finished successfully
	BackupRunConditionTypeSuccessful BackupRunConditionType = "Successful"
	// Backup has finished with an error
	BackupRunConditionTypeFailed BackupRunConditionType = "Failed"
	// May be restored automatically
	BackupRunConditionTypeRestorable BackupRunConditionType = "Restorable"
	// Is encrypted, message will contain public key that was used for encryption
	BackupRunConditionTypeEncrypted BackupRunConditionType = "Encrypted"
	// Is compressed, message will contain algorithm
	BackupRunConditionTypeCompressed BackupRunConditionType = "Compressed"
)

/* BackupRunStatus defines the observed state of BackupRun. */
type BackupRunStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	/* Conditions store. */
	//+operator-sdk:csv:customresourcedefinitions:type=status
	//+patchMergeKey=type
	//+patchStrategy=merge
	//+listType=map
	//+listMapKey=type
	//+kubebuilder:validation:Optional
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,1,rep,name=conditions"`

	/* Current of the Pod that has been launched. */
	//+kubebuilder:default=""
	//+kubebuilder:validation:Optional
	State *string `json:"state,omitempty" protobuf:"bytes,2,opt,name=state"`

	/* Name of the Pod that has been launched. */
	//+kubebuilder:validation:MinLength=1
	//+kubebuilder:validation:Optional
	PodName *string `json:"podName,omitempty" protobuf:"bytes,3,opt,name=podName"`

	/* Result backup file size. */
	//+kubebuilder:validation:Optional
	Size *string `json:"size,omitempty" protobuf:"bytes,4,opt,name=size"`

	/* Same as size, but in bytes. */
	//+kubebuilder:default=0
	//+kubebuilder:validation:Minimum=0
	//+kubebuilder:validation:Optional
	SizeInBytes *uint `json:"sizeInBytes,omitempty" protobuf:"varint,5,opt,name=sizeInBytes"`
}

/*
Actual backup instance. It creates Pod according to specified spec and run the backup command.
Backup is streamed to the BackupStorage through compression\encryption processor if any.
*/
//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:shortName=bacr;backr;backuprun
//+kubebuilder:printcolumn:name="State",type=string,JSONPath=`.status.state`,description="State"
//+kubebuilder:printcolumn:name="Restorable",type=string,JSONPath=`.status.conditions[?(@.type=="Restorable")].status`,description="Can be restored automatically",priority=1
//+kubebuilder:printcolumn:name="Encrypted",type=string,JSONPath=`.status.conditions[?(@.type=="Encrypted")].status`,description="Encryption status",priority=1
//+kubebuilder:printcolumn:name="Compressed",type=string,JSONPath=`.status.conditions[?(@.type=="Compressed")].status`,description="Compression status",priority=1
//+kubebuilder:printcolumn:name="Path",type=string,JSONPath=`.spec.storage.path`,description="Path to file in BackupStorage",priority=1
//+kubebuilder:printcolumn:name="Size",type=string,JSONPath=`.status.size`,description="Backup file size",priority=1
//+kubebuilder:printcolumn:name="Age",type=date,format=date-time,JSONPath=`.metadata.creationTimestamp`,description="Creation timestamp"

type BackupRun struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,3,req,name=metadata"`

	Spec   BackupRunSpec   `json:"spec,omitempty" protobuf:"bytes,4,req,name=metadata"`
	Status BackupRunStatus `json:"status,omitempty" protobuf:"bytes,5,opt,name=metadata"`
}

/* BackupRunList contains a list of BackupRun. */
//+kubebuilder:object:root=true
type BackupRunList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty" protobuf:"bytes,3,opt,name=metadata"`
	Items           []BackupRun `json:"items" protobuf:"bytes,4,req,name=items"`
}

func init() {
	SchemeBuilder.Register(&BackupRun{}, &BackupRunList{})
}
