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

package controllers

import (
	"fmt"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/kustomize/api/filesys"

	controllerv1 "github.com/darkowlzz/composite-reconciler/controller/v1"
	"github.com/darkowlzz/composite-reconciler/declarative/loader"
	operatorv1 "github.com/darkowlzz/composite-reconciler/operator/v1"
	"github.com/darkowlzz/composite-reconciler/operator/v1/executor"
	"github.com/darkowlzz/composite-reconciler/operator/v1/operand"
	appv1alpha1 "github.com/darkowlzz/composite-reconciler/testdata/example-project/api/v1alpha1"
	"github.com/darkowlzz/composite-reconciler/testdata/example-project/controllers/game"
)

// GameReconciler reconciles a Game object
type GameReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme

	controllerv1.CompositeReconciler
}

//+kubebuilder:rbac:groups=app.example.com,resources=games,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=app.example.com,resources=games/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=app.example.com,resources=games/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Game object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.7.0/pkg/reconcile
// func (r *GameReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
//     _ = r.Log.WithValues("game", req.NamespacedName)

//     // your logic here

//     return ctrl.Result{}, nil
// }

// SetupWithManager sets up the controller with the Manager.
func (r *GameReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// TODO: Move filesystem creation and package loading into a helper
	// function.
	// Setup manifests in an in-memory filesystem.
	fs := loader.ManifestFileSystem{FileSystem: filesys.MakeFsInMemory()}
	// Use default channel.
	err := loader.LoadPackages(fs, "channels", "stable")
	if err != nil {
		return fmt.Errorf("failed to load channel packages: %w", err)
	}

	// TODO: Move the operand and operator creation into a separate function in
	// their package, accepting a manager, filesystem and other options.
	// Create the operands.
	configmapOp := game.NewConfigmapOperand("configmap-operand", mgr.GetClient(), []string{}, operand.RequeueOnError, fs)

	// Create a CompositeOperator using the operands.
	co, err := operatorv1.NewCompositeOperator(
		operatorv1.WithEventRecorder(mgr.GetEventRecorderFor("game-reconciler")),
		operatorv1.WithExecutionStrategy(executor.Parallel),
		operatorv1.WithOperands(configmapOp),
	)
	if err != nil {
		return fmt.Errorf("failed to create new CompositeOperator: %w", err)
	}

	// Create a controller that implements the CompositeReconciler controller
	// interface with a CompositeOperator.
	gc := &game.GameController{
		Operator: co,
	}

	// TODO: Replace this New* function with an Init() function, similar to the
	// addons pattern reconciler.
	// Create a CompositeReconciler and embed it into the GameReconciler.
	cr, err := controllerv1.NewCompositeReconciler(
		controllerv1.WithPrototype(&appv1alpha1.Game{}),
		controllerv1.WithController(gc),
		controllerv1.WithClient(mgr.GetClient()),
		controllerv1.WithScheme(mgr.GetScheme()),
		controllerv1.WithCleanupStrategy(controllerv1.OwnerReferenceCleanup),
		controllerv1.WithInitCondition(controllerv1.DefaultInitCondition),
		controllerv1.WithLogger(r.Log.WithValues("component", "composite-reconciler")),
	)
	if err != nil {
		return fmt.Errorf("failed to create new CompositeReconciler: %w", err)
	}
	r.CompositeReconciler = *cr

	return ctrl.NewControllerManagedBy(mgr).
		For(&appv1alpha1.Game{}).
		Complete(r)
}
