package cloudinit

import (
	"bytes"
	"text/template"

	bootstrapv1alpha3 "github.com/mrajashree/etcdadm-bootstrap-provider/api/v1alpha3"
	"github.com/mrajashree/etcdadm-bootstrap-provider/pkg/userdata"
	"github.com/pkg/errors"
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
)

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

func buildEtcdadmArgs(config bootstrapv1alpha3.CloudConfigConfig) userdata.EtcdadmArgs {
	return userdata.EtcdadmArgs{
		Version:        config.Version,
		EtcdReleaseURL: config.EtcdReleaseURL,
	}
}
