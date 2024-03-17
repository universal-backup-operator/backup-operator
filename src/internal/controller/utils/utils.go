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
	"bytes"
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"html/template"
	"io"
	"time"

	"github.com/Masterminds/sprig/v3"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Common event logging reasons
const (
	EventReasonReady        string = "Ready"
	EventReasonCompleted    string = "Completed"
	EventReasonNotReady     string = "Unhealthy"
	EventReasonCreated      string = "Created"
	EventReasonUpdated      string = "Updated"
	EventReasonDeleted      string = "Deleted"
	EventReasonFailed       string = "Failed"
	EventReasonInvalid      string = "Invalid"
	EventReasonTimeout      string = "Timeout"
	EventReasonPending      string = "Pending"
	EventReasonUnknown      string = "Unknown"
	EventReasonReconciled   string = "Reconciled"
	EventReasonInitializing string = "Initializing"
	EventReasonFinalizing   string = "Finalizing"
	EventReasonAccessDenied string = "AccessDenied"
)

// DecodeSecretData decodes the content of a Kubernetes Secret object into a map of strings.
// It takes a pointer to a corev1.Secret object and returns a map[string]string containing the decoded data.
func DecodeSecretData(secret *corev1.Secret) map[string]string {
	data := make(map[string]string)
	for key, value := range secret.Data {
		data[key] = string(value)
	}
	return data
}

// TextTemplateSprig renders the provided text template using Sprig functions and returns the rendered string.
// It takes the text template 't' and a value 'v' of any type for template rendering.
//
// Usage:
//
//	renderedString, err := TextTemplateSprig("Your text template with {{ .Variable }}", valuesStruct)
//
// Parameters:
//
//	t - The text template to render.
//	v - The value to pass for template rendering.
//
// Returns:
//
//	string - The rendered string if successful.
//	error - An error if template parsing or rendering fails.
func TextTemplateSprig(t string, v any) (string, error) {
	// Template backupPath
	var tmpl *template.Template
	var err error
	buffer := bytes.Buffer{}
	// Parse
	if tmpl, err = template.New("backupPath").Funcs(sprig.FuncMap()).Parse(t); err != nil {
		return "", fmt.Errorf("failed to parse template: %s", err.Error())
	}
	// Render
	if err = tmpl.Execute(&buffer, v); err != nil {
		return "", fmt.Errorf("failed to render template: %s", err.Error())
	}
	return buffer.String(), nil
}

// Creates time.Ticker object for nextTime (can be calculated from cronexpr)
func CreateTimeTicker(nextTime time.Time) *time.Ticker {
	// Get the current time
	currentTime := time.Now()
	// Calculate the duration until the next scheduled run
	durationUntilNextRun := nextTime.Sub(currentTime)
	// Create a ticker to perform actions at scheduled intervals
	return time.NewTicker(durationUntilNextRun)
}

// Thanks to https://gist.github.com/yakuter/6bf1e565311d11251febda4a04a6bc64
type PipeCopyFunc func(buffer []byte) (n int, err error)

func (f PipeCopyFunc) Read(buffer []byte) (n int, err error) { return f(buffer) }

// Function for streaming data from writer to reader with context honoring
var PipeCopy = func(ctx context.Context, r io.Reader, w io.Writer) (err error) {
	reader := PipeCopyFunc(func(buffer []byte) (int, error) {
		// golang non-blocking channel: https://gobyexample.com/non-blocking-channel-operations
		select {
		// if context has been canceled
		case <-ctx.Done():
			// stop process and propagate "context canceled" error
			return 0, ctx.Err()
		default:
			// otherwise just run default io.Reader implementation
			return r.Read(buffer)
		}
	})

	if _, err = io.Copy(w, reader); err != nil && err != io.ErrClosedPipe {
		return err
	}
	return nil
}

// Set annotations for an object
func SetAnnotations(ctx context.Context, c client.Client, o client.Object, a map[string]string) (err error) {
	// Prepare and update the status...
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		if err = c.Get(ctx, client.ObjectKeyFromObject(o), o); err != nil {
			return err
		}
		o.SetAnnotations(a)
		return c.Update(ctx, o)
	})
}

// Calculate hash from objects that can be formatted by fmt.Sprint
func Hash(objects ...interface{}) (result string) {
	// nosemgrep: go.lang.security.audit.crypto.use_of_weak_crypto.use-of-md5
	h := md5.New()
	for _, o := range objects {
		h.Write([]byte(fmt.Sprint(o)))
	}
	return hex.EncodeToString(h.Sum(nil))
}
