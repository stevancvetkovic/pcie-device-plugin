/*
Copyright 2024.

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

package controller

import (
	"context"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"

	corev1 "k8s.io/api/core/v1"

	webappv1 "my.domain/nodelabeler/api/v1"
)

// NodeLabelerReconciler reconciles a NodeLabeler object
type NodeLabelerReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=webapp.my.domain,resources=nodelabelers,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=webapp.my.domain,resources=nodelabelers/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=webapp.my.domain,resources=nodelabelers/finalizers,verbs=update

// +kubebuilder:rbac:groups="",resources=nodes,verbs=get;watch;list
// +kubebuilder:rbac:groups="",resources=nodes/status,verbs=get;update;patch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the NodeLabeler object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.18.4/pkg/reconcile
func (r *NodeLabelerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	// List all nodes in the Kubernetes cluster
	nodeList := &corev1.NodeList{}
	err := r.List(ctx, nodeList)
	if err != nil {
		return ctrl.Result{}, err
	}

	for _, node := range nodeList.Items {
		log.Log.Info("got node", "nodeName", node.Name)
		for nodeLabelKey, nodeLabelValue := range node.Labels {
			log.Log.Info("node label", "nodeName", node.Name, "nodeLabelKey", nodeLabelKey, "nodeLabelValue", nodeLabelValue)
		}
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *NodeLabelerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&webappv1.NodeLabeler{}).
		Watches(&corev1.Node{}, &handler.EnqueueRequestForObject{}).
		Complete(r)
}
