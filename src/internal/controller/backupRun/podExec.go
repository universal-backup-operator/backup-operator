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
	"strings"
	"time"

	"backup-operator.io/internal/controller/backupRun/wrappers"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
)

type podExecParameters struct {
	// Exec STDIN
	Stdin io.Reader
	// Exec STDOUT
	Stdout io.Writer
	// Exec STDERR
	Stderr io.Writer
	// Command to run
	Command []string
	// Timeout
	Timeout *time.Duration
	// If true, will create dummy reader/writer for stdin/stdout/stderr...
	// ...in case if particular stream is nil
	CreateStubChannels bool
	// Either TTY is to be allocated
	TTY bool
}

// Makes Pod exec
func podExec(ctx context.Context, config *rest.Config, pod *corev1.Pod, pp *podExecParameters) (err error) {
	// Prepare connection clientset
	var clientset *kubernetes.Clientset
	if clientset, err = kubernetes.NewForConfig(config); err != nil {
		return
	}
	// Prepare the API URL used to execute another process within the Pod. In
	// this case, we'll run a remote shell.
	req := clientset.CoreV1().RESTClient().
		Post().
		Namespace(pod.Namespace).
		Resource("pods").
		Name(pod.Name).
		SubResource("exec").
		VersionedParams(&corev1.PodExecOptions{
			Container: pod.Spec.Containers[0].Name,
			Command:   pp.Command,
			Stdin:     pp.CreateStubChannels || pp.Stdin != nil,
			Stdout:    pp.CreateStubChannels || pp.Stdout != nil,
			Stderr:    pp.CreateStubChannels || pp.Stderr != nil,
			TTY:       pp.TTY,
		}, scheme.ParameterCodec)
	var exec remotecommand.Executor
	if exec, err = remotecommand.NewSPDYExecutor(config, "POST", req.URL()); err != nil {
		err = fmt.Errorf("failed to create remotecommand.Executor: %s", err.Error())
		return
	}
	// Prepare stream options
	opt := remotecommand.StreamOptions{
		Stdin:  pp.Stdin,
		Stdout: pp.Stdout,
		Stderr: pp.Stderr,
		Tty:    pp.TTY,
	}
	// Create dummy channels
	if pp.CreateStubChannels {
		dummyReader := wrappers.ReaderWrapper{Reader: strings.NewReader("")}
		dummyWriter := wrappers.WriterWrapper{Writer: io.Discard}
		if opt.Stdin == nil {
			opt.Stdin = dummyReader
		}
		if opt.Stdout == nil {
			opt.Stdout = dummyWriter
		}
		if opt.Stderr == nil {
			opt.Stderr = dummyWriter
		}
	}
	// Create timeout context
	var streamCtx context.Context
	var streamCtxCancel context.CancelFunc
	if pp.Timeout != nil {
		streamCtx, streamCtxCancel = context.WithTimeout(ctx, *pp.Timeout)
		defer streamCtxCancel()
	} else {
		streamCtx = ctx
	}
	// Make pod exec
	if err = exec.StreamWithContext(streamCtx, opt); err != nil {
		err = fmt.Errorf("pod exec failed: %s", err.Error())
		return
	}
	return
}
