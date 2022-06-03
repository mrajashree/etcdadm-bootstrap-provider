# No longer maintained

## Bootstrap provider for creating an etcd cluster using etcdadm

### Using this with CAPI+CAPD:
1. There are some changes required within CAPI and CAPD for the provisioning of external etcd clusters to work. Checkout this fork+branch locally:
```
https://github.com/mrajashree/cluster-api/tree/etcdadm_bootstrap
```
2. Modify cluster-api/tilt-settings.json to add this provider:
```json
{
  "default_registry": "",
  "provider_repos": ["../../mrajashree/etcdadm-bootstrap-provider"],
  "enable_providers": ["core","docker", "kubeadm-bootstrap", "kubeadm-control-plane", "etcdadm-bootstrap"],
  "kustomize_substitutions": {
    "ETCDADM_BOOTSTRAP_IMAGE": "mrajashree/etcdadm-bootstrap-provider:latest"
  }
}

```
3. This provider has a tilt-provider.json that will be used by CAPI
4. Create a Kind cluster, and run tilt up

### Using this with CAPI+CAPA
1. There are some security group changes required for CAPA, I made them based off of the PR that adds v1alpha4 to CAPA. Checkout the changes locally from this fork+branch:
```
https://github.com/mrajashree/cluster-api-provider-aws/tree/etcadm
```
2. Modify cluster-api/tilt-settings.json to add this provider:
```json
{
  "default_registry": "",
  "provider_repos": ["../cluster-api-provider-aws"],
  "enable_providers": ["core","docker", "aws", "kubeadm-bootstrap", "kubeadm-control-plane", "etcdadm-bootstrap"],
  "kustomize_substitutions": {
    "AWS_B64ENCODED_CREDENTIALS": "...",
    "ETCDADM_BOOTSTRAP_IMAGE": "mrajashree/etcdadm-bootstrap-provider:latest"
  }
}
```
3. Create a Kind cluster, and run tilt up
