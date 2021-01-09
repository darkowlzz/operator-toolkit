package externalGameSync

import (
	"context"

	syncv1 "github.com/darkowlzz/operator-toolkit/controller/sync/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ExternalGameSyncController implements the sync controller interface.
type ExternalGameSyncController struct{}

var _ syncv1.Controller = &ExternalGameSyncController{}

// TODO: Implement fake API that the controller calls.

func (c *ExternalGameSyncController) Ensure(context.Context, client.Object) error { return nil }
func (c *ExternalGameSyncController) Delete(context.Context, client.Object) error { return nil }
func (c *ExternalGameSyncController) List(context.Context) ([]types.NamespacedName, error) {
	return nil, nil
}

func NewExternalGameSyncController() *ExternalGameSyncController {
	return &ExternalGameSyncController{}
}
