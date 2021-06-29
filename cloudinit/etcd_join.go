package cloudinit

import (
	"fmt"
	"github.com/pkg/errors"
	"sigs.k8s.io/cluster-api/util/secret"
)

const (
	etcdPlaneJoinCloudInit = `{{.Header}}
{{template "files" .WriteFiles}}
-   path: /run/cluster-api/placeholder
    owner: root:root
    permissions: '0640'
    content: "This placeholder file is used to create the /run/cluster-api sub directory in a way that is compatible with both Linux and Windows (mkdir -p /run/cluster-api does not work with Windows)"
runcmd:
{{- template "commands" .PreEtcdadmCommands }}
  - {{ .EtcdadmJoinCommand }} && {{ .SentinelFileCommand }}
{{- template "commands" .PostEtcdadmCommands }}
{{- template "ntp" .NTP }}
{{- template "users" .Users }}
{{- template "disk_setup" .DiskSetup}}
{{- template "fs_setup" .DiskSetup}}
{{- template "mounts" .Mounts}}
`
)

// EtcdPlaneJoinInput defines context to generate etcd instance user data for etcd plane node join.
type EtcdPlaneJoinInput struct {
	BaseUserData
	secret.Certificates
	EtcdadmArgs

	EtcdadmJoinCommand string
	JoinAddress        string
	Version            string
}

// NewJoinControlPlane returns the user data string to be used on a new control plane instance.
func NewJoinEtcdPlane(input *EtcdPlaneJoinInput) ([]byte, error) {
	input.WriteFiles = input.Certificates.AsFiles()
	input.ControlPlane = true
	input.EtcdadmJoinCommand = addEtcdadmFlags(&input.EtcdadmArgs, fmt.Sprintf(standardJoinCommand, input.JoinAddress))
	if err := input.prepare(); err != nil {
		return nil, err
	}
	userData, err := generate("JoinControlplane", etcdPlaneJoinCloudInit, input)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to generate user data for machine joining control plane")
	}

	return userData, err
}
