package deleter

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/pkg/errors"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/resource"
	"k8s.io/kubectl/pkg/cmd/delete"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
)

// DirectDeleter deletes a given manifest. It is based on DirectApplier.
// NOTE: This implementation will change after the upstream deleter package is
// refactored to be more like the applier.
type DirectDeleter struct{}

// NewDirectDeleter returns an instance of a DirectDeleter.
func NewDirectDeleter() *DirectDeleter {
	return &DirectDeleter{}
}

// Delete deletes the given manifest.
func (d *DirectDeleter) Delete(ctx context.Context, manifest string) error {
	// Create iostreams for the deleter.
	// TODO: Make this configurable.
	ioStreams := genericclioptions.IOStreams{
		In:     os.Stdin,
		Out:    os.Stdout,
		ErrOut: os.Stderr,
	}

	// Create a new factory for the deleter.
	restClient := genericclioptions.NewConfigFlags(true).WithDeprecatedPasswordFlag()
	matchVersionKubeConfigFlags := cmdutil.NewMatchVersionFlags(restClient)
	f := cmdutil.NewFactory(matchVersionKubeConfigFlags)

	// Write the given file into a temporary file and pass that to the
	// FilenameOptions as Filenames.
	// NOTE: This should not be necessary in the future. Upstream deleter
	// refactoring can help get around this easily, similar to the applier, and
	// resource objects can be easily constructed from a stream and consumed by
	// the delete runner.
	file, err := ioutil.TempFile("", "delete-*.yaml")
	if err != nil {
		return errors.Wrapf(err, "failed to create manifest %q", file.Name())
	}
	defer os.Remove(file.Name())

	_, err = file.WriteString(manifest)
	if err != nil {
		return errors.Wrapf(err, "failed to write manifest %q", file.Name())
	}
	file.Close()

	fopts := resource.FilenameOptions{
		Filenames: []string{file.Name()},
	}

	// Create new delete options, populate the options and run delete.
	opts := NewDeleteOptions(ioStreams, fopts)
	if err := complete(opts, f, []string{}); err != nil {
		return err
	}

	return opts.RunDelete(f)
}

func NewDeleteOptions(ioStreams genericclioptions.IOStreams, fopts resource.FilenameOptions) *delete.DeleteOptions {
	return &delete.DeleteOptions{
		FilenameOptions: fopts,
		IOStreams:       ioStreams,
	}
}

// Complete is based on kubectl/pkg/cmd/delete DeleteOptions.Complete(). It
// populates the DeleteOptions with the given Factory.
// NOTE: The cobra dependency has been removed from the function to be used as
// an independent library. Once this package is refactored in upstream, similar
// to the apply package, it'll be easier to populate the DeleteOptions without
// a factory and cobra command.
// Refer: https://github.com/kubernetes/kubectl/blob/v0.19.2/pkg/cmd/delete/delete.go#L153
func complete(o *delete.DeleteOptions, f cmdutil.Factory, args []string) error {
	// func (o *ApplyOptions) Complete(f cmdutil.Factory, cmd *cobra.Command) error {
	cmdNamespace, enforceNamespace, err := f.ToRawKubeConfigLoader().Namespace()
	if err != nil {
		return err
	}

	o.WarnClusterScope = enforceNamespace && !o.DeleteAllNamespaces

	// if o.DeleteAll || len(o.LabelSelector) > 0 || len(o.FieldSelector) > 0 {
	//     if f := cmd.Flags().Lookup("ignore-not-found"); f != nil && !f.Changed {
	//         // If the user didn't explicitly set the option, default to ignoring NotFound errors when used with --all, -l, or --field-selector
	//         o.IgnoreNotFound = true
	//     }
	// }
	if o.DeleteNow {
		if o.GracePeriod != -1 {
			return fmt.Errorf("--now and --grace-period cannot be specified together")
		}
		o.GracePeriod = 1
	}
	if o.GracePeriod == 0 && !o.ForceDeletion {
		// To preserve backwards compatibility, but prevent accidental data loss, we convert --grace-period=0
		// into --grace-period=1. Users may provide --force to bypass this conversion.
		o.GracePeriod = 1
	}
	if o.ForceDeletion && o.GracePeriod < 0 {
		o.GracePeriod = 0
	}

	// o.DryRunStrategy, err = cmdutil.GetDryRunStrategy(cmd)
	// if err != nil {
	//     return err
	// }
	dynamicClient, err := f.DynamicClient()
	if err != nil {
		return err
	}
	discoveryClient, err := f.ToDiscoveryClient()
	if err != nil {
		return err
	}
	o.DryRunVerifier = resource.NewDryRunVerifier(dynamicClient, discoveryClient)

	if len(o.Raw) == 0 {
		r := f.NewBuilder().
			Unstructured().
			ContinueOnError().
			NamespaceParam(cmdNamespace).DefaultNamespace().
			FilenameParam(enforceNamespace, &o.FilenameOptions).
			LabelSelectorParam(o.LabelSelector).
			FieldSelectorParam(o.FieldSelector).
			SelectAllParam(o.DeleteAll).
			AllNamespaces(o.DeleteAllNamespaces).
			ResourceTypeOrNameArgs(false, args...).RequireObject(false).
			Flatten().
			Do()
		err = r.Err()
		if err != nil {
			return err
		}
		o.Result = r

		o.Mapper, err = f.ToRESTMapper()
		if err != nil {
			return err
		}

		o.DynamicClient, err = f.DynamicClient()
		if err != nil {
			return err
		}
	}

	return nil
}
