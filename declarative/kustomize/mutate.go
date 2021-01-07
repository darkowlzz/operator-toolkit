package kustomize

import (
	apitypes "sigs.k8s.io/kustomize/api/types"
)

// MutateFunc is a type for the kustomization mutate functions.
type MutateFunc func(k *apitypes.Kustomization)

// Mutate takes a Kustomization object and a list of mutations to apply and
// applies all the mutations.
func Mutate(k *apitypes.Kustomization, funcs []MutateFunc) {
	for _, f := range funcs {
		f(k)
	}
}

// AddCommonLabels returns a MutateFunc which adds common labels to a
// kustomization object.
func AddCommonLabels(labels map[string]string) MutateFunc {
	return func(k *apitypes.Kustomization) {
		if k.CommonLabels == nil {
			k.CommonLabels = labels
			return
		}

		for key, val := range labels {
			k.CommonLabels[key] = val
		}
	}
}

// AddCommonLabels returns a MutateFunc which adds common annotations to a
// kustomization object.
func AddCommonAnnotations(annotations map[string]string) MutateFunc {
	return func(k *apitypes.Kustomization) {
		if k.CommonAnnotations == nil {
			k.CommonAnnotations = annotations
			return
		}

		for key, val := range annotations {
			k.CommonAnnotations[key] = val
		}
	}
}

// AddNamespace returns a MutateFunc which adds a namespace to kustomization
// object.
func AddNamespace(namespace string) MutateFunc {
	return func(k *apitypes.Kustomization) {
		k.Namespace = namespace
	}
}

// AddResources returns a MutateFunc which adds resources to kustomization
// object.
func AddResources(resources []string) MutateFunc {
	return func(k *apitypes.Kustomization) {
		k.Resources = append(k.Resources, resources...)
	}
}
