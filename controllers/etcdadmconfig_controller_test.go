package controllers

import (
	"context"
	"testing"
	"time"

	bootstrapv1alpha3 "github.com/mrajashree/etcdadm-bootstrap-provider/api/v1alpha3"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1alpha3"
	"sigs.k8s.io/cluster-api/test/helpers"
	"sigs.k8s.io/cluster-api/util/conditions"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

func setupScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()
	if err := clusterv1.AddToScheme(scheme); err != nil {
		panic(err)
	}
	if err := bootstrapv1alpha3.AddToScheme(scheme); err != nil {
		panic(err)
	}
	if err := corev1.AddToScheme(scheme); err != nil {
		panic(err)
	}
	return scheme
}

// MachineToBootstrapMapFunc enqueues EtcdadmConfig objects for reconciliation when associated Machine object is updated
func TestEtcdadmConfigReconciler_MachineToBootstrapMapFuncReturn(t *testing.T) {
	g := NewWithT(t)

	cluster := newCluster("external-etcd-cluster")
	objs := []runtime.Object{cluster}
	var expectedConfigName string

	m := newMachine(cluster, "etcd-machine")
	configName := "etcdadm-config"
	c := newEtcdadmConfig(m, configName)
	objs = append(objs, m, c)
	expectedConfigName = configName

	fakeClient := helpers.NewFakeClientWithScheme(setupScheme(), objs...)
	reconciler := &EtcdadmConfigReconciler{
		Log:    log.Log,
		Client: fakeClient,
	}
	o := handler.MapObject{
		Object: m,
	}
	configs := reconciler.MachineToBootstrapMapFunc(o)
	g.Expect(configs[0].Name).To(Equal(expectedConfigName))
}

// Reconcile returns early if the etcdadm config is ready because it should never re-generate bootstrap data.
func TestEtcdadmConfigReconciler_Reconcile_ReturnEarlyIfEtcdadmConfigIsReady(t *testing.T) {
	g := NewWithT(t)

	config := newEtcdadmConfig(nil, "etcdadmConfig")
	config.Status.Ready = true

	objects := []runtime.Object{config}
	myclient := helpers.NewFakeClientWithScheme(setupScheme(), objects...)

	k := &EtcdadmConfigReconciler{
		Log:    log.Log,
		Client: myclient,
	}

	request := ctrl.Request{
		NamespacedName: client.ObjectKey{
			Name:      "default",
			Namespace: "etcdadmConfig",
		},
	}
	result, err := k.Reconcile(request)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(result.Requeue).To(BeFalse())
	g.Expect(result.RequeueAfter).To(Equal(time.Duration(0)))
}

// Reconcile returns an error in this case because the owning machine should not go away before the things it owns.
func TestEtcdadmConfigReconciler_Reconcile_ReturnNilIfReferencedMachineIsNotFound(t *testing.T) {
	g := NewWithT(t)

	machine := newMachine(nil, "machine")
	config := newEtcdadmConfig(machine, "etcdadmConfig")

	objects := []runtime.Object{
		// intentionally omitting machine
		config,
	}
	myclient := helpers.NewFakeClientWithScheme(setupScheme(), objects...)

	k := &EtcdadmConfigReconciler{
		Log:    log.Log,
		Client: myclient,
	}

	request := ctrl.Request{
		NamespacedName: client.ObjectKey{
			Namespace: "default",
			Name:      "etcdadmConfig",
		},
	}
	_, err := k.Reconcile(request)
	g.Expect(err).To(BeNil())
}

// Return early If the owning machine does not have an associated cluster
func TestEtcdadmConfigReconciler_Reconcile_ReturnEarlyIfMachineHasNoCluster(t *testing.T) {
	g := NewWithT(t)

	machine := newMachine(nil, "machine") // Machine without a cluster
	config := newEtcdadmConfig(machine, "etcdadmConfig")

	objects := []runtime.Object{
		machine,
		config,
	}
	myclient := helpers.NewFakeClientWithScheme(setupScheme(), objects...)

	k := &EtcdadmConfigReconciler{
		Log:    log.Log,
		Client: myclient,
	}

	request := ctrl.Request{
		NamespacedName: client.ObjectKey{
			Namespace: "default",
			Name:      "etcdadmConfig",
		},
	}
	_, err := k.Reconcile(request)
	g.Expect(err).NotTo(HaveOccurred())
}

// Return early If the associated cluster is paused (typically after clusterctl move)
func TestEtcdadmConfigReconciler_Reconcile_ReturnEarlyIfClusterIsPaused(t *testing.T) {
	g := NewWithT(t)

	cluster := newCluster("external-etcd-cluster")
	cluster.Spec.Paused = true
	machine := newMachine(cluster, "machine")
	config := newEtcdadmConfig(machine, "etcdadmConfig")

	objects := []runtime.Object{
		cluster,
		machine,
		config,
	}
	myclient := helpers.NewFakeClientWithScheme(setupScheme(), objects...)

	k := &EtcdadmConfigReconciler{
		Log:    log.Log,
		Client: myclient,
	}

	request := ctrl.Request{
		NamespacedName: client.ObjectKey{
			Namespace: "default",
			Name:      "etcdadmConfig",
		},
	}
	_, err := k.Reconcile(request)
	g.Expect(err).NotTo(HaveOccurred())
}

// First Etcdadm Machine must initialize cluster since Cluster.Status.ManagedExternalEtcdInitialized is false and lock is not acquired
func TestEtcdadmConfigReconciler_InitializeEtcdIfInitLockIsNotAquired(t *testing.T) {
	g := NewWithT(t)

	cluster := newCluster("external-etcd-cluster")
	machine := newMachine(cluster, "machine")
	config := newEtcdadmConfig(machine, "etcdadmConfig")

	objects := []runtime.Object{
		cluster,
		machine,
		config,
	}
	myclient := helpers.NewFakeClientWithScheme(setupScheme(), objects...)

	k := &EtcdadmConfigReconciler{
		Log:             log.Log,
		Client:          myclient,
		EtcdadmInitLock: &etcdInitLocker{},
	}
	request := ctrl.Request{
		NamespacedName: client.ObjectKey{
			Namespace: "default",
			Name:      "etcdadmConfig",
		},
	}
	result, err := k.Reconcile(request)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(result.Requeue).To(BeFalse())
	g.Expect(result.RequeueAfter).To(BeZero())

	configKey, _ := client.ObjectKeyFromObject(config)
	g.Expect(myclient.Get(context.TODO(), configKey, config)).To(Succeed())
	c := conditions.Get(config, bootstrapv1alpha3.DataSecretAvailableCondition)
	g.Expect(c).ToNot(BeNil())
	g.Expect(c.Status).To(Equal(corev1.ConditionTrue))
}

// If Init lock is already acquired but cluster status does not show etcd initialized and another etcdadmConfig is created, requeue it
func TestEtcdadmConfigReconciler_RequeueIfInitLockIsAquired(t *testing.T) {
	g := NewWithT(t)

	cluster := newCluster("external-etcd-cluster")
	machine := newMachine(cluster, "machine")
	config := newEtcdadmConfig(machine, "etcdadmConfig")

	objects := []runtime.Object{
		cluster,
		machine,
		config,
	}
	myclient := helpers.NewFakeClientWithScheme(setupScheme(), objects...)

	k := &EtcdadmConfigReconciler{
		Log:             log.Log,
		Client:          myclient,
		EtcdadmInitLock: &etcdInitLocker{locked: true},
	}
	request := ctrl.Request{
		NamespacedName: client.ObjectKey{
			Namespace: "default",
			Name:      "etcdadmConfig",
		},
	}
	result, err := k.Reconcile(request)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(result.Requeue).To(BeFalse())
	g.Expect(result.RequeueAfter).To(Equal(30 * time.Second))
	c := conditions.Get(config, bootstrapv1alpha3.DataSecretAvailableCondition)
	g.Expect(c).To(BeNil())
}

func TestEtcdadmConfigReconciler_JoinMemberIfEtcdIsInitialized(t *testing.T) {
	g := NewWithT(t)

	cluster := newCluster("external-etcd-cluster")
	cluster.Status.ManagedExternalEtcdInitialized = true
	machine := newMachine(cluster, "machine")
	config := newEtcdadmConfig(machine, "etcdadmConfig")

	objects := []runtime.Object{
		cluster,
		machine,
		config,
	}
	myclient := helpers.NewFakeClientWithScheme(setupScheme(), objects...)

	k := &EtcdadmConfigReconciler{
		Log:             log.Log,
		Client:          myclient,
		EtcdadmInitLock: &etcdInitLocker{},
	}
	request := ctrl.Request{
		NamespacedName: client.ObjectKey{
			Namespace: "default",
			Name:      "etcdadmConfig",
		},
	}
	result, err := k.Reconcile(request)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(result.Requeue).To(BeFalse())
	g.Expect(result.RequeueAfter).To(BeZero())

	configKey, _ := client.ObjectKeyFromObject(config)
	g.Expect(myclient.Get(context.TODO(), configKey, config)).To(Succeed())
	c := conditions.Get(config, bootstrapv1alpha3.DataSecretAvailableCondition)
	g.Expect(c).ToNot(BeNil())
	g.Expect(c.Status).To(Equal(corev1.ConditionTrue))
}

// newCluster creates a CAPI Cluster object
func newCluster(name string) *clusterv1.Cluster {
	c := &clusterv1.Cluster{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Cluster",
			APIVersion: clusterv1.GroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      name,
		},
	}

	return c
}

// newMachine return a CAPI Machine object; if cluster is not nil, the machine is linked to the cluster as well
func newMachine(cluster *clusterv1.Cluster, name string) *clusterv1.Machine {
	machine := &clusterv1.Machine{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Machine",
			APIVersion: clusterv1.GroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      name,
		},
		Spec: clusterv1.MachineSpec{
			Bootstrap: clusterv1.Bootstrap{
				ConfigRef: &corev1.ObjectReference{
					Kind:       "EtcdadmConfig",
					APIVersion: bootstrapv1alpha3.GroupVersion.String(),
				},
			},
		},
	}
	if cluster != nil {
		machine.Spec.ClusterName = cluster.Name
		machine.ObjectMeta.Labels = map[string]string{
			clusterv1.ClusterLabelName: cluster.Name,
		}
	}
	return machine
}

// newEtcdadmConfig generates an EtcdadmConfig object for the external etcd cluster
func newEtcdadmConfig(machine *clusterv1.Machine, name string) *bootstrapv1alpha3.EtcdadmConfig {
	config := &bootstrapv1alpha3.EtcdadmConfig{
		TypeMeta: metav1.TypeMeta{
			Kind:       "EtcdadmConfig",
			APIVersion: bootstrapv1alpha3.GroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: "default",
		},
	}

	if machine != nil {
		config.ObjectMeta.OwnerReferences = []metav1.OwnerReference{
			{
				Kind:       "Machine",
				APIVersion: clusterv1.GroupVersion.String(),
				Name:       machine.Name,
			},
		}
		machine.Spec.Bootstrap.ConfigRef.Name = config.Name
		machine.Spec.Bootstrap.ConfigRef.Namespace = config.Namespace
	}
	return config
}

type etcdInitLocker struct {
	locked bool
}

func (m *etcdInitLocker) Lock(_ context.Context, _ *clusterv1.Cluster, _ *clusterv1.Machine) bool {
	if !m.locked {
		m.locked = true
		return true
	}
	return false
}

func (m *etcdInitLocker) Unlock(_ context.Context, _ *clusterv1.Cluster) bool {
	if m.locked {
		m.locked = false
	}
	return true
}
