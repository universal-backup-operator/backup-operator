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

package utils

import (
	"fmt"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

/*
Log is a utility function designed to abstract the logging and event recording logic in a Kubernetes-like reconciler system.
It accepts a pointer to a ManagedLifecycleReconcile instance, a logger for structured logging, an optional error,
the object being reconciled, a reason for logging, and a message string.

The function handles different scenarios based on whether the provided error is nil or not:
  - If err is nil, it logs at level V(1) with an event of type Normal using the logger's recorder.
    The log entry includes the specified reason and detailed message without additional details about the error.
  - If err is not nil, it logs at level V(1) with an error and constructs a warning event using the logger's recorder.
    The log entry includes the specified reason and a detailed message including the error's string representation.
*/
func Log(m *ManagedLifecycleReconcile, l logr.Logger, err error, o runtime.Object,
	reason string, message string,
) {
	switch err {
	case nil:
		l.V(1).Info(message)
		m.Recorder.Eventf(o, corev1.EventTypeNormal, reason, message)
	default:
		l.Error(err, reason)
		msg := err.Error()
		if len(message) > 0 {
			msg = fmt.Sprintf("%s: %s", message, err.Error())
		}
		m.Recorder.Eventf(o, corev1.EventTypeWarning, reason, msg)
	}
}
