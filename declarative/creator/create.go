package creator

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/pkg/errors"
	kruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/resource"
	"k8s.io/klog"
	"k8s.io/kubectl/pkg/cmd/create"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
	"k8s.io/kubectl/pkg/rawhttp"
	"k8s.io/kubectl/pkg/scheme"
	"k8s.io/kubectl/pkg/util"
)

type DirectCreator struct {
	ioStreams genericclioptions.IOStreams
}

func NewDirectCreator() *DirectCreator {
	return &DirectCreator{
		ioStreams: genericclioptions.IOStreams{
			In:     os.Stdin,
			Out:    os.Stdout,
			ErrOut: os.Stderr,
		},
	}
}

// IOStreams sets the IOStreams of the creator.
// NOTE: This method is not present in upstream.
func (d *DirectCreator) IOStreams(ioStreams genericclioptions.IOStreams) *DirectCreator {
	d.ioStreams = ioStreams
	return d
}

func (d *DirectCreator) Create(ctx context.Context, manifest string, validate bool) error {
	restClient := genericclioptions.NewConfigFlags(true).WithDeprecatedPasswordFlag()

	file, err := ioutil.TempFile("", "create-*.yaml")
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

	f := cmdutil.NewFactory(restClient)

	createOpts := NewCreateOptions(d.ioStreams, fopts)
	complete(createOpts, f)
	return runCreate(createOpts, f, validate)
}

func NewCreateOptions(ioStreams genericclioptions.IOStreams, fopts resource.FilenameOptions) *create.CreateOptions {
	return &create.CreateOptions{
		FilenameOptions: fopts,
		IOStreams:       ioStreams,
	}
}

// Complete is based on kubectl/pkg/cmd/create CreateOptions.Complete(). It
// populates the CreateOptions with the given Factory.
// NOTE: The cobra dependency has been removed from the function to be used as
// an independent library. Once this package is refactored in upstream, similar
// to the apply package, it'll be easier to populate the CreateOptions without
// a factory and cobra command.
// Refer: https://github.com/kubernetes/kubectl/blob/master/pkg/cmd/create/create.go#L193
func complete(o *create.CreateOptions, f cmdutil.Factory) error {
	// func (o *CreateOptions) Complete(f cmdutil.Factory, cmd *cobra.Command) error {
	var err error
	// o.RecordFlags.Complete(cmd)
	// o.Recorder, err = o.RecordFlags.ToRecorder()
	// if err != nil {
	// 	return err
	// }

	// o.DryRunStrategy, err = cmdutil.GetDryRunStrategy(cmd)
	// if err != nil {
	// 	return err
	// }
	cmdutil.PrintFlagsWithDryRunStrategy(o.PrintFlags, o.DryRunStrategy)
	dynamicClient, err := f.DynamicClient()
	if err != nil {
		return err
	}
	discoveryClient, err := f.ToDiscoveryClient()
	if err != nil {
		return err
	}
	o.DryRunVerifier = resource.NewDryRunVerifier(dynamicClient, discoveryClient)

	printer, err := o.PrintFlags.ToPrinter()
	if err != nil {
		return err
	}

	o.PrintObj = func(obj kruntime.Object) error {
		return printer.PrintObj(obj, o.Out)
	}

	return nil
}

// RunCreate performs the creation
func runCreate(o *create.CreateOptions, f cmdutil.Factory, validate bool) error {
	// raw only makes sense for a single file resource multiple objects aren't likely to do what you want.
	// the validator enforces this, so
	if len(o.Raw) > 0 {
		restClient, err := f.RESTClient()
		if err != nil {
			return err
		}
		return rawhttp.RawPost(restClient, o.IOStreams, o.Raw, o.FilenameOptions.Filenames[0])
	}

	// if o.EditBeforeCreate {
	// 	return RunEditOnCreate(f, o.PrintFlags, o.RecordFlags, o.IOStreams, cmd, &o.FilenameOptions, o.fieldManager)
	// }
	schema, err := f.Validator(validate)
	if err != nil {
		return err
	}

	cmdNamespace, enforceNamespace, err := f.ToRawKubeConfigLoader().Namespace()
	if err != nil {
		return err
	}

	r := f.NewBuilder().
		Unstructured().
		Schema(schema).
		ContinueOnError().
		NamespaceParam(cmdNamespace).DefaultNamespace().
		FilenameParam(enforceNamespace, &o.FilenameOptions).
		LabelSelectorParam(o.Selector).
		Flatten().
		Do()
	err = r.Err()
	if err != nil {
		return err
	}

	count := 0
	err = r.Visit(func(info *resource.Info, err error) error {
		if err != nil {
			return err
		}
		// not storing
		if err := util.CreateOrUpdateAnnotation(false, info.Object, scheme.DefaultJSONEncoder()); err != nil {
			return cmdutil.AddSourceToErr("creating", info.Source, err)
		}

		if err := o.Recorder.Record(info.Object); err != nil {
			klog.V(4).Infof("error recording current command: %v", err)
		}

		if o.DryRunStrategy != cmdutil.DryRunClient {
			if o.DryRunStrategy == cmdutil.DryRunServer {
				if err := o.DryRunVerifier.HasSupport(info.Mapping.GroupVersionKind); err != nil {
					return cmdutil.AddSourceToErr("creating", info.Source, err)
				}
			}
			obj, err := resource.
				NewHelper(info.Client, info.Mapping).
				DryRun(o.DryRunStrategy == cmdutil.DryRunServer).
				WithFieldManager("kubectl-create"). // default value
				Create(info.Namespace, true, info.Object)
			if err != nil {
				return cmdutil.AddSourceToErr("creating", info.Source, err)
			}
			info.Refresh(obj, true)
		}

		count++

		return o.PrintObj(info.Object)
	})
	if err != nil {
		return err
	}
	if count == 0 {
		return fmt.Errorf("no objects passed to create")
	}
	return nil
}
