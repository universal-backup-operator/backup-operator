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

package backupstorage

import (
	"context"
	"io"
	"sync"

	backupoperatoriov1 "backup-operator.io/api/v1"
)

// BackupStorageProvider defines the interface for different storage providers like S3, NFS, etc.
type BackupStorageProvider interface {
	// Get underlying Kubernetes object
	GetObject() *backupoperatoriov1.BackupStorage
	// Read parameters from backup storage and configure provider.
	Constructor(object *backupoperatoriov1.BackupStorage, parameters map[string]string, credentials map[string]string) error
	// Actions to make before object destruction.
	Destructor() error
	// Upload file.
	Put(ctx context.Context, path string, reader io.Reader) error
	// Download file.
	Get(ctx context.Context, path string) (io.ReadCloser, error)
	// List path.
	List(ctx context.Context, path string) ([]string, error)
	// Remove path.
	Delete(ctx context.Context, path string) error
	// Get file size in bytes
	GetSize(ctx context.Context, path string) (uint, error)
}

// All initialized backup storage providers objects
var backupStorageProviders sync.Map

// GetBackupStorageProvider Get backup storage provider by name
func GetBackupStorageProvider(name string) (storage BackupStorageProvider, ok bool) {
	var value any
	if value, ok = backupStorageProviders.Load(name); ok {
		storage, ok = value.(BackupStorageProvider)
	}
	return
}

// AddBackupStorageProvider Add backup storage provider by name
func AddBackupStorageProvider(name string, storage BackupStorageProvider) {
	backupStorageProviders.Store(name, storage)
}

// RemoveBackupStorageProvider Remove backup storage provider by name
func RemoveBackupStorageProvider(name string) (ok bool) {
	if _, ok = backupStorageProviders.Load(name); ok {
		// Remove if exists...
		backupStorageProviders.Delete(name)
	}
	// ...and do not raise any error in case if it is absent
	return
}
