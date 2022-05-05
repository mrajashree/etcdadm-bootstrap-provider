package cloudinit

const (
	filesTemplate = `{{ define "files" -}}
write_files:{{ range . }}
-   path: {{.Path}}
    {{ if ne .Encoding "" -}}
    encoding: "{{.Encoding}}"
    {{ end -}}
    {{ if ne .Owner "" -}}
    owner: {{.Owner}}
    {{ end -}}
    {{ if ne .Permissions "" -}}
    permissions: '{{.Permissions}}'
    {{ end -}}
    content: |
{{.Content | Indent 6}}
{{- end -}}
{{- end -}}
`
)

const (
	commandsTemplate = `{{- define "commands" -}}
{{ range . }}
  - {{printf "%q" .}}
{{- end -}}
{{- end -}}
`
)

const (
	ntpTemplate = `{{ define "ntp" -}}
{{- if . }}
ntp:
  {{ if .Enabled -}}
  enabled: true
  {{ end -}}
  servers:{{ range .Servers }}
    - {{ . }}
  {{- end -}}
{{- end -}}
{{- end -}}
`
)

const (
	usersTemplate = `{{ define "users" -}}
{{- if . }}
users:{{ range . }}
  - name: {{ .Name }}
    {{- if .Passwd }}
    passwd: {{ .Passwd }}
    {{- end -}}
    {{- if .Gecos }}
    gecos: {{ .Gecos }}
    {{- end -}}
    {{- if .Groups }}
    groups: {{ .Groups }}
    {{- end -}}
    {{- if .HomeDir }}
    homedir: {{ .HomeDir }}
    {{- end -}}
    {{- if .Inactive }}
    inactive: true
    {{- end -}}
    {{- if .LockPassword }}
    lock_passwd: {{ .LockPassword }}
    {{- end -}}
    {{- if .Shell }}
    shell: {{ .Shell }}
    {{- end -}}
    {{- if .PrimaryGroup }}
    primary_group: {{ .PrimaryGroup }}
    {{- end -}}
    {{- if .Sudo }}
    sudo: {{ .Sudo }}
    {{- end -}}
    {{- if .SSHAuthorizedKeys }}
    ssh_authorized_keys:{{ range .SSHAuthorizedKeys }}
      - {{ . }}
    {{- end -}}
    {{- end -}}
{{- end -}}
{{- end -}}
{{- end -}}
`
)

const (
	diskSetupTemplate = `{{ define "disk_setup" -}}
{{- if . }}
disk_setup:{{ range .Partitions }}
  {{ .Device }}:
    {{- if .TableType }}
    table_type: {{ .TableType }}
    {{- end }}
    layout: {{ .Layout }}
    {{- if .Overwrite }}
    overwrite: {{ .Overwrite }}
    {{- end -}}
{{- end -}}
{{- end -}}
{{- end -}}
`
)

const (
	fsSetupTemplate = `{{ define "fs_setup" -}}
{{- if . }}
fs_setup:{{ range .Filesystems }}
  - label: {{ .Label }}
    filesystem: {{ .Filesystem }}
    device: {{ .Device }}
  {{- if .Partition }}
    partition: {{ .Partition }}
  {{- end }}
  {{- if .Overwrite }}
    overwrite: {{ .Overwrite }}
  {{- end }}
  {{- if .ReplaceFS }}
    replace_fs: {{ .ReplaceFS }}
  {{- end }}
  {{- if .ExtraOpts }}
    extra_opts: {{ range .ExtraOpts }}
      - {{ . }}
        {{- end -}}
  {{- end -}}
{{- end -}}
{{- end -}}
{{- end -}}
`
)

const (
	mountsTemplate = `{{ define "mounts" -}}
{{- if . }}
mounts:{{ range . }}
  - {{ range . }}- {{ . }}
    {{ end -}}
{{- end -}}
{{- end -}}
{{- end -}}
`
)
