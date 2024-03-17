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
	"fmt"
	"io"
	"os"
	"time"

	"sigs.k8s.io/controller-runtime/pkg/client"

	backupoperatoriov1 "backup-operator.io/api/v1"
	"backup-operator.io/internal/controller/backupRun/compression"
	"backup-operator.io/internal/controller/backupRun/encryption"
	backupstorage "backup-operator.io/internal/controller/backupStorage"
	"backup-operator.io/internal/controller/utils"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"k8s.io/utils/ptr"
)

// Make a restore run
func Restore(ctx context.Context, c client.Client, s *runtime.Scheme,
	config *rest.Config, run *backupoperatoriov1.BackupRun,
	pod *corev1.Pod, storage backupstorage.BackupStorageProvider) (err error) {

	state := AnalyzeRunConditions(run)
	action := run.Spec.Restore

	// Check that backup is restorable
	if !state.Restorable {
		err = fmt.Errorf("backup is not restorable, but restore has been requested")
		return
	}
	// Create a reader from storage
	var backupReader io.ReadCloser
	// Open reader to storage
	if backupReader, err = storage.Get(ctx, run.Spec.Storage.Path); err != nil {
		err = fmt.Errorf("failed to open reader to storage backup: %s", err.Error())
		return
	}
	defer backupReader.Close()
	// Get decryption key
	var decryptionKey string
	if state.Encrypted {
		secret := &corev1.Secret{}
		secret.Name = run.Spec.Encryption.DecryptionKey.Name
		if run.Spec.Encryption.DecryptionKey.Namespace == nil {
			secret.Namespace = run.Namespace
		} else {
			secret.Namespace = *run.Spec.Encryption.DecryptionKey.Namespace
		}
		if err = c.Get(ctx, client.ObjectKeyFromObject(secret), secret); err != nil {
			// Fail if could not read the secret...
			err = fmt.Errorf("failed to fetch secret %s/%s with decryption key: %s",
				secret.Namespace, secret.Name, err.Error())
			return
		}
		var ok bool
		if decryptionKey, ok = utils.DecodeSecretData(secret)[run.Spec.Encryption.DecryptionKey.Key]; !ok {
			// ...or it does not have requested key
			err = fmt.Errorf("secret %s/%s does not have key %s",
				secret.Namespace, secret.Name, run.Spec.Encryption.DecryptionKey.Key)
			return
		}
	}
	// Future stdin stream
	var stdin io.ReadCloser
	// This will be passed to pod exec
	exec := &podExecParameters{
		Stdin:   nil, // Stdin will be set below
		Stdout:  nil, // Restore does not have stdout
		Stderr:  os.Stderr,
		Command: append(action.Command, action.Args...),
	}
	if action.DeadlineSeconds != nil {
		exec.Timeout = ptr.To[time.Duration](time.Second * time.Duration(*action.DeadlineSeconds))
	}
	// Create compressor and encryptor
	var compressor compression.Compression
	var encryptor encryption.Encryption
	if encryptor, compressor, err = getEncryptorAndCompressor(run); err != nil {
		return
	}
	// We have 4 possible schemes
	switch {
	case !state.Encrypted && !state.Compressed:
		// storage -> exec
		stdin = backupReader
		// Will be closed by default
	case !state.Encrypted && state.Compressed:
		// storage -> decompression -> exec
		if stdin, err = compressor.Decompress(backupReader); err != nil {
			err = fmt.Errorf("failed to create compressor reader: %s", err.Error())
			return
		}
	case state.Encrypted && !state.Compressed:
		// storage -> decryption -> exec
		if stdin, err = encryptor.Decrypt(backupReader, decryptionKey); err != nil {
			err = fmt.Errorf("failed to create encryptor reader: %s", err.Error())
			return
		}
	case state.Encrypted && state.Compressed:
		// storage -> decryption -> decompression -> exec
		var compressionReader io.ReadCloser
		if compressionReader, err = encryptor.Decrypt(backupReader, decryptionKey); err != nil {
			err = fmt.Errorf("failed to create encryptor reader: %s", err.Error())
			return
		}
		defer compressionReader.Close()
		if stdin, err = compressor.Decompress(compressionReader); err != nil {
			err = fmt.Errorf("failed to create compressor reader: %s", err.Error())
			return
		}
	}
	defer stdin.Close()
	// Make Pod exec
	exec.Stdin = stdin
	err = podExec(ctx, config, pod, exec)
	return
}
