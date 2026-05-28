/*
Copyright 2026.

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
	"errors"
	"fmt"
	"net/netip"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	clusterv1 "sigs.k8s.io/cluster-api/api/core/v1beta2"
	clusterutil "sigs.k8s.io/cluster-api/util"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	infrav1 "voigt.tngl.sh/cluster-api-provider-stackit/api/v1alpha1"
	"voigt.tngl.sh/cluster-api-provider-stackit/pkg/cloud"
	"voigt.tngl.sh/cluster-api-provider-stackit/pkg/scope"
	"voigt.tngl.sh/cluster-api-provider-stackit/pkg/util"
)

const (
	// defaultAPIServerPort is used when an LB is created without an explicit port.
	defaultAPIServerPort int32 = 6443

	bootstrapTargetName = "capi-bootstrap-placeholder"
)

// StackitClusterReconciler reconciles a StackitCluster object.
type StackitClusterReconciler struct {
	client.Client
	Scheme *runtime.Scheme

	// CloudClientFactory builds a cloud.Client from parsed credentials. It is
	// injected so tests can swap in the in-memory fake.
	CloudClientFactory cloud.Factory
}

// +kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=stackitclusters,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=stackitclusters/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=stackitclusters/finalizers,verbs=update
// +kubebuilder:rbac:groups=cluster.x-k8s.io,resources=clusters,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch

// Reconcile implements the spec section 18 flow.
func (r *StackitClusterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (_ ctrl.Result, err error) {
	log := logf.FromContext(ctx)

	stackitCluster := &infrav1.StackitCluster{}
	if err := r.Get(ctx, req.NamespacedName, stackitCluster); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	cluster, err := clusterutil.GetOwnerCluster(ctx, r.Client, stackitCluster.ObjectMeta)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("get owner cluster: %w", err)
	}
	if cluster == nil {
		log.Info("StackitCluster has no owning Cluster yet, requeueing")
		return ctrl.Result{}, nil
	}

	clusterScope, err := scope.NewClusterScope(r.Client, cluster, stackitCluster)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("create cluster scope: %w", err)
	}
	defer func() {
		if patchErr := clusterScope.PatchObject(ctx); patchErr != nil && err == nil {
			err = patchErr
		}
	}()

	if paused, message := reconciliationPaused(cluster, stackitCluster); paused {
		setPausedCondition(&stackitCluster.Status.Conditions, stackitCluster.Generation, true, message)
		return ctrl.Result{}, nil
	}
	setPausedCondition(&stackitCluster.Status.Conditions, stackitCluster.Generation, false, "")

	if !stackitCluster.DeletionTimestamp.IsZero() {
		return r.reconcileDelete(ctx, clusterScope)
	}
	return r.reconcileNormal(ctx, clusterScope)
}

func (r *StackitClusterReconciler) reconcileNormal(ctx context.Context, s *scope.ClusterScope) (ctrl.Result, error) {
	log := logf.FromContext(ctx)
	sc := s.StackitCluster

	if !controllerutil.ContainsFinalizer(sc, infrav1.ClusterFinalizer) {
		controllerutil.AddFinalizer(sc, infrav1.ClusterFinalizer)
	}
	sc.Status.FailureDomains = stackitFailureDomains(sc.Spec.Region)

	cloudClient, err := r.buildCloudClient(ctx, sc)
	if err != nil {
		sc.Status.Ready = false
		util.SetCondition(&sc.Status.Conditions, infrav1.ClusterCredentialsReadyCondition,
			metav1.ConditionFalse, "CredentialsInvalid", err.Error(), sc.Generation)
		util.SetCondition(&sc.Status.Conditions, infrav1.ClusterReadyCondition,
			metav1.ConditionFalse, "CredentialsInvalid", err.Error(), sc.Generation)
		// Auth/invalid input errors should not aggressively requeue.
		if cloud.IsUnauthorized(err) || cloud.IsInvalidInput(err) || errors.Is(err, util.ErrCredentialsInvalid) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}
	util.SetCondition(&sc.Status.Conditions, infrav1.ClusterCredentialsReadyCondition,
		metav1.ConditionTrue, "Available", "", sc.Generation)

	network, err := cloudClient.GetNetwork(ctx, sc.Spec.Network.ID)
	if err != nil {
		sc.Status.Ready = false
		util.SetCondition(&sc.Status.Conditions, infrav1.ClusterNetworkReadyCondition,
			metav1.ConditionFalse, "NetworkNotFound", err.Error(), sc.Generation)
		util.SetCondition(&sc.Status.Conditions, infrav1.ClusterReadyCondition,
			metav1.ConditionFalse, "NetworkNotFound", err.Error(), sc.Generation)
		if cloud.IsRetryable(err) {
			return ctrl.Result{Requeue: true}, nil
		}
		return ctrl.Result{}, nil
	}
	util.SetCondition(&sc.Status.Conditions, infrav1.ClusterNetworkReadyCondition,
		metav1.ConditionTrue, "Available", "", sc.Generation)

	if sc.Spec.APIServerLoadBalancer.Enabled {
		lb, err := cloudClient.EnsureAPIServerLoadBalancer(ctx, cloud.LoadBalancerInput{
			Name:      sc.Name + "-apiserver",
			ProjectID: sc.Spec.ProjectID,
			Region:    sc.Spec.Region,
			NetworkID: sc.Spec.Network.ID,
			Port:      defaultAPIServerPort,
			Tags:      util.ClusterTags(sc.Name, sc.Namespace, sc.Spec.AdditionalLabels),
			Targets: []cloud.LoadBalancerTargetInput{{
				Name: bootstrapTargetName,
				IP:   bootstrapTargetIP(network),
				Port: defaultAPIServerPort,
			}},
		})
		if err != nil {
			sc.Status.Ready = false
			util.SetCondition(&sc.Status.Conditions, infrav1.ClusterLoadBalancerReadyCondition,
				metav1.ConditionFalse, "LoadBalancerError", err.Error(), sc.Generation)
			util.SetCondition(&sc.Status.Conditions, infrav1.ClusterReadyCondition,
				metav1.ConditionFalse, "LoadBalancerError", err.Error(), sc.Generation)
			if cloud.IsRetryable(err) {
				return ctrl.Result{Requeue: true}, nil
			}
			return ctrl.Result{}, nil
		}
		if lb != nil {
			sc.Status.APIServerLoadBalancerID = lb.ID
		}
		if lb == nil || lb.IP == "" {
			sc.Status.Ready = false
			util.SetCondition(&sc.Status.Conditions, infrav1.ClusterLoadBalancerReadyCondition,
				metav1.ConditionFalse, "Provisioning", "waiting for API server load balancer IP address", sc.Generation)
			util.SetCondition(&sc.Status.Conditions, infrav1.ClusterReadyCondition,
				metav1.ConditionFalse, "Provisioning", "waiting for API server load balancer IP address", sc.Generation)
			return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
		}
		sc.Spec.ControlPlaneEndpoint = clusterv1.APIEndpoint{Host: lb.IP, Port: defaultAPIServerPort}
		sc.Status.APIServerEndpoint = sc.Spec.ControlPlaneEndpoint
		util.SetCondition(&sc.Status.Conditions, infrav1.ClusterLoadBalancerReadyCondition,
			metav1.ConditionTrue, "Available", "", sc.Generation)
	} else if sc.Spec.ControlPlaneEndpoint.Host != "" {
		sc.Status.APIServerEndpoint = sc.Spec.ControlPlaneEndpoint
		util.SetCondition(&sc.Status.Conditions, infrav1.ClusterLoadBalancerReadyCondition,
			metav1.ConditionTrue, "Skipped", "external endpoint provided", sc.Generation)
	} else {
		sc.Status.Ready = false
		util.SetCondition(&sc.Status.Conditions, infrav1.ClusterLoadBalancerReadyCondition,
			metav1.ConditionFalse, "EndpointMissing", "apiServerLoadBalancer.enabled is false and controlPlaneEndpoint is empty", sc.Generation)
		util.SetCondition(&sc.Status.Conditions, infrav1.ClusterReadyCondition,
			metav1.ConditionFalse, "EndpointMissing", "apiServerLoadBalancer.enabled is false and controlPlaneEndpoint is empty", sc.Generation)
		return ctrl.Result{}, nil
	}

	sc.Status.Ready = true
	sc.Status.Initialization.Provisioned = true
	util.SetCondition(&sc.Status.Conditions, infrav1.ClusterReadyCondition,
		metav1.ConditionTrue, "Available", "", sc.Generation)
	log.V(1).Info("StackitCluster ready", "endpoint", sc.Status.APIServerEndpoint)
	return ctrl.Result{}, nil
}

func stackitFailureDomains(region string) []clusterv1.FailureDomain {
	controlPlane := true
	return []clusterv1.FailureDomain{
		{
			Name:         region + "-1",
			ControlPlane: &controlPlane,
			Attributes: map[string]string{
				"region": region,
			},
		},
		{
			Name:         region + "-2",
			ControlPlane: &controlPlane,
			Attributes: map[string]string{
				"region": region,
			},
		},
		{
			Name:         region + "-3",
			ControlPlane: &controlPlane,
			Attributes: map[string]string{
				"region": region,
			},
		},
	}
}

func bootstrapTargetIP(network *cloud.Network) string {
	if network == nil {
		return "10.0.0.1"
	}
	for _, prefixValue := range network.IPv4Prefixes {
		prefix, err := netip.ParsePrefix(prefixValue)
		if err != nil || !prefix.Addr().Is4() {
			continue
		}
		address := prefix.Masked().Addr()
		for i := 0; i < 10; i++ {
			address = address.Next()
		}
		if prefix.Contains(address) {
			return address.String()
		}
	}
	return "10.0.0.1"
}

func (r *StackitClusterReconciler) reconcileDelete(ctx context.Context, s *scope.ClusterScope) (ctrl.Result, error) {
	sc := s.StackitCluster
	if sc.Status.APIServerLoadBalancerID != "" {
		cloudClient, err := r.buildCloudClient(ctx, sc)
		if err != nil {
			// If we cannot reach the cloud during delete, surface the condition
			// but do not block forever; finalizer removal is gated on the LB
			// deletion succeeding (or being already absent).
			util.SetCondition(&sc.Status.Conditions, infrav1.ClusterCredentialsReadyCondition,
				metav1.ConditionFalse, "CredentialsInvalid", err.Error(), sc.Generation)
			return ctrl.Result{}, err
		}
		if err := cloudClient.DeleteAPIServerLoadBalancer(ctx, sc.Status.APIServerLoadBalancerID); err != nil && !cloud.IsNotFound(err) {
			return ctrl.Result{}, err
		}
		sc.Status.APIServerLoadBalancerID = ""
	}
	controllerutil.RemoveFinalizer(sc, infrav1.ClusterFinalizer)
	return ctrl.Result{}, nil
}

func (r *StackitClusterReconciler) buildCloudClient(ctx context.Context, sc *infrav1.StackitCluster) (cloud.Client, error) {
	if r.CloudClientFactory == nil {
		return nil, errors.New("CloudClientFactory is not configured")
	}
	secret := &corev1.Secret{}
	ns := sc.Spec.CredentialsSecretRef.Namespace
	if ns == "" {
		ns = sc.Namespace
	}
	key := types.NamespacedName{Namespace: ns, Name: sc.Spec.CredentialsSecretRef.Name}
	if err := r.Get(ctx, key, secret); err != nil {
		return nil, fmt.Errorf("get credentials secret %s: %w", key, err)
	}
	creds, err := util.ParseCredentialsSecret(secret, sc.Spec.ProjectID, sc.Spec.Region)
	if err != nil {
		return nil, err
	}
	return r.CloudClientFactory(ctx, creds)
}

func (r *StackitClusterReconciler) stackitClusterRequestsForCluster(_ context.Context, obj client.Object) []reconcile.Request {
	cluster, ok := obj.(*clusterv1.Cluster)
	if !ok || !isStackitClusterRef(cluster.Spec.InfrastructureRef) {
		return nil
	}
	return []reconcile.Request{{
		NamespacedName: types.NamespacedName{
			Namespace: cluster.Namespace,
			Name:      cluster.Spec.InfrastructureRef.Name,
		},
	}}
}

func isStackitClusterRef(ref clusterv1.ContractVersionedObjectReference) bool {
	return ref.APIGroup == infrav1.GroupVersion.Group &&
		ref.Kind == "StackitCluster" &&
		ref.Name != ""
}

// SetupWithManager registers the controller with the manager.
func (r *StackitClusterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&infrav1.StackitCluster{}).
		Watches(&clusterv1.Cluster{}, handler.EnqueueRequestsFromMapFunc(r.stackitClusterRequestsForCluster)).
		Named("stackitcluster").
		Complete(r)
}
