package controllers

import (
	"context"
	"fmt"
	"k8s.io/utils/pointer"
	"testing"
	"time"

	etcdbootstrapv1 "github.com/mrajashree/etcdadm-bootstrap-provider/api/v1beta1"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/cluster-api/util/conditions"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

func setupScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()
	if err := clusterv1.AddToScheme(scheme); err != nil {
		panic(err)
	}
	if err := etcdbootstrapv1.AddToScheme(scheme); err != nil {
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
	objs := []client.Object{cluster}
	var expectedConfigName string

	m := newMachine(cluster, "etcd-machine")
	configName := "etcdadm-config"
	c := newEtcdadmConfig(m, configName, etcdbootstrapv1.CloudConfig)
	objs = append(objs, m, c)
	expectedConfigName = configName

	fakeClient := fake.NewClientBuilder().WithScheme(setupScheme()).WithObjects(objs...).Build()
	reconciler := &EtcdadmConfigReconciler{
		Log:    log.Log,
		Client: fakeClient,
	}
	o := m
	configs := reconciler.MachineToBootstrapMapFunc(o)
	g.Expect(configs[0].Name).To(Equal(expectedConfigName))
}

func TestEtcdadmConfigReconciler_ClusterToEtcdadmConfigs(t *testing.T) {
	g := NewWithT(t)

	cluster := newCluster("external-etcd-cluster")
	objs := []client.Object{cluster}
	var expectedConfigName string

	m := newMachine(cluster, "etcd-machine")
	configName := "etcdadm-config"
	c := newEtcdadmConfig(m, configName, etcdbootstrapv1.CloudConfig)
	objs = append(objs, m, c)
	expectedConfigName = configName

	fakeClient := fake.NewClientBuilder().WithScheme(setupScheme()).WithObjects(objs...).Build()
	reconciler := &EtcdadmConfigReconciler{
		Log:    log.Log,
		Client: fakeClient,
	}
	o := cluster
	configs := reconciler.ClusterToEtcdadmConfigs(o)
	g.Expect(configs[0].Name).To(Equal(expectedConfigName))
}

// Reconcile returns early if the etcdadm config is ready because it should never re-generate bootstrap data.
func TestEtcdadmConfigReconciler_Reconcile_ReturnEarlyIfEtcdadmConfigIsReady(t *testing.T) {
	g := NewWithT(t)

	cluster := newCluster("external-etcd-cluster")
	m := newMachine(cluster, "etcd-machine")
	config := newEtcdadmConfig(m, "etcdadmConfig", etcdbootstrapv1.CloudConfig)
	config.Status.Ready = true

	objects := []client.Object{
		cluster,
		m,
		config,
	}
	myclient := fake.NewClientBuilder().WithScheme(setupScheme()).WithObjects(objects...).Build()

	k := &EtcdadmConfigReconciler{
		Log:    log.Log,
		Client: myclient,
	}

	request := ctrl.Request{
		NamespacedName: client.ObjectKey{
			Name:      "etcdadmConfig",
			Namespace: "default",
		},
	}
	result, err := k.Reconcile(ctx, request)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(result.Requeue).To(BeFalse())
	g.Expect(result.RequeueAfter).To(Equal(time.Duration(0)))
}

// Reconcile returns an error in this case because the owning machine should not go away before the things it owns.
func TestEtcdadmConfigReconciler_Reconcile_ReturnNilIfReferencedMachineIsNotFound(t *testing.T) {
	g := NewWithT(t)

	machine := newMachine(nil, "machine")
	config := newEtcdadmConfig(machine, "etcdadmConfig", etcdbootstrapv1.CloudConfig)

	objects := []client.Object{
		// intentionally omitting machine
		config,
	}
	myclient := fake.NewClientBuilder().WithScheme(setupScheme()).WithObjects(objects...).Build()

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
	_, err := k.Reconcile(ctx, request)
	g.Expect(err).To(BeNil())
}

// Return early If the owning machine does not have an associated cluster
func TestEtcdadmConfigReconciler_Reconcile_ReturnEarlyIfMachineHasNoCluster(t *testing.T) {
	g := NewWithT(t)

	machine := newMachine(nil, "machine") // Machine without a cluster (no cluster label present)
	config := newEtcdadmConfig(machine, "etcdadmConfig", etcdbootstrapv1.CloudConfig)

	objects := []client.Object{
		machine,
		config,
	}
	myclient := fake.NewClientBuilder().WithScheme(setupScheme()).WithObjects(objects...).Build()

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
	_, err := k.Reconcile(ctx, request)
	g.Expect(err).NotTo(HaveOccurred())
}

// Return early If the owning machine has an associated cluster but it does not exist
func TestEtcdadmConfigReconciler_Reconcile_ReturnEarlyIfMachinesClusterDoesNotExist(t *testing.T) {
	g := NewWithT(t)

	machine := newMachine(newCluster("external-etcd-cluster"), "machine") // Machine with a cluster label, but cluster won't exist
	config := newEtcdadmConfig(machine, "etcdadmConfig", etcdbootstrapv1.CloudConfig)

	objects := []client.Object{
		machine,
		config,
		// not create cluster object
	}
	myclient := fake.NewClientBuilder().WithScheme(setupScheme()).WithObjects(objects...).Build()

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
	_, err := k.Reconcile(ctx, request)
	g.Expect(err).NotTo(HaveOccurred())
}

// Return early If the associated cluster is paused (typically after clusterctl move)
func TestEtcdadmConfigReconciler_Reconcile_ReturnEarlyIfClusterIsPaused(t *testing.T) {
	g := NewWithT(t)

	cluster := newCluster("external-etcd-cluster")
	cluster.Spec.Paused = true
	machine := newMachine(cluster, "machine")
	config := newEtcdadmConfig(machine, "etcdadmConfig", etcdbootstrapv1.CloudConfig)

	objects := []client.Object{
		cluster,
		machine,
		config,
	}
	myclient := fake.NewClientBuilder().WithScheme(setupScheme()).WithObjects(objects...).Build()

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
	_, err := k.Reconcile(ctx, request)
	g.Expect(err).NotTo(HaveOccurred())
}

// First Etcdadm Machine must initialize cluster since Cluster.Status.ManagedExternalEtcdInitialized is false and lock is not acquired
func TestEtcdadmConfigReconciler_InitializeEtcdIfInitLockIsNotAquired_Cloudinit(t *testing.T) {
	g := NewWithT(t)

	cluster := newCluster("external-etcd-cluster")
	machine := newMachine(cluster, "machine")
	config := newEtcdadmConfig(machine, "etcdadmConfig", etcdbootstrapv1.CloudConfig)

	objects := []client.Object{
		cluster,
		machine,
		config,
	}
	myclient := fake.NewClientBuilder().WithScheme(setupScheme()).WithObjects(objects...).Build()

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
	result, err := k.Reconcile(ctx, request)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(result.Requeue).To(BeFalse())
	g.Expect(result.RequeueAfter).To(BeZero())

	configKey := client.ObjectKeyFromObject(config)
	g.Expect(myclient.Get(context.TODO(), configKey, config)).To(Succeed())
	c := conditions.Get(config, etcdbootstrapv1.DataSecretAvailableCondition)
	g.Expect(c).ToNot(BeNil())
	g.Expect(c.Status).To(Equal(corev1.ConditionTrue))
}

// First Etcdadm Machine must initialize cluster since Cluster.Status.ManagedExternalEtcdInitialized is false and lock is not acquired
func TestEtcdadmConfigReconciler_InitializeEtcdIfInitLockIsNotAquired_Bottlerocket(t *testing.T) {
	g := NewWithT(t)

	cluster := newCluster("external-etcd-cluster")
	machine := newMachine(cluster, "machine")
	config := newEtcdadmConfig(machine, "etcdadmConfig", etcdbootstrapv1.Bottlerocket)

	objects := []client.Object{
		cluster,
		machine,
		config,
	}
	myclient := fake.NewClientBuilder().WithScheme(setupScheme()).WithObjects(objects...).Build()

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
	result, err := k.Reconcile(ctx, request)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(result.Requeue).To(BeFalse())
	g.Expect(result.RequeueAfter).To(BeZero())

	configKey := client.ObjectKeyFromObject(config)
	g.Expect(myclient.Get(context.TODO(), configKey, config)).To(Succeed())
	c := conditions.Get(config, etcdbootstrapv1.DataSecretAvailableCondition)
	g.Expect(c).ToNot(BeNil())
	g.Expect(c.Status).To(Equal(corev1.ConditionTrue))
}

// The Secret containing bootstrap data for first etcdadm machine exists, but etcdadmConfig status is not patched yet
func TestEtcdadmConfigBootstrapDataSecretCreatedStatusNotPatched(t *testing.T) {
	g := NewWithT(t)

	cluster := newCluster("external-etcd-cluster")
	machine := newMachine(cluster, "machine")
	config := newEtcdadmConfig(machine, "etcdadmConfig", etcdbootstrapv1.CloudConfig)

	objects := []client.Object{
		cluster,
		machine,
		config,
	}
	myclient := fake.NewClientBuilder().WithScheme(setupScheme()).WithObjects(objects...).Build()

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
	dataSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      config.Name,
			Namespace: config.Namespace,
			Labels: map[string]string{
				clusterv1.ClusterLabelName: cluster.Name,
			},
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: etcdbootstrapv1.GroupVersion.String(),
					Kind:       config.Kind,
					Name:       config.Name,
					UID:        config.UID,
					Controller: pointer.BoolPtr(true),
				},
			},
		},
		Data: map[string][]byte{
			"value": nil,
		},
		Type: clusterv1.ClusterSecretType,
	}
	err := myclient.Create(ctx, dataSecret)
	g.Expect(err).ToNot(HaveOccurred())
	result, err := k.Reconcile(ctx, request)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(result.Requeue).To(BeFalse())
	g.Expect(result.RequeueAfter).To(BeZero())

	configKey := client.ObjectKeyFromObject(config)
	g.Expect(myclient.Get(context.TODO(), configKey, config)).To(Succeed())
	c := conditions.Get(config, etcdbootstrapv1.DataSecretAvailableCondition)
	g.Expect(c).ToNot(BeNil())
	g.Expect(c.Status).To(Equal(corev1.ConditionTrue))
}

// NA for EKS-A since it etcdadm is built into the OVA
func TestEtcdadmConfigReconciler_PreEtcdadmCommandsWhenEtcdadmNotBuiltin(t *testing.T) {
	g := NewWithT(t)

	cluster := newCluster("external-etcd-cluster")
	machine := newMachine(cluster, "machine")
	config := newEtcdadmConfig(machine, "etcdadmConfig", etcdbootstrapv1.CloudConfig)
	config.Spec.EtcdadmBuiltin = false
	objects := []client.Object{
		cluster,
		machine,
		config,
	}
	myclient := fake.NewClientBuilder().WithScheme(setupScheme()).WithObjects(objects...).Build()

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
	result, err := k.Reconcile(ctx, request)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(result.Requeue).To(BeFalse())
	g.Expect(result.RequeueAfter).To(BeZero())

	configKey := client.ObjectKeyFromObject(config)
	g.Expect(myclient.Get(context.TODO(), configKey, config)).To(Succeed())
	c := conditions.Get(config, etcdbootstrapv1.DataSecretAvailableCondition)
	g.Expect(c).ToNot(BeNil())
	g.Expect(c.Status).To(Equal(corev1.ConditionTrue))
}

// If Init lock is already acquired but cluster status does not show etcd initialized and another etcdadmConfig is created, requeue it
func TestEtcdadmConfigReconciler_RequeueIfInitLockIsAquired(t *testing.T) {
	g := NewWithT(t)

	cluster := newCluster("external-etcd-cluster")
	machine := newMachine(cluster, "machine")
	config := newEtcdadmConfig(machine, "etcdadmConfig", etcdbootstrapv1.CloudConfig)

	objects := []client.Object{
		cluster,
		machine,
		config,
	}
	myclient := fake.NewClientBuilder().WithScheme(setupScheme()).WithObjects(objects...).Build()

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
	result, err := k.Reconcile(ctx, request)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(result.Requeue).To(BeFalse())
	g.Expect(result.RequeueAfter).To(Equal(30 * time.Second))
	c := conditions.Get(config, etcdbootstrapv1.DataSecretAvailableCondition)
	g.Expect(c).To(BeNil())
}

func TestEtcdadmConfigReconciler_JoinMemberInitSecretNotReady(t *testing.T) {
	g := NewWithT(t)

	cluster := newCluster("external-etcd-cluster")
	cluster.Status.ManagedExternalEtcdInitialized = true
	conditions.MarkTrue(cluster, clusterv1.ManagedExternalEtcdClusterInitializedCondition)
	machine := newMachine(cluster, "machine")
	config := newEtcdadmConfig(machine, "etcdadmConfig", etcdbootstrapv1.CloudConfig)

	objects := []client.Object{
		cluster,
		machine,
		config,
		// not generating etcd init secret, so joinMember won't proceed since etcd cluster is not initialized fully
	}
	myclient := fake.NewClientBuilder().WithScheme(setupScheme()).WithObjects(objects...).Build()

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
	_, err := k.Reconcile(ctx, request)
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).Should(ContainSubstring("not found"))

}

func TestEtcdadmConfigReconciler_JoinMemberIfEtcdIsInitialized_CloudInit(t *testing.T) {
	g := NewWithT(t)

	cluster := newCluster("external-etcd-cluster")
	cluster.Status.ManagedExternalEtcdInitialized = true
	conditions.MarkTrue(cluster, clusterv1.ManagedExternalEtcdClusterInitializedCondition)
	etcdInitSecret := newEtcdInitSecret(cluster)

	machine := newMachine(cluster, "machine")
	config := newEtcdadmConfig(machine, "etcdadmConfig", etcdbootstrapv1.CloudConfig)

	etcdCACerts := etcdCACertKeyPair()
	etcdCACerts.Generate()
	etcdCASecret := etcdCACerts[0].AsSecret(client.ObjectKey{Namespace: cluster.Namespace, Name: cluster.Name}, *metav1.NewControllerRef(config, etcdbootstrapv1.GroupVersion.WithKind("EtcdadmConfig")))

	objects := []client.Object{
		cluster,
		machine,
		etcdInitSecret,
		etcdCASecret,
		config,
	}
	myclient := fake.NewClientBuilder().WithScheme(setupScheme()).WithObjects(objects...).Build()

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
	result, err := k.Reconcile(ctx, request)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(result.Requeue).To(BeFalse())
	g.Expect(result.RequeueAfter).To(BeZero())

	configKey := client.ObjectKeyFromObject(config)
	g.Expect(myclient.Get(context.TODO(), configKey, config)).To(Succeed())
	c := conditions.Get(config, etcdbootstrapv1.DataSecretAvailableCondition)
	g.Expect(c).ToNot(BeNil())
	g.Expect(c.Status).To(Equal(corev1.ConditionTrue))

	bootstrapSecret := &corev1.Secret{}
	err = myclient.Get(ctx, configKey, bootstrapSecret)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(bootstrapSecret.Data).To(Not(BeNil()))
	joinData := string(bootstrapSecret.Data["value"])
	g.Expect(joinData).To(ContainSubstring("etcdadm join https://1.2.3.4:2379 --init-system systemd"))
}

func TestEtcdadmConfigReconciler_JoinMemberIfEtcdIsInitialized_Bottlerocket(t *testing.T) {
	g := NewWithT(t)

	cluster := newCluster("external-etcd-cluster")
	cluster.Status.ManagedExternalEtcdInitialized = true
	conditions.MarkTrue(cluster, clusterv1.ManagedExternalEtcdClusterInitializedCondition)
	etcdInitSecret := newEtcdInitSecret(cluster)

	machine := newMachine(cluster, "machine")
	config := newEtcdadmConfig(machine, "etcdadmConfig", etcdbootstrapv1.Bottlerocket)

	etcdCACerts := etcdCACertKeyPair()
	etcdCACerts.Generate()
	etcdCASecret := etcdCACerts[0].AsSecret(client.ObjectKey{Namespace: cluster.Namespace, Name: cluster.Name}, *metav1.NewControllerRef(config, etcdbootstrapv1.GroupVersion.WithKind("EtcdadmConfig")))

	objects := []client.Object{
		cluster,
		machine,
		etcdInitSecret,
		etcdCASecret,
		config,
	}
	myclient := fake.NewClientBuilder().WithScheme(setupScheme()).WithObjects(objects...).Build()

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
	result, err := k.Reconcile(ctx, request)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(result.Requeue).To(BeFalse())
	g.Expect(result.RequeueAfter).To(BeZero())

	configKey := client.ObjectKeyFromObject(config)
	g.Expect(myclient.Get(context.TODO(), configKey, config)).To(Succeed())
	c := conditions.Get(config, etcdbootstrapv1.DataSecretAvailableCondition)
	g.Expect(c).ToNot(BeNil())
	g.Expect(c.Status).To(Equal(corev1.ConditionTrue))

	bootstrapSecret := &corev1.Secret{}
	err = myclient.Get(ctx, configKey, bootstrapSecret)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(bootstrapSecret.Data).To(Not(BeNil()))
}

// Older versions of CAPI fork create etcd secret containing only IP address of the first machine
func TestEtcdadmConfigReconciler_JoinMemberIfEtcdIsInitialized_EtcdInitSecretOlderFormat(t *testing.T) {
	g := NewWithT(t)

	cluster := newCluster("external-etcd-cluster")
	cluster.Status.ManagedExternalEtcdInitialized = true
	conditions.MarkTrue(cluster, clusterv1.ManagedExternalEtcdClusterInitializedCondition)
	etcdInitSecret := newEtcdInitSecret(cluster)
	etcdInitSecret.Data = map[string][]byte{"address": []byte("1.2.3.4")}
	machine := newMachine(cluster, "machine")
	config := newEtcdadmConfig(machine, "etcdadmConfig", etcdbootstrapv1.CloudConfig)

	etcdCACerts := etcdCACertKeyPair()
	err := etcdCACerts.Generate()
	g.Expect(err).NotTo(HaveOccurred())
	etcdCASecret := etcdCACerts[0].AsSecret(client.ObjectKey{Namespace: cluster.Namespace, Name: cluster.Name}, *metav1.NewControllerRef(config, etcdbootstrapv1.GroupVersion.WithKind("EtcdadmConfig")))

	objects := []client.Object{
		cluster,
		machine,
		etcdInitSecret,
		etcdCASecret,
		config,
	}
	myclient := fake.NewClientBuilder().WithScheme(setupScheme()).WithObjects(objects...).Build()

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
	result, err := k.Reconcile(ctx, request)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(result.Requeue).To(BeFalse())
	g.Expect(result.RequeueAfter).To(BeZero())

	configKey := client.ObjectKeyFromObject(config)
	g.Expect(myclient.Get(context.TODO(), configKey, config)).To(Succeed())
	c := conditions.Get(config, etcdbootstrapv1.DataSecretAvailableCondition)
	g.Expect(c).ToNot(BeNil())
	g.Expect(c.Status).To(Equal(corev1.ConditionTrue))

	bootstrapSecret := &corev1.Secret{}
	err = myclient.Get(ctx, configKey, bootstrapSecret)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(bootstrapSecret.Data).To(Not(BeNil()))
	joinData := string(bootstrapSecret.Data["value"])
	g.Expect(joinData).To(ContainSubstring("etcdadm join https://1.2.3.4:2379 --init-system systemd"))
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
					APIVersion: etcdbootstrapv1.GroupVersion.String(),
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
func newEtcdadmConfig(machine *clusterv1.Machine, name string, format etcdbootstrapv1.Format) *etcdbootstrapv1.EtcdadmConfig {
	config := &etcdbootstrapv1.EtcdadmConfig{
		TypeMeta: metav1.TypeMeta{
			Kind:       "EtcdadmConfig",
			APIVersion: etcdbootstrapv1.GroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: "default",
		},
		Spec: etcdbootstrapv1.EtcdadmConfigSpec{
			Format:          format,
			CloudInitConfig: &etcdbootstrapv1.CloudInitConfig{},
			EtcdadmBuiltin:  true,
		},
	}

	switch format {
	case etcdbootstrapv1.Bottlerocket:
		config.Spec.BottlerocketConfig = &etcdbootstrapv1.BottlerocketConfig{}
	default:
		config.Spec.CloudInitConfig = &etcdbootstrapv1.CloudInitConfig{}
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

func newEtcdInitSecret(cluster *clusterv1.Cluster) *corev1.Secret {
	return &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: cluster.Namespace,
			Name:      fmt.Sprintf("%v-%v", cluster.Name, "etcd-init"),
		},
		Data: map[string][]byte{
			"clientUrls": []byte("https://1.2.3.4:2379"),
		},
	}
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
