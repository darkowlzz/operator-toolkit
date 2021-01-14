// The declarative package contains tools for building the manifests that the
// operators use in a declarative manner.
// A Builder can be used to build all the manifests of a package. The build
// process includes transforming specific manifests, common transformations for
// all the manifests in a package and mutating the kustomization file in the
// package. A builder instance can be used to apply or delete the built
// resource manifest.
package declarative
