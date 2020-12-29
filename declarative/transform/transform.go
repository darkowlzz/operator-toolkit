package transform

import (
	"fmt"
	"strconv"

	"sigs.k8s.io/kustomize/kyaml/yaml"

	"github.com/darkowlzz/composite-reconciler/declarative/loader"
)

// TransformFunc is the type of a transform function. A transformation must
// return a TransformFunc.
type TransformFunc func(*yaml.RNode) error

// ManifestTransform is a map of manifest file path and the transformation that
// needs to be run on the manifest.
type ManifestTransform map[string][]TransformFunc

// AddLabels takes a filesystem that contains a manifest to be transformed,
// path to the manifest file and a label to be added, and modifies the manifest
// file to add the labels.
func Transform(fs loader.ManifestFileSystem, manifestTransform ManifestTransform) error {
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
// (spec.replicas)in a given object.
func SetReplicaFunc(replica int) TransformFunc {
	return func(obj *yaml.RNode) error {
		return obj.PipeE(
			yaml.LookupCreate(yaml.ScalarNode, "spec", "replicas"),
			yaml.Set(yaml.NewScalarRNode(strconv.Itoa(replica))),
		)
	}
}
