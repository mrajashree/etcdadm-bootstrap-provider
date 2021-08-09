package bottlerocket

import (
	"github.com/go-logr/logr"
	"github.com/mrajashree/etcdadm-bootstrap-provider/pkg/userdata"
)

func prepare(input *userdata.BaseUserData) {
	input.Header = cloudConfigHeader
	input.WriteFiles = append(input.WriteFiles, input.AdditionalFiles...)
	input.SentinelFileCommand = sentinelFileCommand
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
