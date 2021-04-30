# operator-toolkit

[![PkgGoDev](https://pkg.go.dev/badge/github.com/darkowlzz/operator-toolkit)](https://pkg.go.dev/github.com/darkowlzz/operator-toolkit)

operator-toolkit provides framework and tools to help implement kubernetes
operators.

### Packages

#### controller

`controller` package provides tools to implement certain controller patterns.

- `controller/composite` package contains interface and types to implement the
    composite controller pattern.
- `controller/sync` package contains interface and types to implement sync
    controller pattern.
- `controller/external-object-sync` package uses the sync pattern as the base
    and adds a garbage collector for collecting the orphan objects in external
    system.
- `controller/metadata-sync` package uses the sync pattern as the base and adds
    a resync to periodically sync object metadata between systems.
- `controller/external` package provides tools for building external
    controllers with the same core components that a k8s controller uses but
    for an external system based on non-k8s event source.

#### operator

`operator` package provides tools to implement the core business logic of an
operator that interacts with the world. An `Operand` is a unit of work. An
`Operator` can have one or more `Operand`s. The relationship between the
`Operand`s is modelled using a Directed Asyclic Graph (DAG). The `Operator` can
be configured to define how the `Operand`s are executed.

#### declarative

`declarative` package provides tools to create and transform the kubernetes
manifests in a declarative way. It uses kustomize tools to read, organize and
transform the manifests with the desired configuration. This helps avoid
writing Go structs for all the kubernetes objects and write generic reusable
transforms.

#### rbac

`rbac/client` package provides a controller-runtime generic client
implementation capable of recording the actions performed using the client on
the objects and generate RBAC permissions as Role and ClusterRole.

#### telemetry

`telemetry/export` package provides opentelemetry exporters that can be used to
enable telemetry in an operator.

#### webhook

- `webhook/cert` package provides a local certificate manager that can provision
    self signed certificate for webhook server, backed by k8s secret for
    persistence, and refresh certificate when they expire automatically.

- `webhook/admission` package provides a consistent way of building admission
    controllers for k8s native and custom resources with function chaining
    support.

The above packages can be used together or independently of each other.
`example/` contains an example of using all the packages together in a
kubebuilder based operator.
