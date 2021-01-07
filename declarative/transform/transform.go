package transform

import (
	"fmt"
	"html/template"
	"strconv"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/kustomize/api/filesys"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

// TransformFunc is the type of a transform function. A transformation must
// return a TransformFunc.
type TransformFunc func(*yaml.RNode) error

// ManifestTransform is a map of manifest file path and the transformation that
// needs to be run on the manifest.
type ManifestTransform map[string][]TransformFunc

// Transform takes a Filesystem, a ManifestTransform and a set of common
// transforms to be applied on all the manifests, and transforms all the
// manifests.
func Transform(fs filesys.FileSystem, manifestTransform ManifestTransform, commonTransforms ...TransformFunc) error {
	for manifest, transforms := range manifestTransform {
		// Read and convert the manifest into a resource node.
		o, err := fs.ReadFile(manifest)
		if err != nil {
			return err
		}
		obj, err := yaml.Parse(string(o))
		if err != nil {
			return err
		}

		// Run the transformations.
		for _, t := range transforms {
			if err := t(obj); err != nil {
				return fmt.Errorf("failed to transform %q: %w", manifest, err)
			}
		}

		// Run the common transforms.
		for _, t := range commonTransforms {
			if err := t(obj); err != nil {
				return fmt.Errorf("failed to transform %q: %w", manifest, err)
			}
		}

		// Convert the resource node into string and write as manifest file.
		r, e := obj.String()
		if e != nil {
			return e
		}
		if err := fs.WriteFile(manifest, []byte(r)); err != nil {
			return err
		}
	}
	return nil
}

// AddLabelsFunc returns a TransformFunc that adds the given labels to an
// object.
func AddLabelsFunc(labels map[string]string) TransformFunc {
	return func(obj *yaml.RNode) error {
		// Get existing labels.
		l, err := obj.GetLabels()
		if err != nil {
			return err
		}
		if l == nil {
			l = map[string]string{}
		}

		// Append new labels.
		for k, v := range labels {
			l[k] = v
		}

		// Set labels.
		if err := obj.SetLabels(l); err != nil {
			return err
		}
		return nil
	}
}

// AddAnnotationsFunc returns a TransformFunc that adds the given annotations
// to an object.
func AddAnnotationsFunc(annotations map[string]string) TransformFunc {
	return func(obj *yaml.RNode) error {
		// Get existing annotations.
		a, err := obj.GetAnnotations()
		if err != nil {
			return err
		}
		if a == nil {
			a = map[string]string{}
		}

		// Append new annotations.
		for k, v := range annotations {
			a[k] = v
		}

		// Set annotations.
		if err := obj.SetAnnotations(a); err != nil {
			return err
		}
		return nil
	}
}

// SetReplicaFunc returns a TransformFunc that sets the replicas
// (spec.replicas) in a given object.
func SetReplicaFunc(replica int) TransformFunc {
	return func(obj *yaml.RNode) error {
		return obj.PipeE(
			yaml.LookupCreate(yaml.ScalarNode, "spec", "replicas"),
			yaml.Set(yaml.NewScalarRNode(strconv.Itoa(replica))),
		)
	}
}

// ownerRefTemplate is a template for creating OwnerReference.
const ownerRefTemplate = `
- apiVersion: {{.APIVersion}}
  blockOwnerDeletion: {{.BlockOwnerDeletion}}
  controller: {{.Controller}}
  kind: {{.Kind}}
  name: {{.Name}}
  uid: {{.UID}}
`

// OwnerRefTemplateParam stores the data for the OwnerReference template.
type OwnerRefTemplateParam struct {
	APIVersion         string
	Kind               string
	Name               string
	UID                types.UID
	Controller         bool
	BlockOwnerDeletion bool
}

// GetOwnerRefTemplateParamFor returns a OwnerRefTemplateParam for a given
// OwnerReference.
func GetOwnerRefTemplateParamFor(ownerRef metav1.OwnerReference) OwnerRefTemplateParam {
	var controller, blockOwnerDeletion bool
	if ownerRef.Controller != nil {
		controller = *ownerRef.Controller
	}
	if ownerRef.BlockOwnerDeletion != nil {
		blockOwnerDeletion = *ownerRef.BlockOwnerDeletion
	}

	return OwnerRefTemplateParam{
		APIVersion:         ownerRef.APIVersion,
		Kind:               ownerRef.Kind,
		Name:               ownerRef.Name,
		UID:                ownerRef.UID,
		Controller:         controller,
		BlockOwnerDeletion: blockOwnerDeletion,
	}
}

// SetOwnerReference returns a TransformFunc that sets the ownerReferences in a
// given object.
func SetOwnerReference(ownerRefs []metav1.OwnerReference) TransformFunc {
	return func(obj *yaml.RNode) error {
		tmpl, err := template.New("ownerref").Parse(ownerRefTemplate)
		if err != nil {
			return fmt.Errorf("failed parsing OwnerReference template: %w", err)
		}

		// Store all the string owner references in one string variable.
		var stringOwnerRefs strings.Builder

		// Convert OwnerRefs into string values.
		for _, ownerRef := range ownerRefs {
			var orResult strings.Builder
			// Get template param and execute the template.
			ownerRefParam := GetOwnerRefTemplateParamFor(ownerRef)
			if err := tmpl.Execute(&orResult, ownerRefParam); err != nil {
				return fmt.Errorf("failed to execute OwnerReference template: %w", err)
			}
			// Write the result into the owner ref collection.
			_, err := stringOwnerRefs.WriteString(orResult.String())
			if err != nil {
				return fmt.Errorf("failed appending OwnerReference: %w", err)
			}
		}

		// Parse the string ownerRefs.
		parsedOR, err := yaml.Parse(stringOwnerRefs.String())
		if err != nil {
			return fmt.Errorf("failed to parse string OwnerReferences: %w", err)
		}

		// Write the ownerReferences in metadata.ownerReferences of the object.
		return obj.PipeE(
			yaml.LookupCreate(yaml.SequenceNode, "metadata"),
			yaml.SetField("ownerReferences", parsedOR),
		)
	}
}
