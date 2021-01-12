/*
Copyright 2021.

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
	"time"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	extobjsyncv1 "github.com/darkowlzz/operator-toolkit/controller/external-object-sync/v1"
	syncv1 "github.com/darkowlzz/operator-toolkit/controller/sync/v1"
	appv1alpha1 "github.com/darkowlzz/operator-toolkit/example/api/v1alpha1"
	"github.com/darkowlzz/operator-toolkit/example/controllers/externalGameSync"
)

// ExternalGameSyncReconciler reconciles a Game object to keep it in sync with
// the corresponding external system object.
type ExternalGameSyncReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme

	extobjsyncv1.Reconciler
}

//+kubebuilder:rbac:groups=app.example.com,resources=games,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=app.example.com,resources=games/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=app.example.com,resources=games/finalizers,verbs=update

// SetupWithManager sets up the controller with the Manager.
func (r *ExternalGameSyncReconciler) SetupWithManager(mgr ctrl.Manager) error {
	c := externalGameSync.NewExternalGameSyncController()

	// Set the garbage collection period and initialize the reconciler,
	r.Reconciler.SetGarbageCollectionPeriod(5 * time.Second)
	// r.Reconciler.DisableGarbageCollector()
	err := r.Reconciler.Init(mgr, &appv1alpha1.Game{}, &appv1alpha1.GameList{},
		syncv1.WithName("external-game-sync-controller"),
		syncv1.WithScheme(r.Scheme),
		syncv1.WithController(c),
		syncv1.WithClient(r.Client),
		syncv1.WithLogger(r.Log),
	)
	if err != nil {
		return fmt.Errorf("failed to create new ExternalObjectSyncReconciler: %w", err)
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&appv1alpha1.Game{}).
		Complete(r)
}
