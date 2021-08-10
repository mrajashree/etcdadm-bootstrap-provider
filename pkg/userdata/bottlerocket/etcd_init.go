package bottlerocket

import (
	"fmt"

	"github.com/go-logr/logr"
	"github.com/mrajashree/etcdadm-bootstrap-provider/pkg/userdata"
)

const etcdInitCloudInit = `{{.Header}}
{{template "files" .WriteFiles}}
-   path: /run/cluster-api/placeholder
    owner: root:root
    permissions: '0640'
    content: "This placeholder file is used to create the /run/cluster-api sub directory in a way that is compatible with both Linux and Windows (mkdir -p /run/cluster-api does not work with Windows)"
runcmd: " {{ .EtcdadmInitCommand }}"
`

// NewInitEtcdPlane returns the user data string to be used on a etcd instance.
func NewInitEtcdPlane(input *userdata.EtcdPlaneInput, log logr.Logger) ([]byte, error) {
	input.WriteFiles = input.Certificates.AsFiles()
	prepare(&input.BaseUserData)
	logIgnoredFields(&input.BaseUserData, log)
	input.EtcdadmInitCommand = fmt.Sprintf("EtcdadmInit %s %s", input.ImageRepository, input.Version)
	userData, err := generateUserData("InitEtcdplane", etcdInitCloudInit, input, &input.BaseUserData)
	if err != nil {
		return nil, err
	}

	return userData, nil
}