package cloudinit

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"

	etcdbootstrapv1 "github.com/mrajashree/etcdadm-bootstrap-provider/api/v1beta1"
	"github.com/mrajashree/etcdadm-bootstrap-provider/pkg/userdata"
	"github.com/pkg/errors"
	capbk "sigs.k8s.io/cluster-api/bootstrap/kubeadm/api/v1beta1"
)

const (
	standardInitCommand = "etcdadm init"
	standardJoinCommand = "etcdadm join %s"
	// sentinelFileCommand writes a file to /run/cluster-api to signal successful Kubernetes bootstrapping in a way that
	// works both for Linux and Windows OS.
	sentinelFileCommand = "echo success > /run/cluster-api/bootstrap-success.complete"
	cloudConfigHeader   = `## template: jinja
#cloud-config
`
	proxyConf = `
[Service]
Environment="HTTP_PROXY={{.HTTPProxy}}"
Environment="HTTPS_PROXY={{.HTTPSProxy}}"
Environment="NO_PROXY={{ stringsJoin .NoProxy "," }}"
`
	registryMirrorConf = `
[plugins."io.containerd.grpc.v1.cri".registry.mirrors]
  [plugins."io.containerd.grpc.v1.cri".registry.mirrors."public.ecr.aws"]
    endpoint = ["https://{{.Endpoint}}"]
  [plugins."io.containerd.grpc.v1.cri".registry.configs."{{.Endpoint}}".tls]
  {{- if not .CACert }}
    insecure_skip_verify = true
  {{- else }}
    ca_file = "/etc/containerd/certs.d/{{.Endpoint}}/ca.crt"
  {{- end }}
`
)

var containerdRestart = []string{"sudo systemctl daemon-reload", "sudo systemctl restart containerd"}

var defaultTemplateFuncMap = template.FuncMap{
	"Indent": userdata.TemplateYAMLIndent,
}

func generate(kind string, tpl string, data interface{}) ([]byte, error) {
	tm := template.New(kind).Funcs(defaultTemplateFuncMap)
	if _, err := tm.Parse(filesTemplate); err != nil {
		return nil, errors.Wrap(err, "failed to parse files template")
	}

	if _, err := tm.Parse(commandsTemplate); err != nil {
		return nil, errors.Wrap(err, "failed to parse commands template")
	}

	if _, err := tm.Parse(ntpTemplate); err != nil {
		return nil, errors.Wrap(err, "failed to parse ntp template")
	}

	if _, err := tm.Parse(usersTemplate); err != nil {
		return nil, errors.Wrap(err, "failed to parse users template")
	}

	if _, err := tm.Parse(diskSetupTemplate); err != nil {
		return nil, errors.Wrap(err, "failed to parse disk setup template")
	}

	if _, err := tm.Parse(fsSetupTemplate); err != nil {
		return nil, errors.Wrap(err, "failed to parse fs setup template")
	}

	if _, err := tm.Parse(mountsTemplate); err != nil {
		return nil, errors.Wrap(err, "failed to parse mounts template")
	}

	t, err := tm.Parse(tpl)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse %s template", kind)
	}

	var out bytes.Buffer
	if err := t.Execute(&out, data); err != nil {
		return nil, errors.Wrapf(err, "failed to generate %s template", kind)
	}

	return out.Bytes(), nil
}

func prepare(input *userdata.BaseUserData) error {
	input.Header = cloudConfigHeader
	input.WriteFiles = append(input.WriteFiles, input.AdditionalFiles...)
	input.SentinelFileCommand = sentinelFileCommand
	return nil
}

func buildEtcdadmArgs(config etcdbootstrapv1.EtcdadmConfigSpec) userdata.EtcdadmArgs {
	return userdata.EtcdadmArgs{
		Version:        config.CloudInitConfig.Version,
		EtcdReleaseURL: config.CloudInitConfig.EtcdReleaseURL,
		InstallDir:     config.CloudInitConfig.InstallDir,
		CipherSuites:   config.CipherSuites,
	}
}

func setProxy(proxy *etcdbootstrapv1.ProxyConfiguration, input *userdata.BaseUserData) error {
	if proxy == nil {
		return nil
	}
	tmpl := template.New("proxy").Funcs(template.FuncMap{"stringsJoin": strings.Join})
	t, err := tmpl.Parse(proxyConf)
	if err != nil {
		return fmt.Errorf("failed to parse proxy template: %v", err)
	}

	var out bytes.Buffer
	if err = t.Execute(&out, proxy); err != nil {
		return fmt.Errorf("error generating proxy config file: %v", err)
	}

	input.AdditionalFiles = append(input.AdditionalFiles, capbk.File{
		Content: out.String(),
		Owner:   "root:root",
		Path:    "/etc/systemd/system/containerd.service.d/http-proxy.conf",
	})

	input.PreEtcdadmCommands = append(input.PreEtcdadmCommands, containerdRestart...)
	return nil
}

func setRegistryMirror(registryMirror *etcdbootstrapv1.RegistryMirrorConfiguration, input *userdata.BaseUserData) error {
	if registryMirror == nil {
		return nil
	}
	tmpl := template.New("registryMirror")
	t, err := tmpl.Parse(registryMirrorConf)
	if err != nil {
		return fmt.Errorf("failed to parse registryMirror template: %v", err)
	}

	var out bytes.Buffer
	if err = t.Execute(&out, registryMirror); err != nil {
		return fmt.Errorf("error generating registryMirror config file: %v", err)
	}

	input.AdditionalFiles = append(input.AdditionalFiles,
		capbk.File{
			Content: registryMirror.CACert,
			Owner:   "root:root",
			Path:    fmt.Sprintf("/etc/containerd/certs.d/%s/ca.crt", registryMirror.Endpoint),
		},
		capbk.File{
			Content: out.String(),
			Owner:   "root:root",
			Path:    "/etc/containerd/config_append.toml",
		},
	)

	input.PreEtcdadmCommands = append(input.PreEtcdadmCommands, `cat /etc/containerd/config_append.toml >> /etc/containerd/config.toml`)
	input.PreEtcdadmCommands = append(input.PreEtcdadmCommands, containerdRestart...)
	return nil
}
