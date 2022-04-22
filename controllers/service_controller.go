/*
Copyright 2022.

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

package controllers

import (
	"context"
	"fmt"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/source"

	apiv1 "github.com/lighteness/load-balancer-controller/api/v1"
)

const (
	controllerName = "service"
)

// ServiceReconciler reconciles a Service object
type ServiceReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// ServiceMapFunc is a handler.ToRequestsFunc to be used to enqueue requests for reconciliation
func (r *ServiceReconciler) ServiceMapFunc(o client.Object) []ctrl.Request {
	svc, ok := o.(*corev1.Service)
	if !ok {
		panic(fmt.Sprintf("Expected a MachineDeployment but got a %T", o))
	}

	if svc.Spec.Type != corev1.ServiceTypeLoadBalancer {
		return nil
	}

	return []ctrl.Request{
		{
			NamespacedName: types.NamespacedName{
				Namespace: svc.Namespace,
				Name:      svc.Name,
			},
		},
	}
}

//+kubebuilder:rbac:groups=lighteness.com,resources=services,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=lighteness.com,resources=services/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=lighteness.com,resources=services/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Service object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.11.0/pkg/reconcile
func (r *ServiceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	svc := &corev1.Service{}
	if err := r.Get(ctx, req.NamespacedName, svc); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	lbSvc := &apiv1.AviLoadBalancerService{}
	if err := r.Get(ctx, req.NamespacedName, lbSvc); err != nil && apierrors.IsNotFound(err) {
		newLBService := &apiv1.AviLoadBalancerService{
			TypeMeta: metav1.TypeMeta{
				Kind:       "network.lighteness.com",
				APIVersion: "v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      svc.Name,
				Namespace: svc.Namespace,
			},
		}

		controllerutil.SetControllerReference(svc, newLBService, r.Scheme)
		r.Client.Create(ctx, newLBService)
		log.Info("newLBService created")
	}

	log.Info("request is comming", req.Namespace, req.Name)
	// TODO(user): your logic here

	return ctrl.Result{}, nil
}

func newAviLoadBalancerService(svc *corev1.Service) (lbSvc *apiv1.AviLoadBalancerService) {

	return
}

// SetupWithManager sets up the controller with the Manager.
func (r *ServiceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	c, err := controller.New(controllerName, mgr, controller.Options{
		Reconciler: r,
	})

	if err != nil {
		return err
	}
	if err := r.setupWatches(c); err != nil {
		return err
	}
	return nil

}

func (r *ServiceReconciler) setupWatches(c controller.Controller) error {
	if err := c.Watch(&source.Kind{Type: &corev1.Service{}}, handler.EnqueueRequestsFromMapFunc(r.ServiceMapFunc)); err != nil {
		return err
	}

	return nil
}
