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
	"context"

	"k8s.io/client-go/util/retry"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	backupoperatoriov1 "backup-operator.io/api/v1"
	backupstorage "backup-operator.io/internal/controller/backupStorage"
	"backup-operator.io/internal/controller/utils"
)

// SetBackupSizeInStatus updates backup size status field
func SetBackupSizeInStatus(ctx context.Context, c client.Client,
	run *backupoperatoriov1.BackupRun, storage backupstorage.BackupStorageProvider,
) (err error) {
	return retry.RetryOnConflict(retry.DefaultRetry, func() (err error) {
		if err = c.Get(ctx, client.ObjectKeyFromObject(run), run); err != nil {
			return err
		}
		var bytes uint
		if bytes, err = storage.GetSize(ctx, run.Spec.Storage.Path); err != nil {
			return
		}
		run.Status.SizeInBytes = ptr.To(bytes)
		run.Status.Size = ptr.To(utils.ConvertBytesToHumanReadable(bytes))
		return c.Status().Update(ctx, run)
	})
}
