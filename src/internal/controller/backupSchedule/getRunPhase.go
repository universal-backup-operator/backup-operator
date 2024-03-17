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

package backupschedule

import (
	backupoperatoriov1 "backup-operator.io/api/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

// Function to check job completion and status
func GetRunPhase(run backupoperatoriov1.BackupRun) (notInitialized bool, finished bool, t *backupoperatoriov1.BackupRunConditionType) {
	// If run is recently it may be not processed by operator yet...
	if len(run.Status.Conditions) == 0 {
		// ...we can understand that if run has no conditions
		return true, false, nil
	}
	notInitialized = false
	for _, c := range run.Status.Conditions {
		// We are interested only in conditions that are in True status...
		if c.Status == metav1.ConditionTrue {
			// ...saving pointer to type...
			t = ptr.To[backupoperatoriov1.BackupRunConditionType](backupoperatoriov1.BackupRunConditionType(c.Type))
			switch c.Type {
			case string(backupoperatoriov1.BackupRunConditionTypeFailed),
				string(backupoperatoriov1.BackupRunConditionTypeSuccessful):
				// ...and finish if case of successful/failed...
				finished = true
				return
			case string(backupoperatoriov1.BackupRunConditionTypeInProgress):
				// ...or inProgress state.
				finished = false
				return
			}
		}
	}
	// If nothing has been found - consider initializing
	return true, false, nil
}
