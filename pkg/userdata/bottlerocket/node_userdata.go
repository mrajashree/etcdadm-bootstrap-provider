package bottlerocket

import (
	"bytes"
	"encoding/base64"
	"strconv"
	"strings"
	"text/template"

	"github.com/go-logr/logr"
	bootstrapv1alpha3 "github.com/mrajashree/etcdadm-bootstrap-provider/api/v1alpha3"
	"github.com/pkg/errors"
	bootstrapv1 "sigs.k8s.io/cluster-api/bootstrap/kubeadm/api/v1alpha3"
)

const (
	adminContainerInitTemplate = `{{ define "adminContainerInitSettings" -}}
[settings.host-containers.admin]
enabled = true
user-data = "{{.AdminContainerUserData}}"
{{- end -}}
`
	kubernetesInitTemplate = `{{ define "kubernetesInitSettings" -}}
[settings.kubernetes]
cluster-domain = "cluster.local"
standalone-mode = true
authentication-mode = "tls"
server-tls-bootstrap = false
pod-infra-container-image = "{{.PauseContainerSource}}"
{{- end -}}
`
	bootstrapHostContainerTemplate = `{{define "bootstrapHostContainerSettings" -}}
[settings.host-containers.kubeadm-bootstrap]
enabled = true
superpowered = true
source = "{{.BootstrapContainerSource}}"
user-data = "{{.BootstrapContainerUserData}}"
{{- end -}}
`

	networkInitTemplate = `{{ define "networkInitSettings" -}}
[settings.network]
https-proxy = "{{.HTTPSProxyEndpoint}}"
no-proxy = [{{stringsJoin .NoProxyEndpoints "," }}]
{{- end -}}
`
	registryMirrorTemplate = `{{ define "registryMirrorSettings" -}}
[settings.container-registry.mirrors]
"public.ecr.aws" = ["https://{{.RegistryMirrorEndpoint}}"]
{{- end -}}
`
	registryMirrorCACertTemplate = `{{ define "registryMirrorCACertSettings" -}}
[settings.pki.registry-mirror-ca]
data = "{{.RegistryMirrorCACert}}"
trusted=true
{{- end -}}
`
	bottlerocketNodeInitSettingsTemplate = `{{template "bootstrapHostContainerSettings" .}}

{{template "adminContainerInitSettings" .}}

{{template "kubernetesInitSettings" .}}

{{- if (ne .HTTPSProxyEndpoint "")}}
{{template "networkInitSettings" .}}
{{- end -}}

{{- if (ne .RegistryMirrorEndpoint "")}}
{{template "registryMirrorSettings" .}}
{{- end -}}

{{- if (ne .RegistryMirrorCACert "")}}
{{template "registryMirrorCACertSettings" .}}
{{- end -}}
`
)

type bottlerocketSettingsInput struct {
	BootstrapContainerUserData string
	AdminContainerUserData     string
	BootstrapContainerSource   string
	PauseContainerSource       string
	HTTPSProxyEndpoint         string
	NoProxyEndpoints           []string
	RegistryMirrorEndpoint     string
	RegistryMirrorCACert       string
}

type hostPath struct {
	Path string
	Type string
}

// generateBottlerocketNodeUserData returns the userdata for the host bottlerocket in toml format
func generateBottlerocketNodeUserData(bootstrapContainerUserData []byte, users []bootstrapv1.User, config bootstrapv1alpha3.EtcdadmConfigSpec, log logr.Logger) ([]byte, error) {
	// base64 encode the bootstrapContainer's user data
	b64BootstrapContainerUserData := base64.StdEncoding.EncodeToString(bootstrapContainerUserData)

	// Parse out all the ssh authorized keys
	sshAuthorizedKeys := getAllAuthorizedKeys(users)

	// generate the userdata for the admin container
	adminContainerUserData, err := generateAdminContainerUserData("InitAdminContainer", usersTemplate, sshAuthorizedKeys)
	if err != nil {
		return nil, err
	}
	b64AdminContainerUserData := base64.StdEncoding.EncodeToString(adminContainerUserData)

	bottlerocketInput := &bottlerocketSettingsInput{
		BootstrapContainerUserData: b64BootstrapContainerUserData,
		AdminContainerUserData:     b64AdminContainerUserData,
		BootstrapContainerSource:   config.BottlerocketConfig.BootstrapImage,
		PauseContainerSource:       config.BottlerocketConfig.PauseImage,
	}

	if config.Proxy != nil {
		bottlerocketInput.HTTPSProxyEndpoint = config.Proxy.HTTPSProxy
		for _, noProxy := range config.Proxy.NoProxy {
			bottlerocketInput.NoProxyEndpoints = append(bottlerocketInput.NoProxyEndpoints, strconv.Quote(noProxy))
		}
	}

	if config.RegistryMirror != nil {
		bottlerocketInput.RegistryMirrorEndpoint = config.RegistryMirror.Endpoint
		if config.RegistryMirror.CACert != "" {
			bottlerocketInput.RegistryMirrorCACert = base64.StdEncoding.EncodeToString([]byte(config.RegistryMirror.CACert))
		}
	}

	bottlerocketNodeUserData, err := generateNodeUserData("InitBottlerocketNode", bottlerocketNodeInitSettingsTemplate, bottlerocketInput)
	if err != nil {
		return nil, err
	}
	log.Info("Generated bottlerocket bootstrap userdata", "bootstrapContainerImage", config.BottlerocketConfig.BootstrapImage)
	return bottlerocketNodeUserData, nil
}

// getAllAuthorizedKeys parses through all the users and return list of all user's authorized ssh keys
func getAllAuthorizedKeys(users []bootstrapv1.User) string {
	var sshAuthorizedKeys []string
	for _, user := range users {
		if len(user.SSHAuthorizedKeys) != 0 {
			for _, key := range user.SSHAuthorizedKeys {
				quotedKey := "\"" + key + "\""
				sshAuthorizedKeys = append(sshAuthorizedKeys, quotedKey)
			}
		}
	}
	return strings.Join(sshAuthorizedKeys, ",")
}

func generateAdminContainerUserData(kind string, tpl string, data interface{}) ([]byte, error) {
	tm := template.New(kind)
	if _, err := tm.Parse(usersTemplate); err != nil {
		return nil, errors.Wrapf(err, "failed to parse users - %s template", kind)
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

func generateNodeUserData(kind string, tpl string, data interface{}) ([]byte, error) {
	tm := template.New(kind).Funcs(template.FuncMap{"stringsJoin": strings.Join})
	if _, err := tm.Parse(bootstrapHostContainerTemplate); err != nil {
		return nil, errors.Wrapf(err, "failed to parse hostContainer %s template", kind)
	}
	if _, err := tm.Parse(adminContainerInitTemplate); err != nil {
		return nil, errors.Wrapf(err, "failed to parse adminContainer %s template", kind)
	}
	if _, err := tm.Parse(kubernetesInitTemplate); err != nil {
		return nil, errors.Wrapf(err, "failed to parse kubernetes %s template", kind)
	}
	if _, err := tm.Parse(networkInitTemplate); err != nil {
		return nil, errors.Wrapf(err, "failed to parse networks %s template", kind)
	}
	if _, err := tm.Parse(registryMirrorTemplate); err != nil {
		return nil, errors.Wrapf(err, "failed to parse registry mirror %s template", kind)
	}
	if _, err := tm.Parse(registryMirrorCACertTemplate); err != nil {
		return nil, errors.Wrapf(err, "failed to parse registry mirror ca cert %s template", kind)
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
