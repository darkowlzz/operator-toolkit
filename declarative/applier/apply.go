package applier

import (
	"context"
	"os"
	"strings"

	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/printers"
	"k8s.io/cli-runtime/pkg/resource"
	"k8s.io/kubectl/pkg/cmd/apply"
	cmdDelete "k8s.io/kubectl/pkg/cmd/delete"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
)

// NOTE: This file is mostly based on the kubebuilder-declarative-pattern repo,
// with slight modifications.
// Refer: https://github.com/kubernetes-sigs/kubebuilder-declarative-pattern/blob/f77bb4933dfbae404f03e34b01c84e268cc4b966/pkg/patterns/declarative/pkg/applier/direct.go

type DirectApplier struct {
	// a         apply.ApplyOptions
	ioStreams genericclioptions.IOStreams
}

func NewDirectApplier() *DirectApplier {
	return &DirectApplier{
		ioStreams: genericclioptions.IOStreams{
			In:     os.Stdin,
			Out:    os.Stdout,
			ErrOut: os.Stderr,
		},
	}
}

// IOStreams sets the IOStreams of the applier.
// NOTE: This method is not present in upstream.
func (d *DirectApplier) IOStreams(ioStreams genericclioptions.IOStreams) *DirectApplier {
	d.ioStreams = ioStreams
	return d
}

func (d *DirectApplier) Apply(ctx context.Context,
	namespace string,
	manifest string,
	validate bool,
	extraArgs ...string,
) error {
	// NOTE: This is modified from the upstream to allow configuring IOStreams.
	// ioStreams := genericclioptions.IOStreams{
	//     In:     os.Stdin,
	//     Out:    os.Stdout,
	//     ErrOut: os.Stderr,
	// }
	restClient := genericclioptions.NewConfigFlags(true).WithDeprecatedPasswordFlag()
	ioReader := strings.NewReader(manifest)

	b := resource.NewBuilder(restClient)
	res := b.Unstructured().Stream(ioReader, "manifestString").Do()
	infos, err := res.Infos()
	if err != nil {
		return err
	}

	applyOpts := apply.NewApplyOptions(d.ioStreams)
	applyOpts.Namespace = namespace
	applyOpts.SetObjects(infos)
	applyOpts.ToPrinter = func(operation string) (printers.ResourcePrinter, error) {
		applyOpts.PrintFlags.NamePrintFlags.Operation = operation
		cmdutil.PrintFlagsWithDryRunStrategy(applyOpts.PrintFlags, applyOpts.DryRunStrategy)
		return applyOpts.PrintFlags.ToPrinter()
	}
	applyOpts.DeleteOptions = &cmdDelete.DeleteOptions{
		IOStreams: d.ioStreams,
	}

	return applyOpts.Run()
}
