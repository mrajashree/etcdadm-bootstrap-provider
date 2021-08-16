package bottlerocket

import (
	"fmt"

	"github.com/go-logr/logr"
	bootstrapv1alpha3 "github.com/mrajashree/etcdadm-bootstrap-provider/api/v1alpha3"
	"github.com/mrajashree/etcdadm-bootstrap-provider/pkg/userdata"
)

const etcdInitCloudInit = `{{.Header}}
{{template "files" .WriteFiles}}
-   path: /run/cluster-api/placeholder
    owner: root:root
    permissions: '0640'
    content: "This placeholder file is used to create the /run/cluster-api sub directory in a way that is compatible with both Linux and Windows (mkdir -p /run/cluster-api does not work with Windows)"
runcmd: "{{ .EtcdadmInitCommand }}"
`

// NewInitEtcdPlane returns the user data string to be used on a etcd instance.
func NewInitEtcdPlane(input *userdata.EtcdPlaneInput, config bootstrapv1alpha3.BottlerocketConfig, log logr.Logger) ([]byte, error) {
	input.WriteFiles = input.Certificates.AsFiles()
	prepare(&input.BaseUserData)
	input.EtcdadmArgs = buildEtcdadmArgs(config)
	logIgnoredFields(&input.BaseUserData, log)
	input.EtcdadmInitCommand = fmt.Sprintf("EtcdadmInit %s %s", input.ImageRepository, input.Version)
	userData, err := generateUserData("InitEtcdplane", etcdInitCloudInit, input, &input.BaseUserData, config)
	if err != nil {
		return nil, err
	}

	return userData, nil
}
