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
	"fmt"

	backupoperatoriov1 "backup-operator.io/api/v1"
	"backup-operator.io/internal/controller/backupRun/compression"
	"backup-operator.io/internal/controller/backupRun/encryption"
)

// Prepare compression and encryption objects
func getEncryptorAndCompressor(run *backupoperatoriov1.BackupRun) (
	e encryption.Encryption, c compression.Compression, err error) {

	state := AnalyzeRunConditions(run)
	if state.Encrypted {
		// Create standard encryptor...
		e = &encryption.AgeEncryption{}
	}
	if state.Compressed {
		switch run.Spec.Compression.Algorithm {
		// ...and compressor depending on the algorithm
		case backupoperatoriov1.GZIP:
			c = &compression.GZIPCompression{}
		default:
			err = fmt.Errorf("unknown compression algorithm: %s", run.Spec.Compression.Algorithm)
		}
	}
	return
}
