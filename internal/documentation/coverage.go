// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

// Package docs checks that every registered provider surface has authored documentation inputs.
package docs

import (
	"errors"
	"fmt"
	"io/fs"
	"path"
	"slices"
	"strings"

	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

var ErrMissingInput = errors.New("registered provider surface is missing an authored documentation input")

type surface struct {
	category string
	example  string
	names    []string
}

// CheckInputs requires a dedicated template, embedded minimal/maximal examples and resource import instructions.
// It reads GetProviderSchema output without configuring a provider or connecting to Active Directory.
func CheckInputs(files fs.FS, schema *tfprotov6.GetProviderSchemaResponse) error {
	if schema == nil || schema.Provider == nil {
		return fmt.Errorf("provider schema: %w", ErrMissingInput)
	}
	var problems []error
	check := func(templatePath string, required []string) {
		content, err := fs.ReadFile(files, templatePath)
		if err != nil {
			problems = append(problems, fmt.Errorf("%s: %w", templatePath, ErrMissingInput))
			return
		}
		if !strings.Contains(string(content), ".SchemaMarkdown") {
			problems = append(problems, fmt.Errorf("%s must render SchemaMarkdown: %w", templatePath, ErrMissingInput))
		}
		for _, name := range required {
			data, err := fs.ReadFile(files, name)
			if err != nil || len(strings.TrimSpace(string(data))) == 0 {
				problems = append(problems, fmt.Errorf("%s: %w", name, ErrMissingInput))
			}
			include := `tffile "` + name + `"`
			if !strings.Contains(string(content), include) {
				problems = append(problems, fmt.Errorf("%s must embed %s: %w", templatePath, name, ErrMissingInput))
			}
		}
	}
	check("templates/index.md.tmpl", []string{"examples/provider/provider.tf"})
	surfaces := []surface{
		{"resources", "resource.tf", keys(schema.ResourceSchemas)},
		{"data-sources", "data-source.tf", keys(schema.DataSourceSchemas)},
		{"list-resources", "list-resource.tfquery.hcl", keys(schema.ListResourceSchemas)},
		{"actions", "action.tf", keys(schema.ActionSchemas)},
	}
	for _, item := range surfaces {
		for _, name := range item.names {
			short, ok := strings.CutPrefix(name, "activedirectory_")
			if !ok || short == "" || path.Base(short) != short || strings.ContainsAny(short, `\:`) {
				problems = append(problems, fmt.Errorf("invalid registered type name %q: %w", name, ErrMissingInput))
				continue
			}
			folder := path.Join("examples", item.category, name)
			maximal := strings.Replace(item.example, ".tf", "_maximal.tf", 1)
			required := []string{path.Join(folder, item.example), path.Join(folder, maximal)}
			if item.category == "resources" {
				required = append(required, path.Join(folder, "import-by-string-id.tf"))
			}
			check(path.Join("templates", item.category, short+".md.tmpl"), required)
		}
	}
	return errors.Join(problems...)
}

func keys[T any](values map[string]T) []string {
	names := make([]string, 0, len(values))
	for name := range values {
		names = append(names, name)
	}
	slices.Sort(names)
	return names
}
