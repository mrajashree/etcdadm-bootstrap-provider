package userdata

import (
	"fmt"
	"strings"

	bootstrapv1 "sigs.k8s.io/cluster-api/bootstrap/kubeadm/api/v1alpha3"
	"sigs.k8s.io/cluster-api/util/secret"
)

// ControlPlaneInput defines the context to generate a controlplane instance user data.
type EtcdPlaneInput struct {
	BaseUserData
	secret.Certificates
	EtcdadmArgs

	EtcdadmInitCommand string
}

// EtcdPlaneJoinInput defines context to generate etcd instance user data for etcd plane node join.
type EtcdPlaneJoinInput struct {
	BaseUserData
	secret.Certificates
	EtcdadmArgs

	EtcdadmJoinCommand string
	JoinAddress        string
}

// BaseUserData is shared across all the various types of files written to disk.
type BaseUserData struct {
	Header              string
	PreEtcdadmCommands  []string
	PostEtcdadmCommands []string
	AdditionalFiles     []bootstrapv1.File
	WriteFiles          []bootstrapv1.File
	Users               []bootstrapv1.User
	NTP                 *bootstrapv1.NTP
	DiskSetup           *bootstrapv1.DiskSetup
	Mounts              []bootstrapv1.MountPoints
	ControlPlane        bool
	SentinelFileCommand string
}

type EtcdadmArgs struct {
	Version         string
	ImageRepository string
	EtcdReleaseURL  string
	InstallDir      string
}

func (args *EtcdadmArgs) SystemdFlags() []string {
	flags := make([]string, 0, 3)
	flags = append(flags, "--init-system systemd")
	if args.Version != "" {
		flags = append(flags, fmt.Sprintf("--version %s", args.Version))
	}
	if args.ImageRepository != "" {
		flags = append(flags, fmt.Sprintf(" --release-url %s", args.EtcdReleaseURL))
	}
	if args.InstallDir != "" {
		flags = append(flags, fmt.Sprintf("--install-dir %s", args.InstallDir))
	}
	return flags
}

func AddSystemdArgsToCommand(cmd string, args *EtcdadmArgs) string {
	flags := args.SystemdFlags()
	fullCommand := make([]string, len(flags)+1)
	fullCommand = append(fullCommand, cmd)
	fullCommand = append(fullCommand, flags...)

	return strings.Join(fullCommand, " ")
}
