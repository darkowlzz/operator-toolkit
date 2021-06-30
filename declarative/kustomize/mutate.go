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

// AddImages returns a MutateFunc which adds images to kustomization object.
// NOTE: In kustomization, if an images list contains duplicate entries for the
// same name, the first entry takes precedence. To avoid confusion, any
// existing entry with the same name is updated with the new entry.
func AddImages(images []apitypes.Image) MutateFunc {
	return func(k *apitypes.Kustomization) {
		// Remove duplicate entries from the list of images, giving preference
		// to the last instance of the image in the list.
		uniqueImages := []apitypes.Image{}
		imageMap := map[string]apitypes.Image{}
		// Record the sequence of the images for a deterministic result.
		sequence := []string{}
		for _, i := range images {
			if _, exists := imageMap[i.Name]; !exists {
				sequence = append(sequence, i.Name)
			}
			imageMap[i.Name] = i
		}
		// Populate the unique images in the recorded sequence.
		for _, iName := range sequence {
			uniqueImages = append(uniqueImages, imageMap[iName])
		}

		newImages := []apitypes.Image{}

		// Compare the images in kustomization and the image list and add
		// unique images in kustomization to the new images list. This will get
		// rid of any duplicates in the kustomization images.
		for _, eImage := range k.Images {
			found := false
			for _, nImage := range uniqueImages {
				// If entry for the same name is found, ignore it.
				if eImage.Name == nImage.Name {
					found = true
					break
				}
			}
			if !found {
				newImages = append(newImages, eImage)
			}
		}

		// Add new images.
		newImages = append(newImages, uniqueImages...)

		// Set kustomization images.
		k.Images = newImages
	}
}

// AddNamePrefix returns a MutateFunc which adds namePrefix to kustomization
// object.
func AddNamePrefix(prefix string) MutateFunc {
	return func(k *apitypes.Kustomization) {
		k.NamePrefix = prefix
	}
}

// AddNameSuffix returns a MutateFunc which adds nameSuffix to kustomization
// object.
func AddNameSuffix(suffix string) MutateFunc {
	return func(k *apitypes.Kustomization) {
		k.NameSuffix = suffix
	}
}
