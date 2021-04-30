/*
Copyright 2020.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strconv"
	"time"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	extcache "github.com/darkowlzz/operator-toolkit/cache"
	"github.com/darkowlzz/operator-toolkit/telemetry/export"
	"github.com/darkowlzz/operator-toolkit/webhook/cert"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/discovery"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	appv1alpha1 "github.com/darkowlzz/operator-toolkit/example/api/v1alpha1"
	"github.com/darkowlzz/operator-toolkit/example/controllers"
	//+kubebuilder:scaffold:imports
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(appv1alpha1.AddToScheme(scheme))
	//+kubebuilder:scaffold:scheme
}

func main() {
	var metricsAddr string
	var enableLeaderElection bool
	var probeAddr string
	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	opts := zap.Options{
		Development: true,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	// Setup telemetry.
	telemetryShutdown, err := export.InstallJaegerExporter("game-operator")
	if err != nil {
		setupLog.Error(err, "unable to setup telemetry exporter")
		os.Exit(1)
	}
	defer telemetryShutdown()

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme,
		MetricsBindAddress:     metricsAddr,
		Port:                   9443,
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "f4d65789.example.com",
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	// Create an uncached client to be used in the certificate manager.
	// NOTE: Cached client from manager can't be used here because the cache is
	// uninitialized at this point.
	cli, err := client.New(mgr.GetConfig(), client.Options{Scheme: mgr.GetScheme()})
	if err != nil {
		setupLog.Error(err, "failed to create raw client")
		os.Exit(1)
	}
	// Configure the certificate manager.
	certOpts := cert.Options{
		CertRefreshInterval: 10 * time.Second,
		Service: &admissionregistrationv1.ServiceReference{
			Name:      "webhook-service",
			Namespace: "system",
		},
		Client:                      cli,
		SecretRef:                   &types.NamespacedName{Name: "webhook-secret", Namespace: "system"},
		MutatingWebhookConfigRefs:   []types.NamespacedName{{Name: "mutating-webhook-configuration"}},
		ValidatingWebhookConfigRefs: []types.NamespacedName{{Name: "validating-webhook-configuration"}},
	}
	// Create certificate manager without manager to start the provisioning
	// immediately.
	// NOTE: Certificate Manager implements nonLeaderElectionRunnable interface
	// but since the webhook server is also a nonLeaderElectionRunnable, they
	// start at the same time, resulting in a race condition where sometimes
	// the certificates aren't available when the webhook server starts. By
	// passing nil instead of the manager, the certificate manager is not
	// managed by the controller manager. It starts immediately, in a blocking
	// fashion, ensuring that the cert is created before the webhook server
	// starts.
	if err := cert.NewManager(nil, certOpts); err != nil {
		setupLog.Error(err, "unable to provision certificate")
		os.Exit(1)
	}

	// Create a new cache for an external system events.
	spaceCache := createSpaceCache(mgr.GetScheme())
	// Let the controller manager manage it.
	if err := mgr.Add(spaceCache); err != nil {
		setupLog.Error(err, "unable to start space cache")
		os.Exit(1)
	}

	// TODO: Create a cached client that reads form the created cache as a
	// DelegatingClient, refer:
	// https://github.com/kubernetes-sigs/controller-runtime/blob/v0.8.3/pkg/client/split.go#L44.
	// This client can be passed to the controllers that use this cache.

	if err = (&controllers.GameReconciler{
		Client: mgr.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("Game"),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Game")
		os.Exit(1)
	}

	if err = (&controllers.ExternalGameSyncReconciler{
		Client: mgr.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("ExternalGameSync"),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "ExternalGameSync")
		os.Exit(1)
	}

	// This is an external controller whose events are fetched from outside of
	// k8s and stored in a dedicated controller cache. The setup is the same as
	// any other controller except that the cache is not shared with any other
	// controllers.
	if err = (&controllers.SpaceReconciler{
		Client: mgr.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("Space"),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Space")
		os.Exit(1)
	}

	// Following controllers use a different shared cache.

	// This controller watches and reconciles Game objects.
	if err = (&controllers.SpaceInformer1Reconciler{
		Client: mgr.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("SpaceInformer1"),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr, spaceCache); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "SpaceInformer1")
		os.Exit(1)
	}

	// This controller also watches and reconciles Game objects.
	// NOTE: Since this and the above controllers have a shared cache, they
	// both will reconcile on the same events. These controllers can also have
	// predicates to filter the events they receive and selectively reconcile.
	if err = (&controllers.SpaceInformer2Reconciler{
		Client: mgr.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("SpaceInformer2"),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr, spaceCache); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "SpaceInformer2")
		os.Exit(1)
	}

	// This controller uses the same shared cache as the above two controllers
	// but it watches a different object, Pod.
	// NOTE: This Pod data is not fetched from k8s API server, but is using the
	// same API scheme to construct Pod from a different data source.
	if err = (&controllers.PodInformer1Reconciler{
		Client: mgr.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("PodInformer1"),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr, spaceCache); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "PodInformer1")
		os.Exit(1)
	}

	if err = (&controllers.NamespaceRecorderReconciler{
		Client: mgr.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("nsRecorder"),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "NamespaceRecorder")
		os.Exit(1)
	}

	dc, err := discovery.NewDiscoveryClientForConfig(mgr.GetConfig())
	if err != nil {
		setupLog.Error(err, "unable to create discovery client")
		os.Exit(1)
	}

	// ConfigMap admission controller that uses unified admission controller.
	if err = controllers.NewConfigMapAdmissionController(
		"configmap-admission-webhook-controller",
		mgr.GetClient(),
		dc,
		ctrl.Log.WithName("admission-controllers").WithName("configmap"),
	).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create admission controller", "controller", "configmap-admission-controller")
		os.Exit(1)
	}

	if err = (&appv1alpha1.Game{}).SetupWebhookWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create webhook", "webhook", "Game")
		os.Exit(1)
	}
	//+kubebuilder:scaffold:builder

	if err := mgr.AddHealthzCheck("health", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("check", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}

func createSpaceCache(scheme *runtime.Scheme) cache.Cache {
	setupLog.Info("starting space cache")

	gameChan := make(chan watch.Event)
	podChan := make(chan watch.Event)
	x := &XClient{
		gameChan: gameChan,
		podChan:  podChan,
		scheme:   scheme,
	}
	// Start the mock API polling.
	// NOTE: In case of a real API server with Watch API, the connection to
	// the watch API will take place when an informer for a particular kind
	// starts.
	// Since this is a mock of a real API client, start the Watch event
	// pusher to push events to the event channel. When the informer starts
	// reading the event channel, it'll receive the events from the mock event
	// pusher.
	// Separate channels for different kinds.
	x.Start()

	lw := extcache.ListWatcher{
		ListWatcherClient: x,
	}

	cacheOpt := extcache.Options{
		Scheme:    scheme,
		Namespace: "default",
	}
	return extcache.New(lw.CreateListWatcherFunc(), cacheOpt)
}

// XClient is a client to an API server that implements List and Watch methods.
type XClient struct {
	// scheme is the scheme used to decode the objects.
	scheme *runtime.Scheme
	// gameChan is an event channel for game related events.
	gameChan chan watch.Event
	// podChan is an event channel for pod related events.
	podChan chan watch.Event
}

// List returns a list of the given object type.
// Determine the object kind and return the appropriate mocked object list.
// NOTE: In case of a real API server, query the API for the type of object
// with context and namespace, and convert the obtained data into the object
// type.
func (c *XClient) List(ctx context.Context, namespace string, obj runtime.Object) (runtime.Object, error) {

	// Convert the object to unstructured object and get the object kind.
	u := &unstructured.Unstructured{}
	if err := c.scheme.Convert(obj, u, nil); err != nil {
		return nil, err
	}

	switch u.GetKind() {
	case "GameList":
		return &appv1alpha1.GameList{
			ListMeta: metav1.ListMeta{
				ResourceVersion: "888889999",
			},
			Items: []appv1alpha1.Game{
				{
					ObjectMeta: metav1.ObjectMeta{Name: "spacefoo1", Namespace: "default"},
					Spec:       appv1alpha1.GameSpec{Foo: "aaa"},
				},
				{
					ObjectMeta: metav1.ObjectMeta{Name: "spacefoo2", Namespace: "default"},
					Spec:       appv1alpha1.GameSpec{Foo: "bbb"},
				},
			},
		}, nil
	case "PodList":
		return &corev1.PodList{
			ListMeta: metav1.ListMeta{
				ResourceVersion: "4444447777",
			},
			Items: []corev1.Pod{
				{
					ObjectMeta: metav1.ObjectMeta{Name: "podfoo1", Namespace: "default"},
					Spec:       corev1.PodSpec{NodeName: "ccc"},
				},
			},
		}, nil
	default:
		return nil, fmt.Errorf("unknown kind to list: %q", u.GetKind())
	}

}

// Watch returns the event channel for the respective kind.
// NOTE: In case of real Watch API sever, use the context, namespace and kind
// to send a watch request for a specific kind of object to the API server and
// return an events channel that streams events from the API server.
func (c *XClient) Watch(ctx context.Context, namespace string, kind string) (watch.Interface, error) {
	// Wrap the event channel with a Watcher.
	switch kind {
	case "Game":
		return watch.NewProxyWatcher(c.gameChan), nil
	case "Pod":
		return watch.NewProxyWatcher(c.podChan), nil
	default:
		return nil, fmt.Errorf("unknown kind to watch: %q", kind)
	}
}

// Start starts the event generators for the event channels.
func (c *XClient) Start() {
	// Game events.
	go func() {
		setupLog.Info("starting mock space watch API server for Games")
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		count := 0
		for {
			<-ticker.C
			count++
			c.gameChan <- watch.Event{
				Type: watch.Modified,
				Object: &appv1alpha1.Game{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "spacefoo1",
						Namespace: "default",
					},
					Spec: appv1alpha1.GameSpec{
						Foo: "aaa" + strconv.Itoa(count),
					},
				},
			}
		}
	}()

	// Pod events.
	go func() {
		setupLog.Info("starting mock space watch API server for Pods")
		ticker := time.NewTicker(7 * time.Second)
		defer ticker.Stop()

		count := 0
		for {
			<-ticker.C
			count++
			c.podChan <- watch.Event{
				Type: watch.Modified,
				Object: &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "podfoo1",
						Namespace: "default",
					},
					Spec: corev1.PodSpec{
						NodeName: "ccc" + strconv.Itoa(count),
					},
				},
			}
		}
	}()
}
