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
	"sigs.k8s.io/cluster-api/util/secret"
)

const (
	etcdPlaneCloudInit = `{{.Header}}
{{template "files" .WriteFiles}}
runcmd:
{{- template "commands" .PreEtcdadmCommands }}
  - 'etcdadm init'
{{- template "commands" .PostEtcdadmCommands }}
{{- template "ntp" .NTP }}
{{- template "users" .Users }}
{{- template "disk_setup" .DiskSetup}}
{{- template "fs_setup" .DiskSetup}}
{{- template "mounts" .Mounts}}
`
)

// ControlPlaneInput defines the context to generate a controlplane instance user data.
type EtcdPlaneInput struct {
	BaseUserData
	secret.Certificates

	ClusterConfiguration string
	InitConfiguration    string
}

// NewInitControlPlane returns the user data string to be used on a controlplane instance.
func NewInitEtcdPlane(input *EtcdPlaneInput) ([]byte, error) {
	input.Header = cloudConfigHeader
	input.WriteFiles = input.Certificates.AsFiles()
	input.WriteFiles = append(input.WriteFiles, input.AdditionalFiles...)
	input.SentinelFileCommand = sentinelFileCommand
	userData, err := generate("InitEtcdplane", etcdPlaneCloudInit, input)
	if err != nil {
		return nil, err
	}

	return userData, nil
}
