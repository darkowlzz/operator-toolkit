package dag

import (
	"testing"

	"github.com/darkowlzz/composite-reconciler/operate/v1/operand"
)

func TestDAG(t *testing.T) {
	//  +---+    +---+
	//  | A |    | B |
	//  +---+    +---+
	//   ^         ^
	//   |         |
	//   |        +---+
	// +---+----->+ C +<---+
	// | D |      +---+    |
	// +---+               |
	//   ^               +---+
	//   |               | F |
	//   | +---+         +---+
	//   +-+ E |
	//     +---+
	//
	// Expected run order: [A:0 B:0 C:1 D:2 F:2 E:3]

	ops := []*operand.Operand{
		&operand.Operand{
			Name: "A",
		},
		&operand.Operand{
			Name: "B",
		},
		&operand.Operand{
			Name:     "C",
			Requires: []string{"B"},
		},
		&operand.Operand{
			Name:     "D",
			Requires: []string{"A", "C"},
		},
		&operand.Operand{
			Name:     "E",
			Requires: []string{"D"},
		},
		&operand.Operand{
			Name:     "F",
			Requires: []string{"C"},
		},
	}

	expectedResult := `[
  0: [ A B ]
  1: [ C ]
  2: [ D F ]
  3: [ E ]
]`
	expectedReverseResult := `[
  0: [ E ]
  1: [ D F ]
  2: [ C ]
  3: [ A B ]
]`

	opd, err := NewOperandDAG(ops)
	if err != nil {
		t.Fatalf("unexpected error while creating OperandDAG: %v", err)
	}

	ordered, err := opd.Order()
	if err != nil {
		t.Errorf("failed to order the operands: %v", err)
	}
	if ordered.String() != expectedResult {
		t.Errorf("unexpected results:\n\t(WNT) %q\n\t(GOT) %q", expectedResult, ordered)
	}

	reverseOrder := ordered.Reverse()
	if reverseOrder.String() != expectedReverseResult {
		t.Errorf("unexpected reverse results:\n\t(WNT) %q\n\t(GOT) %q", expectedReverseResult, reverseOrder)
	}
}
