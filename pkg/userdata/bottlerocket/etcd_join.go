package bottlerocket

import (
	"fmt"

	"github.com/go-logr/logr"
	etcdbootstrapv1 "github.com/mrajashree/etcdadm-bootstrap-provider/api/v1beta1"
	"github.com/mrajashree/etcdadm-bootstrap-provider/pkg/userdata"
	"github.com/pkg/errors"
)

const (
	etcdPlaneJoinCloudInit = `{{.Header}}
{{template "files" .WriteFiles}}
-   path: /run/cluster-api/placeholder
    owner: root:root
    permissions: '0640'
    content: "This placeholder file is used to create the /run/cluster-api sub directory in a way that is compatible with both Linux and Windows (mkdir -p /run/cluster-api does not work with Windows)"
runcmd: "{{ .EtcdadmJoinCommand }}"
`
)

// NewJoinControlPlane returns the user data string to be used on a new control plane instance.
func NewJoinEtcdPlane(input *userdata.EtcdPlaneJoinInput, config etcdbootstrapv1.EtcdadmConfigSpec, log logr.Logger) ([]byte, error) {
	input.WriteFiles = input.Certificates.AsFiles()
	prepare(&input.BaseUserData)
	input.EtcdadmArgs = buildEtcdadmArgs(config)
	logIgnoredFields(&input.BaseUserData, log)
	input.ControlPlane = true
	input.EtcdadmJoinCommand = fmt.Sprintf("EtcdadmJoin %s %s %s %s", input.ImageRepository, input.Version, input.CipherSuites, input.JoinAddress)
	userData, err := generateUserData("JoinControlplane", etcdPlaneJoinCloudInit, input, &input.BaseUserData, config, log)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to generate user data for machine joining control plane")
	}

	return userData, err
}
