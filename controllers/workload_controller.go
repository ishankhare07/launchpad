/*


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

	istiov1alpha3 "istio.io/client-go/pkg/apis/networking/v1alpha3"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	labsv1alpha1 "github.com/ishankhare07/launchpad/api/v1alpha1"
	"github.com/ishankhare07/launchpad/pkg/spawn"
)

// WorkloadReconciler reconciles a Workload object
type WorkloadReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=labs.ishankhare.dev,resources=workloads,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=labs.ishankhare.dev,resources=workloads/status,verbs=get;update;patch

func (r *WorkloadReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = context.Background()
	reqLogger := r.Log.WithValues("workload", req.NamespacedName)

	// your logic here
	reqLogger.Info("====== Reconciling Workload =======")
	instance := &labsv1alpha1.Workload{}

	err := r.Get(context.TODO(), req.NamespacedName, instance)
	if err != nil {
		// object not found, could have been deleted after
		// reconcile request, hence don't requeue
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		// error reading the object, requeue the request
		return ctrl.Result{}, err
	}

	reqLogger.Info("Workload created for following targets.")
	for _, target := range instance.Spec.Targets {
		reqLogger.Info("target",
			"name", target.Name,
			"namespace", target.Namespace,
			"hosts", fmt.Sprintf("%v", target.TrafficSplit.Hosts))
		reqLogger.Info("subset",
			"subset name", target.TrafficSplit.SubsetName,
			"subset labels", fmt.Sprintf("%v", target.TrafficSplit.SubsetLabels),
			"weight", target.TrafficSplit.Weight)
	}

	// check current status of owned resources
	if instance.Status.VirtualServiceState == "" {
		instance.Status.VirtualServiceState = labsv1alpha1.PhasePending
	}

	if instance.Status.DestinationRuleState == "" {
		instance.Status.DestinationRuleState = labsv1alpha1.PhasePending
	}

	// check virtual service status
	vsResult, err := r.checkVirtualService(instance, req, reqLogger)
	if err != nil {
		return vsResult, err
	}

	// check destination rule status
	destRuleResult, err := r.checkDestinationRuleStatus(instance, req, reqLogger)
	if err != nil {
		return destRuleResult, err
	}

	// update status
	err = r.Status().Update(context.TODO(), instance)
	if err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *WorkloadReconciler) checkVirtualService(instance *labsv1alpha1.Workload, req ctrl.Request, logger logr.Logger) (ctrl.Result, error) {
	switch instance.Status.VirtualServiceState {
	case labsv1alpha1.PhasePending:
		logger.Info("Virtual Service", "PHASE:", instance.Status.VirtualServiceState)
		logger.Info("Transitioning state to create Virtual Service")
		instance.Status.VirtualServiceState = labsv1alpha1.PhaseCreated
	case labsv1alpha1.PhaseCreated:
		logger.Info("Virtual Service", "PHASE:", instance.Status.VirtualServiceState)
		query := &istiov1alpha3.VirtualService{}

		// check if virtual service already exists
		lookupKey := types.NamespacedName{
			Name:      instance.GetIstioResourceName(),
			Namespace: instance.GetNamespace(),
		}
		err := r.Get(context.TODO(), lookupKey, query)
		if err != nil && errors.IsNotFound(err) {
			logger.Info("virtual service not found but should exist", "lookup key", lookupKey)
			logger.Info(err.Error())
			// virtual service got deleted or hasn't been created yet
			// create one now
			vs := spawn.CreateVirtualService(instance)
			err = ctrl.SetControllerReference(instance, vs, r.Scheme)
			if err != nil {
				logger.Error(err, "Error setting controller reference")
				return ctrl.Result{}, err
			}

			err = r.Create(context.TODO(), vs)
			if err != nil {
				logger.Error(err, "Unable to create Virtual Service")
				return ctrl.Result{}, err
			}

			logger.Info("Successfully created virtual service")
		} else if err != nil {
			logger.Error(err, "Unable to create Virtual Service")
			return ctrl.Result{}, err
		} else {
			// don't requeue, it will happen automatically when
			// virtual service status changes
			return ctrl.Result{}, nil
		}

		// more fields related to virtual service status can be checked
		// see more at https://pkg.go.dev/istio.io/api/meta/v1alpha1#IstioStatus

	}

	return ctrl.Result{}, nil
}

func (r *WorkloadReconciler) checkDestinationRuleStatus(instance *labsv1alpha1.Workload, req ctrl.Request, logger logr.Logger) (ctrl.Result, error) {
	switch instance.Status.DestinationRuleState {
	case labsv1alpha1.PhasePending:
		logger.Info("Destination Rule", "PHASE:", instance.Status.DestinationRuleState)
		logger.Info("Transitioning state to create Destination Rule")
		instance.Status.DestinationRuleState = labsv1alpha1.PhaseCreated
	case labsv1alpha1.PhaseCreated:
		logger.Info("Destination Rule", "PHASE:", instance.Status.DestinationRuleState)
		query := &istiov1alpha3.DestinationRule{}

		// check if destination rule already exists
		lookupKey := types.NamespacedName{
			Name:      instance.GetIstioResourceName(),
			Namespace: instance.GetNamespace(),
		}
		err := r.Get(context.TODO(),
			lookupKey, query)
		if err != nil && errors.IsNotFound(err) {
			logger.Info("destination rule not found but should exist", "req key", lookupKey)
			logger.Info(err.Error())
			// destination rule got deleted or hasn't been created yet
			// create one now

			dr := spawn.CreateDestinationRule(instance)
			err := ctrl.SetControllerReference(instance, dr, r.Scheme)
			if err != nil {
				logger.Error(err, "Error setting controller reference")
				return ctrl.Result{}, err
			}

			err = r.Create(context.TODO(), dr)
			if err != nil {
				logger.Error(err, "Unable to create Destination Rule")
				return ctrl.Result{}, err
			}

			logger.Info("Successfully created Destination Rule")
		} else if err != nil {
			logger.Error(err, "Unable to create destination rule")
			return ctrl.Result{}, err
		} else {
			// don't requeue, it will happen automatically when
			// destination rule status changes
			return ctrl.Result{}, nil
		}
	}

	return ctrl.Result{}, nil
}

func (r *WorkloadReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&labsv1alpha1.Workload{}).
		Owns(&istiov1alpha3.VirtualService{}).
		Owns(&istiov1alpha3.DestinationRule{}).
		Complete(r)
}
