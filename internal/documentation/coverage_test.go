// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package docs

import (
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

func TestCoverageRequiresAuthoredInputsForEverySurface(t *testing.T) {
	files, schema := fixture()
	if err := CheckInputs(files, schema); err != nil {
		t.Fatal(err)
	}
	for name := range files {
		t.Run(name, func(t *testing.T) {
			broken, _ := fixture()
			delete(broken, name)
			if err := CheckInputs(broken, schema); !errors.Is(err, ErrMissingInput) {
				t.Fatalf("missing documentation input was accepted: %v", err)
			}
		})
	}
	for _, replacement := range []string{"{{ .SchemaMarkdown }}", "{{ tffile \"examples/provider/provider.tf\" }}"} {
		broken, _ := fixture()
		broken["templates/index.md.tmpl"] = &fstest.MapFile{Data: []byte(replacement)}
		if err := CheckInputs(broken, schema); !errors.Is(err, ErrMissingInput) {
			t.Fatal("template accepted without both schema and example rendering")
		}
	}
	if err := CheckInputs(files, nil); !errors.Is(err, ErrMissingInput) {
		t.Fatal("missing provider schema was accepted")
	}
}

// TestFourSurfaceGeneration runs the pinned external generator against synthetic schemas, never AD.
// CI supplies TF_DOCS_GENERATOR after building the pinned tool once.
func TestFourSurfaceGeneration(t *testing.T) {
	tool := os.Getenv("TF_DOCS_GENERATOR")
	if tool == "" {
		t.Skip("set TF_DOCS_GENERATOR to run the documentation generator integration check")
	}
	files, schema := fixture()
	if err := CheckInputs(files, schema); err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	if err := os.CopyFS(root, files); err != nil {
		t.Fatal(err)
	}
	block := map[string]any{"version": 0, "block": map[string]any{
		"description": "Representative schema for documentation verification.",
		"attributes": map[string]any{
			"name":        map[string]any{"type": "string", "required": true, "description": "Representative schema attribute."},
			"description": map[string]any{"type": "string", "optional": true},
		},
	}}
	provider := map[string]any{"provider": block}
	for _, category := range []string{"resource_schemas", "data_source_schemas", "list_resource_schemas", "action_schemas"} {
		provider[category] = map[string]any{"activedirectory_sample": block}
	}
	payload, err := json.Marshal(map[string]any{
		"format_version":   "1.0",
		"provider_schemas": map[string]any{"activedirectory": provider},
	})
	if err != nil {
		t.Fatal(err)
	}
	schemaPath := filepath.Join(root, "schema.json")
	if err := os.WriteFile(schemaPath, payload, 0o600); err != nil {
		t.Fatal(err)
	}
	command := exec.CommandContext(t.Context(), tool, "generate", "--provider-dir", root,
		"--provider-name", "terraform-provider-activedirectory", "--providers-schema", schemaPath)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("generate documentation: %v\n%s", err, output)
	}
	for _, category := range []string{"resources", "data-sources", "list-resources", "actions"} {
		content, err := os.ReadFile(filepath.Join(root, "docs", category, "sample.md"))
		if err != nil {
			t.Fatal(err)
		}
		for _, expected := range []string{"Representative schema attribute.", "minimal-example", "maximal-example"} {
			if !strings.Contains(string(content), expected) {
				t.Errorf("%s documentation omitted %q", category, expected)
			}
		}
	}
}

func fixture() (fstest.MapFS, *tfprotov6.GetProviderSchemaResponse) {
	files := fstest.MapFS{
		"examples/provider/provider.tf": &fstest.MapFile{Data: []byte(`provider "activedirectory" {}`)},
		"templates/index.md.tmpl":       &fstest.MapFile{Data: []byte("{{ .SchemaMarkdown }}\n{{ tffile \"examples/provider/provider.tf\" }}")},
	}
	for _, item := range []surface{
		{category: "resources", example: "resource.tf"},
		{category: "data-sources", example: "data-source.tf"},
		{category: "list-resources", example: "list-resource.tfquery.hcl"},
		{category: "actions", example: "action.tf"},
	} {
		folder := "examples/" + item.category + "/activedirectory_sample/"
		minimal, maximal := folder+item.example, folder+strings.Replace(item.example, ".tf", "_maximal.tf", 1)
		kind := map[string]string{"resources": "resource", "data-sources": "data", "list-resources": "list", "actions": "action"}[item.category]
		bodyFor := func(name string, maximal bool) []byte {
			config := "  name = \"" + name + "\"\n"
			if maximal {
				config += "  description = \"Optional configuration\"\n"
			}
			if kind == "list" || kind == "action" {
				config = "  config {\n" + config + "  }\n"
			}
			return []byte(kind + " \"activedirectory_sample\" \"example\" {\n" + config + "}\n")
		}
		files[minimal] = &fstest.MapFile{Data: bodyFor("minimal-example", false)}
		files[maximal] = &fstest.MapFile{Data: bodyFor("maximal-example", true)}
		body := "{{ .SchemaMarkdown }}\n{{ tffile \"" + minimal + "\" }}\n{{ tffile \"" + maximal + "\" }}"
		if item.category == "resources" {
			name := folder + "import-by-string-id.tf"
			files[name] = &fstest.MapFile{Data: []byte("import {\n  to = activedirectory_sample.example\n  id = \"a3e9767a-200e-43dd-bb1d-e5ab400e5d6f\"\n}\n")}
			body += "\n{{ tffile \"" + name + "\" }}"
		}
		files["templates/"+item.category+"/sample.md.tmpl"] = &fstest.MapFile{Data: []byte(body)}
	}
	return files, &tfprotov6.GetProviderSchemaResponse{
		Provider:            &tfprotov6.Schema{},
		ResourceSchemas:     map[string]*tfprotov6.Schema{"activedirectory_sample": {}},
		DataSourceSchemas:   map[string]*tfprotov6.Schema{"activedirectory_sample": {}},
		ListResourceSchemas: map[string]*tfprotov6.Schema{"activedirectory_sample": {}},
		ActionSchemas:       map[string]*tfprotov6.ActionSchema{"activedirectory_sample": {}},
	}
}
