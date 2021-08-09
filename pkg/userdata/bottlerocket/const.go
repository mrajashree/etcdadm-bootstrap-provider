package bottlerocket

const (
	filesTemplate = `{{ define "files" -}}
write_files:{{ range . }}
-   path: {{.Path}}
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
	usersTemplate = `{{- if . }}
{
	"ssh": {
		"authorized-keys": [{{.}}]
	}
}
{{- end -}}
`
)
