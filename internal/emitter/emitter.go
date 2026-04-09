// Package emitter generates Go source files from the codegen IR using text/template.
// It makes no decisions — all type mapping, naming, and structure come from the IR.
package emitter

import (
	"bytes"
	"fmt"
	"sort"
	"strings"
	"text/template"

	"github.com/iankaplan/ixian/internal/ir"
)

// File represents a single generated Go source file.
type File struct {
	Name    string // e.g., "types.go", "cmd/root.go"
	Content []byte
}

// Emit generates Go source files from the codegen IR.
func Emit(plan *ir.Plan) ([]File, error) {
	var files []File

	// Generate types.go
	typesContent, err := emitTypes(plan)
	if err != nil {
		return nil, fmt.Errorf("emitting types: %w", err)
	}
	files = append(files, File{Name: "types/types.go", Content: typesContent})

	// Generate client.go
	clientContent, err := emitClient(plan)
	if err != nil {
		return nil, fmt.Errorf("emitting client: %w", err)
	}
	files = append(files, File{Name: "client/client.go", Content: clientContent})

	// Generate cmd/root.go
	rootContent, err := emitRootCmd(plan)
	if err != nil {
		return nil, fmt.Errorf("emitting root cmd: %w", err)
	}
	files = append(files, File{Name: "cmd/root.go", Content: rootContent})

	// Generate command files grouped by tag
	groups := groupCommands(plan.Commands)
	for group, cmds := range groups {
		content, err := emitCommandGroup(group, cmds)
		if err != nil {
			return nil, fmt.Errorf("emitting command group %s: %w", group, err)
		}
		files = append(files, File{Name: fmt.Sprintf("cmd/%s.go", group), Content: content})
	}

	// Generate main.go
	mainContent, err := emitMain(plan)
	if err != nil {
		return nil, fmt.Errorf("emitting main: %w", err)
	}
	files = append(files, File{Name: "main.go", Content: mainContent})

	return files, nil
}

func groupCommands(cmds []ir.GoCommand) map[string][]ir.GoCommand {
	groups := make(map[string][]ir.GoCommand)
	for _, cmd := range cmds {
		groups[cmd.GroupName] = append(groups[cmd.GroupName], cmd)
	}
	return groups
}

var funcMap = template.FuncMap{
	"toLower": strings.ToLower,
	"toTitle": func(s string) string {
		if len(s) == 0 {
			return s
		}
		return strings.ToUpper(s[:1]) + s[1:]
	},
	"hasPrefix": strings.HasPrefix,
	"join":      strings.Join,
}

func execTemplate(name, tmpl string, data any) ([]byte, error) {
	t, err := template.New(name).Funcs(funcMap).Parse(tmpl)
	if err != nil {
		return nil, fmt.Errorf("parsing template %s: %w", name, err)
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("executing template %s: %w", name, err)
	}
	return buf.Bytes(), nil
}

const typesTmpl = `package types
{{range .Types}}
{{- if eq .Kind 0}}
// {{.Name}} {{- if .Description}} — {{.Description}}{{end}}
type {{.Name}} struct {
{{- range .Fields}}
	{{.Name}} {{.Type}} ` + "`" + `json:"{{.JSONName}}{{if not .Required}},omitempty{{end}}"` + "`" + `
{{- end}}
}
{{else if eq .Kind 1}}
// {{.Name}} {{- if .Description}} — {{.Description}}{{end}}
type {{.Name}} = {{.AliasOf}}

const (
{{- range .EnumValues}}
	{{$.Name}}{{toTitle .}} {{$.Name}} = "{{.}}"
{{- end}}
)
{{end}}
{{end}}`

func emitTypes(plan *ir.Plan) ([]byte, error) {
	// Sort types for deterministic output
	sorted := make([]ir.GoType, len(plan.Types))
	copy(sorted, plan.Types)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Name < sorted[j].Name })

	data := struct{ Types []ir.GoType }{Types: sorted}
	return execTemplate("types", typesTmpl, data)
}

const clientTmpl = `package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

type Client struct {
	BaseURL    string
	HTTPClient *http.Client
	Headers    map[string]string
{{- range .ClientConfig.AuthSchemes}}
	{{.GoName}} string
{{- end}}
}

func New(baseURL string) *Client {
	return &Client{
		BaseURL:    strings.TrimRight(baseURL, "/"),
		HTTPClient: http.DefaultClient,
		Headers:    make(map[string]string),
	}
}

func (c *Client) Do(method, path string, query url.Values, body any) ([]byte, error) {
	u := c.BaseURL + path
	if len(query) > 0 {
		u += "?" + query.Encode()
	}

	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshaling body: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequest(method, u, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	// Apply auth
{{- range .ClientConfig.AuthSchemes}}
{{- if eq .Type "bearer"}}
	if c.{{.GoName}} != "" {
		req.Header.Set("Authorization", "Bearer "+c.{{.GoName}})
	}
{{- else if eq .Type "basic"}}
	if c.{{.GoName}} != "" {
		req.Header.Set("Authorization", "Basic "+c.{{.GoName}})
	}
{{- else if eq .Type "apiKey"}}
{{- if eq .In "header"}}
	if c.{{.GoName}} != "" {
		req.Header.Set("{{.HeaderName}}", c.{{.GoName}})
	}
{{- else if eq .In "query"}}
	if c.{{.GoName}} != "" {
		q := req.URL.Query()
		q.Set("{{.HeaderName}}", c.{{.GoName}})
		req.URL.RawQuery = q.Encode()
	}
{{- end}}
{{- end}}
{{- end}}

	// Apply custom headers
	for k, v := range c.Headers {
		req.Header.Set(k, v)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(data))
	}

	return data, nil
}
`

func emitClient(plan *ir.Plan) ([]byte, error) {
	return execTemplate("client", clientTmpl, plan)
}

const rootCmdTmpl = `package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	baseURL string
	output  string
	headers []string
{{- range .ClientConfig.AuthSchemes}}
	{{.GoName}} string
{{- end}}
)

var rootCmd = &cobra.Command{
	Use:   "{{.PackageName}}",
	Short: "CLI for {{.PackageName}} API",
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVar(&baseURL, "base-url", "{{.ClientConfig.BaseURL}}", "API base URL")
	rootCmd.PersistentFlags().StringVarP(&output, "output", "o", "json", "Output format (json, raw)")
	rootCmd.PersistentFlags().StringArrayVarP(&headers, "header", "H", nil, "Custom headers in key:value format (repeatable)")
{{- range .ClientConfig.AuthSchemes}}
{{- if eq .Type "bearer"}}
	rootCmd.PersistentFlags().StringVar(&{{.GoName}}, "{{.FlagName}}", "", "Bearer authentication token")
{{- else if eq .Type "basic"}}
	rootCmd.PersistentFlags().StringVar(&{{.GoName}}, "{{.FlagName}}", "", "Basic authentication credentials (base64)")
{{- else if eq .Type "apiKey"}}
	rootCmd.PersistentFlags().StringVar(&{{.GoName}}, "{{.FlagName}}", "", "API key (sent as {{.HeaderName}} {{.In}})")
{{- end}}
{{- end}}
}
`

func emitRootCmd(plan *ir.Plan) ([]byte, error) {
	return execTemplate("root", rootCmdTmpl, plan)
}

const cmdGroupTmpl = `package cmd

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

// Ensure imports are used.
var (
	_ = json.Marshal
	_ = fmt.Sprintf
	_ = url.Values{}
	_ = os.Stdout
	_ = strings.Replace
)

var {{.Group}}Cmd = &cobra.Command{
	Use:   "{{.Group}}",
	Short: "{{toTitle .Group}} operations",
}

func init() {
	rootCmd.AddCommand({{.Group}}Cmd)
{{range .Commands}}
	// {{.OperationID}}
	{
		{{- range .Flags}}
		var flag{{.GoName}} {{.Type}}
		{{- end}}

		cmd := &cobra.Command{
			Use:   "{{.Name}}",
			Short: "{{.Summary}}",
			RunE: func(cmd *cobra.Command, args []string) error {
				{{- if .Flags}}
				query := url.Values{}
				path := "{{.Path}}"
				{{range .Flags -}}
				{{if eq .In "path"}}
				path = strings.Replace(path, "{{"{"}}{{.Name}}{{"}"}}", fmt.Sprintf("%v", flag{{.GoName}}), 1)
				{{else if eq .In "query"}}
				if cmd.Flags().Changed("{{.Name}}") {
					query.Set("{{.Name}}", fmt.Sprintf("%v", flag{{.GoName}}))
				}
				{{end}}
				{{- end}}
				_ = query
				_ = path
				{{- end}}
				return nil
			},
		}
		{{range .Flags -}}
		{{if eq .Type "string"}}
		cmd.Flags().StringVar(&flag{{.GoName}}, "{{.Name}}", "", "{{.Description}}")
		{{else if eq .Type "int32"}}
		cmd.Flags().Int32Var(&flag{{.GoName}}, "{{.Name}}", 0, "{{.Description}}")
		{{else if eq .Type "int64"}}
		cmd.Flags().Int64Var(&flag{{.GoName}}, "{{.Name}}", 0, "{{.Description}}")
		{{else if eq .Type "int"}}
		cmd.Flags().IntVar(&flag{{.GoName}}, "{{.Name}}", 0, "{{.Description}}")
		{{else if eq .Type "bool"}}
		cmd.Flags().BoolVar(&flag{{.GoName}}, "{{.Name}}", false, "{{.Description}}")
		{{else if eq .Type "float64"}}
		cmd.Flags().Float64Var(&flag{{.GoName}}, "{{.Name}}", 0, "{{.Description}}")
		{{else if eq .Type "float32"}}
		cmd.Flags().Float32Var(&flag{{.GoName}}, "{{.Name}}", 0, "{{.Description}}")
		{{else}}
		cmd.Flags().StringVar(&flag{{.GoName}}, "{{.Name}}", "", "{{.Description}}")
		{{end}}
		{{- if .Required}}
		_ = cmd.MarkFlagRequired("{{.Name}}")
		{{- end}}
		{{end}}
		{{.GroupName}}Cmd.AddCommand(cmd)
	}
{{end}}
}
`

func emitCommandGroup(group string, cmds []ir.GoCommand) ([]byte, error) {
	// Sort commands for deterministic output
	sorted := make([]ir.GoCommand, len(cmds))
	copy(sorted, cmds)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Name < sorted[j].Name })

	data := struct {
		Group    string
		Commands []ir.GoCommand
	}{
		Group:    group,
		Commands: sorted,
	}
	return execTemplate("cmdGroup", cmdGroupTmpl, data)
}

const mainTmpl = `package main

import "{{.PackageName}}/cmd"

func main() {
	cmd.Execute()
}
`

func emitMain(plan *ir.Plan) ([]byte, error) {
	return execTemplate("main", mainTmpl, plan)
}
