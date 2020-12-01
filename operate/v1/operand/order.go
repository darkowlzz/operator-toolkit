package operand

import (
	"fmt"
	"sort"
	"strings"
)

// OperandOrder stores the operands in order of their execution. The first
// dimension of the slice depicts the execution step and the second dimention
// contains the operands that can be run in parallel.
type OperandOrder [][]*Operand

// String implements the Stringer interface for OperandOrder.
// Example string result:
// [
//  0: [ A B ]
//  1: [ C ]
//  2: [ D F ]
//  3: [ E ]
// ]
func (o OperandOrder) String() string {
	var result strings.Builder
	result.WriteString("[\n")

	for i, s := range o {
		// Sort the items for deterministic results.
		items := []string{}
		for _, op := range s {
			items = append(items, op.Name)
		}
		sort.Strings(items)
		itemsStr := strings.Join(items, " ")
		line := fmt.Sprintf("  %d: [ %s ]\n", i, itemsStr)
		result.WriteString(line)
	}
	result.WriteString("]")
	return result.String()
}
