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

	"golang.org/x/sync/errgroup"
	"sigs.k8s.io/controller-runtime/pkg/client"

	backupoperatoriov1 "backup-operator.io/api/v1"
	"backup-operator.io/internal/controller/backupRun/compression"
	"backup-operator.io/internal/controller/backupRun/encryption"
	backupstorage "backup-operator.io/internal/controller/backupStorage"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"k8s.io/utils/ptr"
)

// Make a backup run
func Backup(ctx context.Context, c client.Client, s *runtime.Scheme,
	config *rest.Config, run *backupoperatoriov1.BackupRun,
	pod *corev1.Pod, storage backupstorage.BackupStorageProvider) (err error) {

	state := AnalyzeRunConditions(run)
	action := run.Spec.Backup

	// Result pipe
	resultReader, resultWriter := io.Pipe()
	defer resultReader.Close()
	// Future stdout stream
	var stdout io.WriteCloser
	var encryptionWriter io.WriteCloser
	// This will be passed to pod exec
	exec := &podExecParameters{
		Stdin:   nil, // Backup does not have stdin
		Stdout:  nil, // Stdout will be set below
		Stderr:  os.Stdout,
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
	// Start stream to storage file
	storageRoutineEgr, storageRoutineEgrCtx := errgroup.WithContext(context.WithoutCancel(ctx))
	storageRoutineEgr.Go(func() (err error) {
		return storage.Put(storageRoutineEgrCtx, run.Spec.Storage.Path, resultReader)
	})
	// We have 4 possible schemes
	switch {
	case !state.Encrypted && !state.Compressed:
		// exec -> storage -> result
		stdout = resultWriter
		// Will be closed by default
	case !state.Encrypted && state.Compressed:
		// exec -> compression -> result -> storage
		if stdout, err = compressor.Compress(resultWriter, int(run.Spec.Compression.Level)); err != nil {
			err = fmt.Errorf("failed to create compressor writer: %s", err.Error())
			return
		}
	case state.Encrypted && !state.Compressed:
		// exec -> encryption -> result -> storage
		if stdout, err = encryptor.Encrypt(resultWriter, run.Spec.Encryption.Recipients...); err != nil {
			err = fmt.Errorf("failed to create encryptor writer: %s", err.Error())
			return
		}
	case state.Encrypted && state.Compressed:
		// exec -> compression -> encryption -> result -> storage
		if encryptionWriter, err = encryptor.Encrypt(resultWriter, run.Spec.Encryption.Recipients...); err != nil {
			err = fmt.Errorf("failed to create encryptor writer: %s", err.Error())
			return
		}
		if stdout, err = compressor.Compress(encryptionWriter, int(run.Spec.Compression.Level)); err != nil {
			err = fmt.Errorf("failed to create compressor writer: %s", err.Error())
			return
		}
	}
	defer stdout.Close()
	// Make Pod exec
	exec.Stdout = stdout
	if err = podExec(ctx, config, pod, exec); err != nil {
		return
	}
	// Close streams
	stdout.Close()
	if state.Encrypted || state.Compressed {
		if state.Encrypted && state.Compressed {
			encryptionWriter.Close()
		}
		resultWriter.Close()
	}
	// Check whether any of the egr goroutines failed. Since egr is accumulating...
	// ...the errors, we don't need to send them (or check for them) in...
	// ...the individual results sent on the channel.
	if err = storageRoutineEgr.Wait(); err != nil {
		err = fmt.Errorf("failed storage put routine: %s", err.Error())
		return
	}
	return
}
