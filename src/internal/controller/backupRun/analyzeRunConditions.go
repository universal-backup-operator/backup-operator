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

package backuprun

import (
	"sync"

	backupoperatoriov1 "backup-operator.io/api/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type BackupRunState struct {
	// True if compression is enabled
	Compressed bool
	// True if restoration is available
	Restorable bool
	// True if encryption is enabled
	Encrypted bool
	// True if run has been interrupted
	Interrupted bool
	// True if run has finished
	Completed bool
	// True if run has never happened
	NeverRun bool
	// True if inProgress
	InProgress bool
	// True if completed and failed
	Failed bool
	// True if completed and successful
	Successful bool
	// True if we have to make a backup
	HaveToBackup bool
	// True if we have to make a restoration
	HaveToRestore bool
}

var inProgressRuns = &sync.Map{}

// Analyze BackupRun conditions in one place
func AnalyzeRunConditions(run *backupoperatoriov1.BackupRun) (s *BackupRunState) {
	s = &BackupRunState{}
	// Either we have are in progress according to conditions
	for _, c := range run.Status.Conditions {
		switch c.Type {
		case string(backupoperatoriov1.BackupRunConditionTypeInProgress):
			s.InProgress = c.Status == metav1.ConditionTrue
		case string(backupoperatoriov1.BackupRunConditionTypeFailed):
			s.Failed = c.Status == metav1.ConditionTrue
		case string(backupoperatoriov1.BackupRunConditionTypeSuccessful):
			s.Successful = c.Status == metav1.ConditionTrue
		}
	}
	// Manage in progress runs
	switch s.InProgress {
	case true:
		if _, known := inProgressRuns.LoadOrStore(run.UID, true); !known {
			// If run is in progress and it is absent in the map - controller has been restarted...
			// ...and run will hang forever
			s.Interrupted = true
		}
	case false:
		inProgressRuns.Delete(run.UID)
	}
	// Check either runs is successful or failed and not in progress
	s.Completed = !s.InProgress && (s.Successful || s.Failed)
	// Either we never run and .status.mode is empty at all
	s.NeverRun = !(s.InProgress || s.Successful || s.Failed)
	// Checking annotation
	_, restoreAnnotationExists := run.GetAnnotations()[backupoperatoriov1.AnnotationRestore]
	backupIsDefined := run.Spec.Backup != nil
	restoreIsDefined := run.Spec.Restore != nil
	restoreOnlyMode := !backupIsDefined && restoreIsDefined
	// Determine if backup is necessary based on the following conditions:
	//   - If nothing has been completed,
	//   - If a backup block is defined, and
	//   - If it is the first run.
	s.HaveToBackup = !s.Completed &&
		backupIsDefined &&
		s.NeverRun
	// Determine if restoration is necessary based on the following conditions:
	//   - If in restore-only mode and the backup is not completed or it was, but repeat is requested
	//   - If not in restore-only mode and the restore block is defined, and a restore is requested, and backup is completed
	s.HaveToRestore = (restoreOnlyMode && (!s.Completed || restoreAnnotationExists)) ||
		(restoreIsDefined && s.Completed && restoreAnnotationExists)
	// Check encryption
	s.Encrypted = run.Spec.Encryption != nil
	// Check compression
	s.Compressed = run.Spec.Compression != nil
	// Check restoration
	s.Restorable = run.Spec.Restore != nil
	if s.Encrypted {
		s.Restorable = s.Restorable && run.Spec.Encryption.DecryptionKey != nil
	}
	return
}
