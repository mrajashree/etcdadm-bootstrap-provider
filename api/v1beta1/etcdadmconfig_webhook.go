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

package v1beta1

import (
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// log is for logging in this package.
var etcdadmconfiglog = logf.Log.WithName("etcdadmconfig-resource")

func (r *EtcdadmConfig) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

// +kubebuilder:webhook:path=/mutate-bootstrap-cluster-x-k8s-io-cluster-x-k8s-io-v1beta1-etcdadmconfig,mutating=true,failurePolicy=fail,groups=bootstrap.cluster.x-k8s.io.cluster.x-k8s.io,resources=etcdadmconfigs,verbs=create;update,versions=v1beta1,name=metcdadmconfig.kb.io,sideEffects=None,admissionReviewVersions=v1;v1beta1

var _ webhook.Defaulter = &EtcdadmConfig{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *EtcdadmConfig) Default() {
	etcdadmconfiglog.Info("default", "name", r.Name)

	// TODO(user): fill in your defaulting logic.
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
// +kubebuilder:webhook:verbs=create;update,path=/validate-bootstrap-cluster-x-k8s-io-cluster-x-k8s-io-v1beta1-etcdadmconfig,mutating=false,failurePolicy=fail,groups=bootstrap.cluster.x-k8s.io.cluster.x-k8s.io,resources=etcdadmconfigs,versions=v1beta1,name=vetcdadmconfig.kb.io,sideEffects=None,admissionReviewVersions=v1;v1beta1

var _ webhook.Validator = &EtcdadmConfig{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *EtcdadmConfig) ValidateCreate() error {
	etcdadmconfiglog.Info("validate create", "name", r.Name)

	// TODO(user): fill in your validation logic upon object creation.
	return nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *EtcdadmConfig) ValidateUpdate(old runtime.Object) error {
	etcdadmconfiglog.Info("validate update", "name", r.Name)

	// TODO(user): fill in your validation logic upon object update.
	return nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *EtcdadmConfig) ValidateDelete() error {
	etcdadmconfiglog.Info("validate delete", "name", r.Name)

	// TODO(user): fill in your validation logic upon object deletion.
	return nil
}
