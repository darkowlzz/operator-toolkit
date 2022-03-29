package dag

import (
	"testing"

	"github.com/golang/mock/gomock"

	"github.com/darkowlzz/operator-toolkit/operator/v1/operand"
	"github.com/darkowlzz/operator-toolkit/operator/v1/operand/mocks"
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

	// Set up mock operands.
	mctrl := gomock.NewController(t)
	defer mctrl.Finish()

	mA := mocks.NewMockOperand(mctrl)
	mA.EXPECT().Name().Return("A").AnyTimes()
	mA.EXPECT().Requires().Return([]string{})

	mB := mocks.NewMockOperand(mctrl)
	mB.EXPECT().Name().Return("B").AnyTimes()
	mB.EXPECT().Requires().Return([]string{})

	mC := mocks.NewMockOperand(mctrl)
	mC.EXPECT().Name().Return("C").AnyTimes()
	mC.EXPECT().Requires().Return([]string{"B"})

	mD := mocks.NewMockOperand(mctrl)
	mD.EXPECT().Name().Return("D").AnyTimes()
	mD.EXPECT().Requires().Return([]string{"A", "C"})

	mE := mocks.NewMockOperand(mctrl)
	mE.EXPECT().Name().Return("E").AnyTimes()
	mE.EXPECT().Requires().Return([]string{"D"})

	mF := mocks.NewMockOperand(mctrl)
	mF.EXPECT().Name().Return("F").AnyTimes()
	mF.EXPECT().Requires().Return([]string{"C"})

	ops := []operand.Operand{mA, mB, mC, mD, mE, mF}

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

	if ordered.String() != expectedResult {
		t.Errorf("unexpected results after reverse:\n\t(WNT) %q\n\t(GOT) %q", expectedResult, ordered)
	}
}
