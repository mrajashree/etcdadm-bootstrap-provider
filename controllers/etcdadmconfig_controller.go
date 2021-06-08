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
	"github.com/go-logr/logr"
	"github.com/mrajashree/etcdadm-bootstrap-provider/cloudinit"
	"github.com/mrajashree/etcdadm-bootstrap-provider/internal/locking"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/pointer"
	"path/filepath"
	bootstrapv1 "sigs.k8s.io/cluster-api/bootstrap/kubeadm/api/v1alpha4"
	"sigs.k8s.io/cluster-api/util"
	"sigs.k8s.io/cluster-api/util/conditions"
	"sigs.k8s.io/cluster-api/util/patch"
	"sigs.k8s.io/cluster-api/util/secret"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"time"

	bootstrapv1alpha4 "github.com/mrajashree/etcdadm-bootstrap-provider/api/v1alpha4"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1alpha4"
)

// InitLocker is a lock that is used around kubeadm init
type InitLocker interface {
	Lock(ctx context.Context, cluster *clusterv1.Cluster, machine *clusterv1.Machine) bool
	Unlock(ctx context.Context, cluster *clusterv1.Cluster) bool
}

var installEtcdadmCommands = []string{`curl -O -L https://github.com/mrajashree/etcdadm-bootstrap-provider/releases/download/v0.0.0/etcdadm`,
	`chmod +x etcdadm`, `mv etcdadm /usr/local/bin/etcdadm`}

// EtcdadmConfigReconciler reconciles a EtcdadmConfig object
type EtcdadmConfigReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme

	EtcdadmInitLock InitLocker
}

type Scope struct {
	logr.Logger
	Config *bootstrapv1alpha4.EtcdadmConfig
	//ConfigOwner *bsutil.ConfigOwner
	Cluster *clusterv1.Cluster
	Machine *clusterv1.Machine
}

// +kubebuilder:rbac:groups=bootstrap.cluster.x-k8s.io,resources=etcdadmconfigs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=bootstrap.cluster.x-k8s.io,resources=etcdadmconfigs/status,verbs=get;update;patch

func (r *EtcdadmConfigReconciler) Reconcile(ctx context.Context, req ctrl.Request) (_ ctrl.Result, rerr error) {
	log := ctrl.LoggerFrom(ctx)

	// Lookup the etcdadm config
	config := &bootstrapv1alpha4.EtcdadmConfig{}
	if err := r.Client.Get(ctx, req.NamespacedName, config); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		log.Error(err, "Failed to get config")
		return ctrl.Result{}, err
	}

	// Look up the Machine that owns this KubeConfig if there is one
	machine, err := util.GetOwnerMachine(ctx, r.Client, config.ObjectMeta)
	if err != nil {
		log.Error(err, "could not get owner machine")
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
			log.Info("Cluster does not exist yet , waiting until it is created")
			return ctrl.Result{}, nil
		}
		log.Error(err, "could not get cluster by machine metadata")
		return ctrl.Result{}, err
	}

	// Initialize the patch helper.
	patchHelper, err := patch.NewHelper(config, r.Client)
	if err != nil {
		return ctrl.Result{}, err
	}

	// Attempt to Patch the KubeadmConfig object and status after each reconciliation if no error occurs.
	defer func() {
		// always update the readyCondition; the summary is represented using the "1 of x completed" notation.
		conditions.SetSummary(config,
			conditions.WithConditions(
				bootstrapv1alpha4.DataSecretAvailableCondition,
			),
		)
		// Patch ObservedGeneration only if the reconciliation completed successfully
		patchOpts := []patch.Option{}
		if rerr == nil {
			patchOpts = append(patchOpts, patch.WithStatusObservedGeneration{})
		}
		if err := patchHelper.Patch(ctx, config, patchOpts...); err != nil {
			log.Error(rerr, "Failed to patch config")
			log.Info(fmt.Sprintf("error from patch call: %v", err))
			if rerr == nil {
				rerr = err
			}
		}
	}()

	scope := Scope{
		Logger:  log,
		Config:  config,
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

func (r *EtcdadmConfigReconciler) SetupWithManager(ctx context.Context, mgr ctrl.Manager) error {
	if r.EtcdadmInitLock == nil {
		r.EtcdadmInitLock = locking.NewEtcdadmInitMutex(ctrl.LoggerFrom(ctx).WithName("etcd-init-locker"), mgr.GetClient())
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&bootstrapv1alpha4.EtcdadmConfig{}).
		Complete(r)
}

func (r *EtcdadmConfigReconciler) initializeEtcd(ctx context.Context, scope *Scope) (_ ctrl.Result, rerr error) {
	log := ctrl.LoggerFrom(ctx)
	// acquire the init lock so that only the first machine configured
	// as etcd node gets processed here
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
	log.Info("Creating BootstrapData for the init etcd plane")

	CACertKeyPair := etcdCACertKeyPair()
	rerr = CACertKeyPair.LookupOrGenerate(
		ctx,
		r.Client,
		util.ObjectKey(scope.Cluster),
		*metav1.NewControllerRef(scope.Config, bootstrapv1alpha4.GroupVersion.WithKind("EtcdadmConfig")),
		)

	cloudInitData, err := cloudinit.NewInitEtcdPlane(&cloudinit.EtcdPlaneInput{
		BaseUserData: cloudinit.BaseUserData{
			PreEtcdadmCommands: append(scope.Config.Spec.PreEtcdadmCommands,installEtcdadmCommands...),
			Users: scope.Config.Spec.Users,
		},
		Certificates: CACertKeyPair,
	})

	if err != nil {
		log.Error(err, "Failed to generate cloud init for bootstrap etcd plane")
		return ctrl.Result{}, err
	}
	if err := r.storeBootstrapData(ctx, scope.Config, cloudInitData, scope.Cluster.Name); err != nil {
		log.Error(err, "Failed to store bootstrap data")
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

func (r *EtcdadmConfigReconciler) joinEtcd(ctx context.Context, scope *Scope) (_ ctrl.Result, rerr error) {
	log := ctrl.LoggerFrom(ctx)
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
	rerr = etcdCerts.Lookup(
		ctx,
		r.Client,
		util.ObjectKey(scope.Cluster),
	)

	initMachineAddress := string(existingSecret.Data["address"])
	joinAddress := fmt.Sprintf("https://%v:2379", initMachineAddress)

	cloudInitData, err := cloudinit.NewJoinEtcdPlane(&cloudinit.EtcdPlaneJoinInput{
		BaseUserData: cloudinit.BaseUserData{
			PreEtcdadmCommands: append(scope.Config.Spec.PreEtcdadmCommands,installEtcdadmCommands...),
			Users: scope.Config.Spec.Users,
			JoinAddress: joinAddress,
		},
		Certificates: etcdCerts,
	})
	if err != nil {
		log.Error(err, "Failed to generate cloud init for bootstrap etcd plane - join")
		return ctrl.Result{}, err
	}
	if err := r.storeBootstrapData(ctx, scope.Config, cloudInitData, scope.Cluster.Name); err != nil {
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
func (r *EtcdadmConfigReconciler) storeBootstrapData(ctx context.Context, config *bootstrapv1alpha4.EtcdadmConfig, data []byte, clusterName string) error {
	log := ctrl.LoggerFrom(ctx)

	se := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      config.Name,
			Namespace: config.Namespace,
			Labels: map[string]string{
				clusterv1.ClusterLabelName: clusterName,
			},
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: bootstrapv1alpha4.GroupVersion.String(),
					Kind:       "EtcdadmConfig",
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
