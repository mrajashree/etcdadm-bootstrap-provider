package cloudinit

import (
	"fmt"

	"github.com/mrajashree/etcdadm-bootstrap-provider/pkg/userdata"
)

func addEtcdadmFlags(input *userdata.EtcdadmArgs, cmd string) string {
	if input.Version != "" {
		cmd += fmt.Sprintf(" --version %s", input.Version)
	}
	if input.EtcdReleaseURL != "" {
		cmd += fmt.Sprintf(" --release-url %s", input.EtcdReleaseURL)
	}
	return cmd
}
