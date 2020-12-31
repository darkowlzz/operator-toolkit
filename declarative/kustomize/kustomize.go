package kustomize

import (
	"fmt"
	"html/template"
	"strings"

	"sigs.k8s.io/kustomize/api/filesys"
	"sigs.k8s.io/kustomize/api/krusty"
)

const kustomizationFile string = "kustomization.yaml"

// Kustomize takes a filesystem and a kustomization configuration and
// runs the kustomization on the filesystem. It returns the result of
// kustomization.
func Kustomize(fs filesys.FileSystem, kustom []byte) (result []byte, err error) {
	// Create a kustomization file with the given content.
	if fErr := fs.WriteFile(kustomizationFile, kustom); fErr != nil {
		err = fErr
		return
	}

	// Remove the kustomization file at the end.
	defer func() {
		if rmErr := fs.RemoveAll(kustomizationFile); err != nil {
			err = rmErr
		}
	}()

	// Run kustomization.
	opt := krusty.MakeDefaultOptions()
	k := krusty.MakeKustomizer(fs, opt)
	m, rErr := k.Run("/")
	if rErr != nil {
		err = rErr
		return
	}

	return m.AsYaml()
}

// RenderKustomizationTemplate reads a template from the given filesystem in
// the given path and renders it with the given data.
func RenderKustomizationTemplate(fs filesys.FileSystem, path string, data interface{}) ([]byte, error) {
	templateContent, err := fs.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read template file %q: %w", path, err)
	}

	var result strings.Builder
	tmpl, err := template.New("kustomization").Parse(string(templateContent))
	if err != nil {
		return nil, fmt.Errorf("error parsing template %q: %w", path, err)
	}
	if err := tmpl.Execute(&result, data); err != nil {
		return nil, fmt.Errorf("error executing template %q: %w", path, err)
	}

	return []byte(result.String()), nil
}

// RenderTemplateAndKustomize renders the kustomization template and runs
// kustomization.
func RenderTemplateAndKustomize(fs filesys.FileSystem, path string, data interface{}) ([]byte, error) {
	m, err := RenderKustomizationTemplate(fs, path, data)
	if err != nil {
		return nil, err
	}

	return Kustomize(fs, m)
}
