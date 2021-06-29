package cloudinit

import "fmt"

type EtcdadmArgs struct {
	Version        string
	EtcdReleaseURL string
}

func addEtcdadmFlags(input *EtcdadmArgs, cmd string) string {
	if input.Version != "" {
		cmd += fmt.Sprintf(" --version %s", input.Version)
	}
	if input.EtcdReleaseURL != "" {
		cmd += fmt.Sprintf(" --release-url %s", input.EtcdReleaseURL)
	}
	return cmd
}
