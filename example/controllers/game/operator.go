package game

import (
	"fmt"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/kubebuilder-declarative-pattern/pkg/patterns/declarative"
	"sigs.k8s.io/kustomize/api/filesys"

	operatorv1 "github.com/darkowlzz/operator-toolkit/operator/v1"
	"github.com/darkowlzz/operator-toolkit/operator/v1/executor"
	"github.com/darkowlzz/operator-toolkit/operator/v1/operand"
)

// NewOperator creates and returns a CompositeOperator with all the operands
// configured.
func NewOperator(mgr ctrl.Manager, fs filesys.FileSystem, execStrategy executor.ExecutionStrategy, labelMaker declarative.LabelMaker) (*operatorv1.CompositeOperator, error) {
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
func NewGameController(mgr ctrl.Manager, fs filesys.FileSystem, execStrategy executor.ExecutionStrategy, labelMaker declarative.LabelMaker) (*GameController, error) {
	operator, err := NewOperator(mgr, fs, execStrategy, labelMaker)
	if err != nil {
		return nil, fmt.Errorf("failed to create a new operator: %w", err)
	}
	return &GameController{Operator: operator}, nil
}
