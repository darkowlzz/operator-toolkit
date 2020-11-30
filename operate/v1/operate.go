package v1

import (
	"github.com/goombaio/dag"
	ctrl "sigs.k8s.io/controller-runtime"

	eventv1 "github.com/darkowlzz/composite-reconciler/event/v1"
	"github.com/darkowlzz/composite-reconciler/operate/v1/operand"
)

// Operator is the operator interface that can be implemented by an operator to
// use the composite operator lifecycle.
type Operator interface {
	// IsSuspended tells if an operator is suspended and should not run any
	// operation.
	IsSuspended() bool

	// Operate runs all the operands in order defined by their dependencies.
	Operate() (result ctrl.Result, event eventv1.ReconcilerEvent, err error)
}

type CompositeOperator struct {
	Operands []operand.Operand
}

func (co *CompositeOperator) Order() []operand.Operand {
	operands := []operand.Operand{}

	// Create new DAG.
	opDag := dag.NewDAG()
	// Map operand name and vertex.
	vertices := map[string]*dag.Vertex{}
	for _, operand := range co.Operands {
		// TODO: Ensure no two vertices have the same name.
		v := dag.NewVertex(operand.Name, operand)
		vertices[operand.Name] = v
		opDag.AddVertex(v)
	}

	// for name, vertex := range vertices {
	for k, v := range vertices {
		op := v.Value.(Operand)
		vert := nil
		for _, dep := range op.DependsOn {
			vert = v[dep]
		}
		opDag.AddEdge(v, vert)
	}

	return operands
}
