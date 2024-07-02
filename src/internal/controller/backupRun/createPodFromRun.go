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

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/util/retry"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	backupoperatoriov1 "backup-operator.io/api/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

func CreatePodFromRun(ctx context.Context, c client.Client, s *runtime.Scheme,
	config *rest.Config, run *backupoperatoriov1.BackupRun, pod *corev1.Pod) (err error) {
	// Delete run Pod if it exists
	if err = c.Delete(ctx, pod.DeepCopy(), &client.DeleteOptions{
		GracePeriodSeconds: ptr.To[int64](0),
		PropagationPolicy:  ptr.To(metav1.DeletePropagationBackground),
	}); client.IgnoreNotFound(err) != nil {
		return
	}
	// Set custom labels and/or annotations if any
	if run.Spec.Template.Metadata != nil {
		pod.SetLabels(run.Spec.Template.Metadata.Labels)
		pod.SetAnnotations(run.Spec.Template.Metadata.Annotations)
	}
	// Set ownership
	if err = ctrl.SetControllerReference(run, pod, s); err != nil {
		return
	}
	// Create the pod
	if err = c.Create(ctx, pod); err != nil {
		return
	}
	// Set status.podName
	if err = retry.RetryOnConflict(retry.DefaultRetry, func() (err error) {
		if err = c.Get(ctx, client.ObjectKeyFromObject(run), run); err != nil {
			return
		}
		run.Status.PodName = ptr.To[string](pod.Name)
		return c.Status().Update(ctx, run)
	}); err != nil {
		return err
	}
	// Prepare connection clientset
	var clientset *kubernetes.Clientset
	if clientset, err = kubernetes.NewForConfig(config); err != nil {
		return
	}
	// Watch for Pod changes...
	var watcher watch.Interface
	if watcher, err = clientset.CoreV1().Pods(pod.Namespace).
		Watch(ctx, metav1.SingleObject(pod.ObjectMeta)); err != nil {
		return
	}
	var internalErrors uint
	// ...parsing  event channel values...
	for event := range watcher.ResultChan() {
		switch event.Type {
		case watch.Modified:
			// ...till it is modified...
			pod = event.Object.(*corev1.Pod)
			for _, cond := range pod.Status.Conditions {
				// ...and ready. Now we can
				if cond.Type == corev1.PodReady &&
					cond.Status == corev1.ConditionTrue {
					// ...stop watcher and continue the run
					watcher.Stop()
				}
			}
		default:
			if status, ok := event.Object.(*metav1.Status); !ok {
				err = fmt.Errorf("failed to convert watcherEvent to metav1.Status: %+v", event)
				return
			} else {
				switch status.Reason {
				case metav1.StatusReasonInternalError:
					internalErrors++
					if internalErrors < 3 {
						continue
					}
				}
			}
			// We will be there in case of ERROR or DELETED event
			// Or we have received more than 3 internalErrors
			watcher.Stop()
			err = fmt.Errorf("pod is in the wrong state: %s", string(event.Type))
			return
		}
	}
	return
}
