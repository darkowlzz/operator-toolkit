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

	compositev1 "github.com/darkowlzz/operator-toolkit/controller/composite/v1"
	"github.com/darkowlzz/operator-toolkit/declarative/loader"
	appv1alpha1 "github.com/darkowlzz/operator-toolkit/example/api/v1alpha1"
	"github.com/darkowlzz/operator-toolkit/example/controllers/game"
	"github.com/darkowlzz/operator-toolkit/operator/v1/executor"
)

// GameReconciler reconciles a Game object
type GameReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme

	compositev1.CompositeReconciler
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
	// Load manifests in an in-memory filesystem.
	fs, err := loader.NewLoadedManifestFileSystem("channels", "stable")
	if err != nil {
		return fmt.Errorf("failed to create loaded ManifestFileSystem: %w", err)
	}

	// TODO: Expose the executor strategy option via SetupWithManager.
	gc, err := game.NewGameController(mgr, fs, executor.Parallel)
	if err != nil {
		return err
	}

	// Initialize the reconciler.
	err = r.CompositeReconciler.Init(mgr, &appv1alpha1.Game{},
		compositev1.WithName("game-controller"),
		compositev1.WithController(gc),
		compositev1.WithCleanupStrategy(compositev1.OwnerReferenceCleanup),
		compositev1.WithInitCondition(compositev1.DefaultInitCondition),
		compositev1.WithLogger(r.Log),
	)
	if err != nil {
		return fmt.Errorf("failed to create new CompositeReconciler: %w", err)
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&appv1alpha1.Game{}).
		Complete(r)
}
