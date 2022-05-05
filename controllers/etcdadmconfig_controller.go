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
	"path/filepath"
	"time"

	"github.com/go-logr/logr"
	etcdbootstrapv1 "github.com/mrajashree/etcdadm-bootstrap-provider/api/v1beta1"
	"github.com/mrajashree/etcdadm-bootstrap-provider/internal/locking"
	"github.com/mrajashree/etcdadm-bootstrap-provider/pkg/userdata"
	"github.com/mrajashree/etcdadm-bootstrap-provider/pkg/userdata/bottlerocket"
	"github.com/mrajashree/etcdadm-bootstrap-provider/pkg/userdata/cloudinit"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/pointer"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	bootstrapv1 "sigs.k8s.io/cluster-api/bootstrap/kubeadm/api/v1beta1"
	"sigs.k8s.io/cluster-api/util"
	"sigs.k8s.io/cluster-api/util/annotations"
	"sigs.k8s.io/cluster-api/util/conditions"
	"sigs.k8s.io/cluster-api/util/patch"
	"sigs.k8s.io/cluster-api/util/predicates"
	"sigs.k8s.io/cluster-api/util/secret"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const stopKubeletCommand = "systemctl stop kubelet"

// InitLocker is a lock that is used around etcdadm init
type InitLocker interface {
	Lock(ctx context.Context, cluster *clusterv1.Cluster, machine *clusterv1.Machine) bool
	Unlock(ctx context.Context, cluster *clusterv1.Cluster) bool
}

// TODO: replace with etcdadm release
var defaultEtcdadmInstallCommands = []string{`curl -OL https://github.com/mrajashree/etcdadm-bootstrap-provider/releases/download/v0.0.0/etcdadm`, `chmod +x etcdadm`, `mv etcdadm /usr/local/bin/etcdadm`}

// EtcdadmConfigReconciler reconciles a EtcdadmConfig object
type EtcdadmConfigReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme

	EtcdadmInitLock InitLocker
}

type Scope struct {
	logr.Logger
	Config  *etcdbootstrapv1.EtcdadmConfig
	Cluster *clusterv1.Cluster
	Machine *clusterv1.Machine
}

func (r *EtcdadmConfigReconciler) SetupWithManager(ctx context.Context, mgr ctrl.Manager) error {
	if r.EtcdadmInitLock == nil {
		r.EtcdadmInitLock = locking.NewEtcdadmInitMutex(ctrl.LoggerFrom(ctx).WithName("etcd-init-locker"), mgr.GetClient())
	}

	b := ctrl.NewControllerManagedBy(mgr).
		For(&etcdbootstrapv1.EtcdadmConfig{}).
		WithEventFilter(predicates.ResourceNotPaused(r.Log)).
		Watches(
			&source.Kind{Type: &clusterv1.Machine{}},
			handler.EnqueueRequestsFromMapFunc(r.MachineToBootstrapMapFunc),
		)

	c, err := b.Build(r)
	if err != nil {
		return errors.Wrap(err, "failed setting up with a controller manager")
	}

	err = c.Watch(
		&source.Kind{Type: &clusterv1.Cluster{}},
		handler.EnqueueRequestsFromMapFunc(r.ClusterToEtcdadmConfigs),
		predicates.ClusterUnpausedAndInfrastructureReady(r.Log),
	)
	if err != nil {
		return errors.Wrap(err, "failed adding Watch for Clusters to controller manager")
	}
	return nil
}

// +kubebuilder:rbac:groups=bootstrap.cluster.x-k8s.io,resources=etcdadmconfigs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=bootstrap.cluster.x-k8s.io,resources=etcdadmconfigs/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=cluster.x-k8s.io,resources=clusters;clusters/status;machines;machines/status,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=configmaps;events;secrets,verbs=get;list;watch;create;update;patch;delete

func (r *EtcdadmConfigReconciler) Reconcile(ctx context.Context, req ctrl.Request) (_ ctrl.Result, rerr error) {
	log := r.Log.WithValues("etcdadmconfig", req.Name, "namespace", req.Namespace)

	// Lookup the etcdadm config
	etcdadmConfig := &etcdbootstrapv1.EtcdadmConfig{}
	if err := r.Client.Get(ctx, req.NamespacedName, etcdadmConfig); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		log.Error(err, "Failed to get etcdadm config")
		return ctrl.Result{}, err
	}

	// Look up the Machine associated with this EtcdadmConfig resource
	machine, err := util.GetOwnerMachine(ctx, r.Client, etcdadmConfig.ObjectMeta)
	if err != nil {
		if apierrors.IsNotFound(err) {
			// could not find owning machine, reconcile when owner is set
			return ctrl.Result{}, nil
		}
		log.Error(err, "could not get owner machine for the EtcdadmConfig")
		return ctrl.Result{}, err
	}
	if machine == nil {
		log.Info("Waiting for Machine Controller to set OwnerRef on the EtcdadmConfig")
		return ctrl.Result{}, nil
	}
	log = log.WithValues("machine-name", machine.Name)
	// Lookup the cluster the machine is associated with
	cluster, err := util.GetClusterFromMetadata(ctx, r.Client, machine.ObjectMeta)
	if err != nil {
		if errors.Cause(err) == util.ErrNoCluster {
			log.Info("Machine does not belong to a cluster yet, waiting until its part of a cluster")
			return ctrl.Result{}, nil
		}
		if apierrors.IsNotFound(err) {
			log.Info("Cluster does not exist yet, waiting until it is created")
			return ctrl.Result{}, nil
		}
		log.Error(err, "could not get cluster by machine metadata")
		return ctrl.Result{}, err
	}

	if annotations.IsPaused(cluster, etcdadmConfig) {
		log.Info("Reconciliation is paused for this object")
		return ctrl.Result{}, nil
	}
	// Initialize the patch helper.
	patchHelper, err := patch.NewHelper(etcdadmConfig, r.Client)
	if err != nil {
		return ctrl.Result{}, err
	}

	// Attempt to Patch the EtcdadmConfig object and status after each reconciliation if no error occurs.
	defer func() {
		// always update the readyCondition; the summary is represented using the "1 of x completed" notation.
		conditions.SetSummary(etcdadmConfig,
			conditions.WithConditions(
				etcdbootstrapv1.DataSecretAvailableCondition,
			),
		)
		// Patch ObservedGeneration only if the reconciliation completed successfully
		patchOpts := []patch.Option{}
		if rerr == nil {
			patchOpts = append(patchOpts, patch.WithStatusObservedGeneration{})
		}
		if err := patchHelper.Patch(ctx, etcdadmConfig, patchOpts...); err != nil {
			log.Error(err, "Failed to patch etcdadmConfig")
			if rerr == nil {
				rerr = err
			}
		}
	}()

	if etcdadmConfig.Status.Ready {
		return ctrl.Result{}, nil
	}

	scope := Scope{
		Logger:  log,
		Config:  etcdadmConfig,
		Cluster: cluster,
		Machine: machine,
	}

	if !conditions.IsTrue(cluster, clusterv1.ManagedExternalEtcdClusterInitializedCondition) {
		return r.initializeEtcd(ctx, &scope)
	}
	// Unlock any locks that might have been set during init process
	r.EtcdadmInitLock.Unlock(ctx, cluster)

	res, err := r.joinEtcd(ctx, &scope)
	if err != nil {
		return res, err
	}

	return ctrl.Result{}, nil
}

func (r *EtcdadmConfigReconciler) initializeEtcd(ctx context.Context, scope *Scope) (_ ctrl.Result, rerr error) {
	log := r.Log
	// acquire the init lock so that only the first machine configured as etcd node gets processed here
	// if not the first, requeue
	if !r.EtcdadmInitLock.Lock(ctx, scope.Cluster, scope.Machine) {
		log.Info("An etcd node is already being initialized, requeing until etcd plane is ready")
		return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
	}

	defer func() {
		if rerr != nil {
			r.EtcdadmInitLock.Unlock(ctx, scope.Cluster)
		}
	}()
	log.Info("Creating cloudinit for the init etcd plane")

	CACertKeyPair := etcdCACertKeyPair()
	rerr = CACertKeyPair.LookupOrGenerate(
		ctx,
		r.Client,
		util.ObjectKey(scope.Cluster),
		*metav1.NewControllerRef(scope.Config, etcdbootstrapv1.GroupVersion.WithKind("EtcdadmConfig")),
	)

	initInput := userdata.EtcdPlaneInput{
		BaseUserData: userdata.BaseUserData{
			Users: scope.Config.Spec.Users,
		},
		Certificates: CACertKeyPair,
	}

	// only do this if etcdadm not baked in image
	if !scope.Config.Spec.EtcdadmBuiltin {
		if len(scope.Config.Spec.EtcdadmInstallCommands) > 0 {
			initInput.PreEtcdadmCommands = append(scope.Config.Spec.PreEtcdadmCommands, scope.Config.Spec.EtcdadmInstallCommands...)
		} else {
			initInput.PreEtcdadmCommands = append(scope.Config.Spec.PreEtcdadmCommands, defaultEtcdadmInstallCommands...)
		}
	}

	var bootstrapData []byte
	var err error

	switch scope.Config.Spec.Format {
	case etcdbootstrapv1.Bottlerocket:
		bootstrapData, err = bottlerocket.NewInitEtcdPlane(&initInput, scope.Config.Spec, log)
	default:
		initInput.PreEtcdadmCommands = append(initInput.PreEtcdadmCommands, stopKubeletCommand)
		bootstrapData, err = cloudinit.NewInitEtcdPlane(&initInput, scope.Config.Spec)
	}
	if err != nil {
		log.Error(err, "Failed to generate cloud init for initializing etcd plane")
		return ctrl.Result{}, err
	}

	if err := r.storeBootstrapData(ctx, scope.Config, bootstrapData, scope.Cluster.Name); err != nil {
		log.Error(err, "Failed to store bootstrap data")
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

func (r *EtcdadmConfigReconciler) joinEtcd(ctx context.Context, scope *Scope) (_ ctrl.Result, rerr error) {
	log := r.Log
	etcdSecretName := fmt.Sprintf("%v-%v", scope.Cluster.Name, "etcd-init")
	existingSecret := &corev1.Secret{}
	if err := r.Client.Get(ctx, client.ObjectKey{Namespace: scope.Cluster.Namespace, Name: etcdSecretName}, existingSecret); err != nil {
		if apierrors.IsNotFound(err) {
			// this is not an error, just means the first machine didn't get an address yet, reconcile
			log.Info("Waiting for Machine Controller to set address on init machine and returning error")
			return ctrl.Result{}, err
		}
		log.Error(err, "Failed to get secret containing first machine address")
		return ctrl.Result{}, err
	}
	log.Info("Machine Controller has set address on init machine")

	etcdCerts := etcdCACertKeyPair()
	if err := etcdCerts.Lookup(
		ctx,
		r.Client,
		util.ObjectKey(scope.Cluster),
	); err != nil {
		return ctrl.Result{}, errors.Wrap(err, "failed doing a lookup for certs during join")
	}

	var joinAddress string
	if clientURL, ok := existingSecret.Data["clientUrls"]; ok {
		joinAddress = string(clientURL)
	} else {
		initMachineAddress := string(existingSecret.Data["address"])
		joinAddress = fmt.Sprintf("https://%v:2379", initMachineAddress)
	}

	joinInput := userdata.EtcdPlaneJoinInput{
		BaseUserData: userdata.BaseUserData{
			Users: scope.Config.Spec.Users,
		},
		JoinAddress:  joinAddress,
		Certificates: etcdCerts,
	}

	if !scope.Config.Spec.EtcdadmBuiltin {
		if len(scope.Config.Spec.EtcdadmInstallCommands) > 0 {
			joinInput.PreEtcdadmCommands = append(scope.Config.Spec.PreEtcdadmCommands, scope.Config.Spec.EtcdadmInstallCommands...)
		} else {
			joinInput.PreEtcdadmCommands = append(scope.Config.Spec.PreEtcdadmCommands, defaultEtcdadmInstallCommands...)
		}
	}

	var bootstrapData []byte
	var err error

	switch scope.Config.Spec.Format {
	case etcdbootstrapv1.Bottlerocket:
		bootstrapData, err = bottlerocket.NewJoinEtcdPlane(&joinInput, scope.Config.Spec, log)
	default:
		joinInput.PreEtcdadmCommands = append(joinInput.PreEtcdadmCommands, stopKubeletCommand)
		bootstrapData, err = cloudinit.NewJoinEtcdPlane(&joinInput, scope.Config.Spec)
	}
	if err != nil {
		log.Error(err, "Failed to generate cloud init for bootstrap etcd plane - join")
		return ctrl.Result{}, err
	}

	if err := r.storeBootstrapData(ctx, scope.Config, bootstrapData, scope.Cluster.Name); err != nil {
		log.Error(err, "Failed to store bootstrap data - join")
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

func etcdCACertKeyPair() secret.Certificates {
	certificatesDir := "/etc/etcd/pki"
	certificates := secret.Certificates{
		&secret.Certificate{
			Purpose:  secret.ManagedExternalEtcdCA,
			CertFile: filepath.Join(certificatesDir, "ca.crt"),
			KeyFile:  filepath.Join(certificatesDir, "ca.key"),
		},
	}

	return certificates
}

// storeBootstrapData creates a new secret with the data passed in as input,
// sets the reference in the configuration status and ready to true.
func (r *EtcdadmConfigReconciler) storeBootstrapData(ctx context.Context, config *etcdbootstrapv1.EtcdadmConfig, data []byte, clusterName string) error {
	log := r.Log

	se := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      config.Name,
			Namespace: config.Namespace,
			Labels: map[string]string{
				clusterv1.ClusterLabelName: clusterName,
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
			"value": data,
		},
		Type: clusterv1.ClusterSecretType,
	}

	// as secret creation and scope.Config status patch are not atomic operations
	// it is possible that secret creation happens but the config.Status patches are not applied
	if err := r.Client.Create(ctx, se); err != nil {
		if !apierrors.IsAlreadyExists(err) {
			return errors.Wrapf(err, "failed to create bootstrap data secret for EtcdadmConfig %s/%s", config.Namespace, config.Name)
		}
		log.Info("bootstrap data secret for EtcdadmConfig already exists, updating", "secret", secret.Name, "EtcdadmConfig", config.Name)
		if err := r.Client.Update(ctx, se); err != nil {
			return errors.Wrapf(err, "failed to update bootstrap data secret for EtcdadmConfig %s/%s", config.Namespace, config.Name)
		}
	}
	config.Status.DataSecretName = pointer.StringPtr(se.Name)
	config.Status.Ready = true
	conditions.MarkTrue(config, bootstrapv1.DataSecretAvailableCondition)
	return nil
}
