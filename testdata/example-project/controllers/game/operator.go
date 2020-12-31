package game

import (
	"fmt"

	"github.com/darkowlzz/composite-reconciler/declarative/loader"
	operatorv1 "github.com/darkowlzz/composite-reconciler/operator/v1"
	"github.com/darkowlzz/composite-reconciler/operator/v1/executor"
	"github.com/darkowlzz/composite-reconciler/operator/v1/operand"
	ctrl "sigs.k8s.io/controller-runtime"
)

// NewOperator creates and returns a CompositeOperator with all the operands
// configured.
func NewOperator(mgr ctrl.Manager, fs *loader.ManifestFileSystem, execStrategy executor.ExecutionStrategy) (*operatorv1.CompositeOperator, error) {
	// Create the operands.
	configmapOp := NewConfigmapOperand("configmap-operand", mgr.GetClient(), []string{}, operand.RequeueOnError, fs)

	// Create and return CompositeOperator.
	return operatorv1.NewCompositeOperator(
		operatorv1.WithEventRecorder(mgr.GetEventRecorderFor("game-controller")),
		operatorv1.WithExecutionStrategy(execStrategy),
		operatorv1.WithOperands(configmapOp),
	)
}

// NewGameController creates an Operator and a GameController that uses the
// created operator, and returns the GameController.
func NewGameController(mgr ctrl.Manager, fs *loader.ManifestFileSystem, execStrategy executor.ExecutionStrategy) (*GameController, error) {
	operator, err := NewOperator(mgr, fs, execStrategy)
	if err != nil {
		return nil, fmt.Errorf("failed to create a new operator: %w", err)
	}
	return &GameController{Operator: operator}, nil
}
