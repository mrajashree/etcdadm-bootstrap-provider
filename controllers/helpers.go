package controllers

import (
	"context"

	bootstrapv1alpha3 "github.com/mrajashree/etcdadm-bootstrap-provider/api/v1alpha3"
	"github.com/pkg/errors"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1alpha3"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
)

// MachineToBootstrapMapFunc is a handler.ToRequestsFunc to be used to enqueue
// requests for reconciliation of EtcdadmConfig.
func (r *EtcdadmConfigReconciler) MachineToBootstrapMapFunc(o handler.MapObject) []ctrl.Request {
	var result []ctrl.Request

	m, ok := o.Object.(*clusterv1.Machine)
	if !ok {
		return nil
	}
	if m.Spec.Bootstrap.ConfigRef != nil && m.Spec.Bootstrap.ConfigRef.GroupVersionKind() == bootstrapv1alpha3.GroupVersion.WithKind("EtcdadmConfig") {
		name := client.ObjectKey{Namespace: m.Namespace, Name: m.Spec.Bootstrap.ConfigRef.Name}
		result = append(result, ctrl.Request{NamespacedName: name})
	}
	return result
}

// ClusterToEtcdadmConfigs is a handler.ToRequestsFunc to be used to enqeue
// requests for reconciliation of EtcdadmConfigs.
func (r *EtcdadmConfigReconciler) ClusterToEtcdadmConfigs(o handler.MapObject) []ctrl.Request {
	var result []ctrl.Request

	c, ok := o.Object.(*clusterv1.Cluster)
	if !ok {
		r.Log.Error(errors.Errorf("expected a Cluster but got a %T", o.Object), "failed to get EtcdadmConfigs for Cluster")
		return nil
	}

	selectors := []client.ListOption{
		client.InNamespace(c.Namespace),
		client.MatchingLabels{
			clusterv1.ClusterLabelName: c.Name,
		},
	}

	machineList := &clusterv1.MachineList{}
	if err := r.Client.List(context.Background(), machineList, selectors...); err != nil {
		r.Log.Error(err, "failed to list Machines", "Cluster", c.Name, "Namespace", c.Namespace)
		return nil
	}

	for _, m := range machineList.Items {
		if m.Spec.Bootstrap.ConfigRef != nil &&
			m.Spec.Bootstrap.ConfigRef.GroupVersionKind().GroupKind() == bootstrapv1alpha3.GroupVersion.WithKind("EtcdadmConfig").GroupKind() {
			name := client.ObjectKey{Namespace: m.Namespace, Name: m.Spec.Bootstrap.ConfigRef.Name}
			result = append(result, ctrl.Request{NamespacedName: name})
		}
	}
	return result
}
