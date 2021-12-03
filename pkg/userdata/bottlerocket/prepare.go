package bottlerocket

import (
	"path/filepath"
	"strings"

	"github.com/go-logr/logr"
	bootstrapv1alpha3 "github.com/mrajashree/etcdadm-bootstrap-provider/api/v1alpha3"
	"github.com/mrajashree/etcdadm-bootstrap-provider/pkg/userdata"
)

const (
	orgCertsPath = "/etc/etcd/pki"
	newCertsPath = "/var/lib/etcd/pki"
)

func prepare(input *userdata.BaseUserData) {
	input.Header = cloudConfigHeader
	input.WriteFiles = append(input.WriteFiles, input.AdditionalFiles...)
	input.SentinelFileCommand = sentinelFileCommand
	patchCertPaths(input)
}

func patchCertPaths(input *userdata.BaseUserData) {
	for ind, file := range input.WriteFiles {
		if filepath.Dir(file.Path) == orgCertsPath {
			file.Path = filepath.Join(newCertsPath, filepath.Base(file.Path))
		}
		input.WriteFiles[ind] = file
	}
}

func buildEtcdadmArgs(config bootstrapv1alpha3.EtcdadmConfigSpec) userdata.EtcdadmArgs {
	repository, tag := splitRepositoryAndTag(config.BottlerocketConfig.EtcdImage)
	return userdata.EtcdadmArgs{
		Version:         strings.TrimPrefix(tag, "v"), // trim "v" to get pure simver because that's what etcdadm expects.
		ImageRepository: repository,
		CipherSuites:    config.CipherSuites,
	}
}

func splitRepositoryAndTag(image string) (repository, tag string) {
	lastInd := strings.LastIndex(image, ":")
	if lastInd == -1 {
		return image, ""
	}

	if lastInd == len(image)-1 {
		return image[:lastInd], ""
	}

	return image[:lastInd], image[lastInd+1:]
}

func logIgnoredFields(input *userdata.BaseUserData, log logr.Logger) {
	if len(input.PreEtcdadmCommands) > 0 {
		log.Info("Ignoring PreEtcdadmCommands. Not supported with bottlerocket")
	}
	if len(input.PostEtcdadmCommands) > 0 {
		log.Info("Ignoring PostEtcdadmCommands. Not supported with bottlerocket")
	}
	if input.NTP != nil {
		log.Info("Ignoring NTP. Not supported with bottlerocket")
	}
	if input.DiskSetup != nil {
		log.Info("Ignoring DiskSetup. Not supported with bottlerocket")
	}
	if len(input.Mounts) > 0 {
		log.Info("Ignoring Mounts. Not supported with bottlerocket")
	}
}
