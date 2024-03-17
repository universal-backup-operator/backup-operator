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

package encryption

import (
	"fmt"
	"io"

	"backup-operator.io/internal/controller/backupRun/wrappers"
	"filippo.io/age"
)

type Encryption interface {
	Encrypt(out io.Writer, keys ...string) (encrypted io.WriteCloser, err error)
	Decrypt(in io.Reader, keys ...string) (plain io.ReadCloser, err error)
}

type AgeEncryption struct{}

func (a *AgeEncryption) Encrypt(out io.Writer, keys ...string) (encrypted io.WriteCloser, err error) {
	var recipients []age.Recipient
	// Parse recipients
	for _, key := range keys {
		var recipient age.Recipient
		if recipient, err = age.ParseX25519Recipient(key); err != nil {
			return nil, fmt.Errorf("failed to parse X25519Recipient: %s", key)
		}
		recipients = append(recipients, recipient)
	}
	// Create encrypt writer
	if encrypted, err = age.Encrypt(out, recipients...); err != nil && err != io.ErrClosedPipe {
		return nil, fmt.Errorf("failed to create encrypted writer: %s", err.Error())
	}
	return encrypted, err
}

func (a *AgeEncryption) Decrypt(in io.Reader, keys ...string) (plain io.ReadCloser, err error) {
	var identities []age.Identity
	// Parse identities
	for _, key := range keys {
		var identity age.Identity
		if identity, err = age.ParseX25519Identity(key); err != nil {
			return nil, fmt.Errorf("failed to parse X25519Recipient: %s", key)
		}
		identities = append(identities, identity)
	}
	// Create encrypt reader
	var ar io.Reader
	if ar, err = age.Decrypt(in, identities...); err != nil && err != io.ErrClosedPipe {
		return nil, fmt.Errorf("failed to create encrypted writer: %s", err.Error())
	}
	return &wrappers.ReaderWrapper{Reader: ar}, err
}
