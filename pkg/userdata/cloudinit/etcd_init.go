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

import "github.com/mrajashree/etcdadm-bootstrap-provider/pkg/userdata"

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
func NewInitEtcdPlane(input *userdata.EtcdPlaneInput) ([]byte, error) {
	input.Header = cloudConfigHeader
	input.WriteFiles = input.Certificates.AsFiles()
	input.WriteFiles = append(input.WriteFiles, input.AdditionalFiles...)
	input.SentinelFileCommand = sentinelFileCommand
	input.EtcdadmInitCommand = userdata.AddArgsToCommand(standardInitCommand, &input.EtcdadmArgs)
	userData, err := generate("InitEtcdplane", etcdPlaneCloudInit, input)
	if err != nil {
		return nil, err
	}

	return userData, nil
}