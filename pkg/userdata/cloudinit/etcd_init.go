/*
Copyright 2019 The Kubernetes Authors.

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

package cloudinit

import (
	bootstrapv1alpha3 "github.com/mrajashree/etcdadm-bootstrap-provider/api/v1alpha3"
	"github.com/mrajashree/etcdadm-bootstrap-provider/pkg/userdata"
	"github.com/pkg/errors"
)

const (
	etcdPlaneCloudInit = `{{.Header}}
{{template "files" .WriteFiles}}
-   path: /run/cluster-api/placeholder
    owner: root:root
    permissions: '0640'
    content: "This placeholder file is used to create the /run/cluster-api sub directory in a way that is compatible with both Linux and Windows (mkdir -p /run/cluster-api does not work with Windows)"
runcmd:
{{- template "commands" .PreEtcdadmCommands }}
  - {{ .EtcdadmInitCommand }} && {{ .SentinelFileCommand }}
{{- template "commands" .PostEtcdadmCommands }}
{{- template "ntp" .NTP }}
{{- template "users" .Users }}
{{- template "disk_setup" .DiskSetup}}
{{- template "fs_setup" .DiskSetup}}
{{- template "mounts" .Mounts}}
`
)

// NewInitEtcdPlane returns the user data string to be used on a etcd instance.
func NewInitEtcdPlane(input *userdata.EtcdPlaneInput, config bootstrapv1alpha3.EtcdadmConfigSpec) ([]byte, error) {
	input.WriteFiles = input.Certificates.AsFiles()
	input.EtcdadmArgs = buildEtcdadmArgs(*config.CloudInitConfig)
	input.EtcdadmInitCommand = userdata.AddSystemdArgsToCommand(standardInitCommand, &input.EtcdadmArgs)
	if err := setProxy(config.Proxy, &input.BaseUserData); err != nil {
		return nil, err
	}
	if err := setRegistryMirror(config.RegistryMirror, &input.BaseUserData); err != nil {
		return nil, err
	}
	if err := prepare(&input.BaseUserData); err != nil {
		return nil, err
	}
	userData, err := generate("InitEtcdCluster", etcdPlaneCloudInit, input)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to generate user data for machine initializing etcd cluster")
	}

	return userData, nil
}
