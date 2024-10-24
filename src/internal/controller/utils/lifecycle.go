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
	"context"
	"sync"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

// Target to reconcile
type ManagedLifecycleReconcile struct {
	client.Client
	// REST config for advanced operations
	Config *rest.Config
	// Reconciler scheme
	Scheme *runtime.Scheme
	// Object Event recorder
	Recorder record.EventRecorder
	// Reconciler control request
	Request ctrl.Request
	// Reconciler object
	Object client.Object
}

// Constructor and destructor to call for Kubernetes object after creation and before deletion
type ManagedLifecycleObject interface {
	// Initialize the object on first creation
	Constructor(ctx context.Context, m *ManagedLifecycleReconcile) (ctrl.Result, error)
	// Make some work with object
	Processor(ctx context.Context, m *ManagedLifecycleReconcile) (ctrl.Result, error)
	// Clean before object deletion
	Destructor(ctx context.Context, m *ManagedLifecycleReconcile) (ctrl.Result, error)
}

// List of processed objects resourceVersions
// Key is object UID
// Value is resourceVersion.(int)
var processed = &sync.Map{}

// ManageLifecycle manages the lifecycle of a Kubernetes object, including creation and deletion, while handling finalizers.
//
// Parameters:
//
//	ctx: Context carries deadlines, cancellation signals, and other request-scoped values.
//	r: ManagedLifecycleReconcile containing necessary components for reconciliation.
//	m: The ManagedLifecycleObject interface with constructor and destructor methods to invoke.
//
// Returns an error encountered during the lifecycle management operations. On success, returns nil.
//
// Note:
//
//	This function performs standard CRUD (Create, Read, Update, Delete) operations on the provided object
//	within the Kubernetes cluster, managing its lifecycle accordingly.
//	Ensure proper authentication, permission settings, and configuration for the client and runtime scheme
//	before calling this function.
//
// Example:
//
//	processedResourcesMap := &sync.Map{}
//	err := ManageLifecycle(ctx, &ManagedLifecycleReconcile{
//	    Client:   r.Client,
//	    Scheme:   r.Scheme,
//	    Recorder: r.Recorder,
//	    Request:  req,
//	    Object:   &MyCustomObject{},
//	}, &MyCustomObjectLifecycle{})
//	if err != nil {
//	    // Handle error
//	}
func ManageLifecycle(ctx context.Context, r *ManagedLifecycleReconcile, m ManagedLifecycleObject) (result ctrl.Result, err error) {
	log := log.FromContext(ctx)
	if err = r.Get(ctx, r.Request.NamespacedName, r.Object); err != nil {
		// DELETED or ERROR
		// Handle error except of absent object
		if !errors.IsNotFound(err) {
			Log(r, log, err, r.Object, "FailedGet", "could not get an object")
		}
		return result, client.IgnoreNotFound(err)
	}
	// Prepare finalizer name
	finalizer := r.Object.GetObjectKind().GroupVersionKind().GroupVersion().String()
	// Check whether we have processed this resource already, so we can skip constructor
	_, isConstructed := processed.LoadOrStore(r.Object.GetUID(), r.Object.GetResourceVersion())
	// Examine DeletionTimestamp to determine if object is under deletion
	if !r.Object.GetDeletionTimestamp().IsZero() {
		// DELETING
		// The object is being deleted
		Log(r, log, err, r.Object, "Reconciliation", "deleting the object")
		if result, err = m.Destructor(ctx, r); err != nil {
			Log(r, log, err, r.Object, "FailedDeletion", "failed to delete the object")
			return
		}
		// The object is to be rescheduled, but no error
		if result.RequeueAfter != 0 {
			return
		}
		// Removing finalizer
		if controllerutil.ContainsFinalizer(r.Object, finalizer) {
			// Remove our finalizer
			if err = retry.RetryOnConflict(retry.DefaultRetry, func() (err error) {
				if err = r.Client.Get(ctx, client.ObjectKeyFromObject(r.Object), r.Object); err != nil {
					return
				}
				controllerutil.RemoveFinalizer(r.Object, finalizer)
				return r.Client.Update(ctx, r.Object)
			}); client.IgnoreNotFound(err) != nil {
				Log(r, log, err, r.Object, "FailedRemoveFinalizer", "failed to remove a finalizer")
				return
			}
		}
		err = client.IgnoreNotFound(err)
		Log(r, log, err, r.Object, "DeletionCompleted", "deletion has been completed successfully")
		// Remove from constructed
		processed.Delete(r.Object.GetUID())
		return
	}
	// CREATED or UPDATED
	// Registering finalizer
	if !controllerutil.ContainsFinalizer(r.Object, finalizer) {
		// Add operator finalizer
		if err = retry.RetryOnConflict(retry.DefaultRetry, func() (err error) {
			if err = r.Client.Get(ctx, client.ObjectKeyFromObject(r.Object), r.Object); err != nil {
				return
			}
			controllerutil.AddFinalizer(r.Object, finalizer)
			return r.Client.Update(ctx, r.Object)
		}); err != nil {
			r.Recorder.Eventf(r.Object, corev1.EventTypeWarning, EventReasonFailed, "%s:", err.Error())
			return
		}
	}
	// Run Constructor callback
	if !isConstructed {
		if result, err = m.Constructor(ctx, r); err != nil {
			r.Recorder.Eventf(r.Object, corev1.EventTypeWarning, EventReasonFailed, "%s", err.Error())
			return
		}
	}
	// Run Processor callback
	if result, err = m.Processor(ctx, r); err != nil {
		r.Recorder.Eventf(r.Object, corev1.EventTypeWarning, EventReasonFailed, "%s", err.Error())
		return
	}
	return
}
