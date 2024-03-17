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

func (in *BackupRunAction) DeepCopy() *BackupRunAction {
	if in == nil {
		return nil
	}
	out := new(BackupRunAction)
	in.DeepCopyInto(out)
	return out
}

func (in *BackupRunAction) DeepCopyInto(out *BackupRunAction) {
	*out = *in
}

func (in *backupCompression) DeepCopy() *backupCompression {
	if in == nil {
		return nil
	}
	out := new(backupCompression)
	in.DeepCopyInto(out)
	return out
}

func (in *backupCompression) DeepCopyInto(out *backupCompression) {
	*out = *in
}

func (in *backupEncryption) DeepCopy() *backupEncryption {
	if in == nil {
		return nil
	}
	out := new(backupEncryption)
	in.DeepCopyInto(out)
	return out
}

func (in *backupEncryption) DeepCopyInto(out *backupEncryption) {
	*out = *in
}

func (in *pod) DeepCopy() *pod {
	if in == nil {
		return nil
	}
	out := new(pod)
	in.DeepCopyInto(out)
	return out
}

func (in *pod) DeepCopyInto(out *pod) {
	*out = *in
}
