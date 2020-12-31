package dag

import (
	"github.com/goombaio/dag"

	"github.com/darkowlzz/operator-toolkit/operator/v1/operand"
)

// OperandDAG is a directed acyclic graph representation of the opereand
// dependencies. This is used to resolve the dependencies of the operands on
// each other and find an optimal execution path.
type OperandDAG struct {
	*dag.DAG
}

func NewOperandDAG(operands []operand.Operand) (*OperandDAG, error) {
	od := &OperandDAG{DAG: dag.NewDAG()}

	// Create vertices for all the operands.
	for _, op := range operands {
		v := dag.NewVertex(op.Name(), op)
		if err := od.AddVertex(v); err != nil {
			return nil, err
		}
	}

	// Create edges between the vertices based on the operand's depends on
	// property.
	for _, op := range operands {
		headVertex, err := od.GetVertex(op.Name())
		if err != nil {
			return nil, err
		}

		// Connect the operand to all the vertices it depends on.
		for _, dep := range op.Requires() {
			tailVertex, err := od.GetVertex(dep)
			if err != nil {
				return nil, err
			}
			if err := od.AddEdge(tailVertex, headVertex); err != nil {
				return nil, err
			}
		}
	}

	return od, nil
}

func (od *OperandDAG) Order() (operand.OperandOrder, error) {
	soln, steps, err := od.solve()
	if err != nil {
		return nil, err
	}

	result := make([][]operand.Operand, steps)

	for name, step := range soln {
		v, verr := od.GetVertex(name)
		if verr != nil {
			return result, verr
		}
		result[step] = append(result[step], v.Value.(operand.Operand))
	}

	return result, nil
}

// Solve solves the graph traversal in DAG with steps. Returns a map containing
// vertex name with step number and total number of steps in the solution.
func (od *OperandDAG) solve() (map[string]int, int, error) {
	order := map[string]int{}
	// Start from root.
	roots := od.SourceVertices()

	// Init order step and roots.
	step := 0
	newRoots := roots
	var err error
	for len(newRoots) > 0 {
		newRoots, err = od.solveStep(step, newRoots, order)
		if err != nil {
			return nil, step, err
		}
		step++
	}

	return order, step, nil
}

// solveStep takes a step number, current roots and an order, and returns new
// current roots and updates the order.
func (od *OperandDAG) solveStep(step int, currentRoots []*dag.Vertex, order map[string]int) ([]*dag.Vertex, error) {
	newRoots := []*dag.Vertex{}

	for _, c := range currentRoots {
		// Check if the current root exists in the order.
		if _, exists := order[c.ID]; !exists {
			// Check if the predecessors exists in the order. If not, skip,
			// else, add to order.
			pp, perr := od.Predecessors(c)
			if perr != nil {
				return nil, perr
			}

			if len(pp) == 0 {
				// If no predecessor, add to order, it's the root.
				order[c.ID] = step
				var serr error
				newRoots, serr = od.addSuccessorsToNewRoots(c, newRoots)
				if serr != nil {
					return nil, serr
				}
				continue
			}

			satisfied := true
			for _, p := range pp {
				if _, exists := order[p.ID]; !exists {
					satisfied = false
				}
			}

			// Satisfied, then add to order.
			if satisfied {
				order[c.ID] = step
			}
		}

		// Get successors and add to new roots.
		var serr error
		newRoots, serr = od.addSuccessorsToNewRoots(c, newRoots)
		if serr != nil {
			return nil, serr
		}
	}

	return newRoots, nil
}

// addSuccessorsToNewRoots takes a vertex, fetches its successors and adds the
// successors to the newRoots list. This is used to create a list of all the
// adjacent vertices at the same level in the graph.
func (od *OperandDAG) addSuccessorsToNewRoots(v *dag.Vertex, newRoots []*dag.Vertex) ([]*dag.Vertex, error) {
	ss, serr := od.Successors(v)
	if serr != nil {
		return nil, serr
	}

	// Add to root if not exists.
	for _, s := range ss {
		if !od.vertexExists(newRoots, s) {
			newRoots = append(newRoots, s)
		}
	}

	return newRoots, nil
}

func (od *OperandDAG) vertexExists(vs []*dag.Vertex, target *dag.Vertex) bool {
	for _, v := range vs {
		if v.ID == target.ID {
			return true
		}
	}
	return false
}
