// Copyright 2025 Patryk Rostkowski
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package controlplane

import (
	"context"

	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type BaseReconciler struct {
	client.Client
	Log      logr.Logger
	Recorder record.EventRecorder
	Scheme   *runtime.Scheme
}

func (r *BaseReconciler) GetOrIgnoreNotFound(
	ctx context.Context,
	key client.ObjectKey,
	obj client.Object,
) error {
	if err := r.Get(ctx, key, obj); err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}
		return err
	}
	return nil
}

func (r *BaseReconciler) IsDeploymentReady(dep *appsv1.Deployment) bool {
	desired := *dep.Spec.Replicas
	if dep.Status.ReadyReplicas < desired {
		return false
	}
	if dep.Generation > dep.Status.ObservedGeneration {
		return false
	}
	if dep.Status.UpdatedReplicas < desired {
		return false
	}
	if dep.Status.ReadyReplicas < desired || dep.Status.AvailableReplicas < desired {
		return false
	}
	return true
}

type ObjectHelper interface {
	client.Object
	GetConditions() *[]metav1.Condition
	GetStatus() *Status
}
