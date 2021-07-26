module github.com/mrajashree/etcdadm-bootstrap-provider

go 1.13

require (
	github.com/go-logr/logr v0.1.0
	github.com/onsi/gomega v1.10.1
	github.com/pkg/errors v0.9.1
	k8s.io/api v0.17.9
	k8s.io/apimachinery v0.17.9
	k8s.io/client-go v0.17.9
	k8s.io/utils v0.0.0-20200619165400-6e3d28b6ed19
	sigs.k8s.io/cluster-api v0.3.11-0.20210329151847-96ab9172b7c1
	sigs.k8s.io/controller-runtime v0.5.14
)

replace sigs.k8s.io/cluster-api => github.com/mrajashree/cluster-api v0.3.15-0.20210725060504-c2b0d0baee44
