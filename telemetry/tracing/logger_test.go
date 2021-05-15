package tracing

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/label"
)

func TestKeyValues(t *testing.T) {
	cases := []struct {
		name       string
		input      []interface{}
		wantOutput []label.KeyValue
	}{
		{
			name:       "empty input",
			wantOutput: []label.KeyValue{},
		},
		{
			name:       "nil input",
			input:      nil,
			wantOutput: []label.KeyValue{},
		},
		{
			name:  "valid key-val",
			input: []interface{}{"apple", "ball", "cat", 4},
			wantOutput: []label.KeyValue{
				label.String("apple", "ball"),
				label.Int("cat", 4),
			},
		},
		{
			name:  "non-string key",
			input: []interface{}{1, 2},
			wantOutput: []label.KeyValue{
				label.Int(nonStringKey, 2),
			},
		},
		{
			name:       "invalid input",
			input:      []interface{}{"aa"},
			wantOutput: []label.KeyValue{},
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			result := keyValues(tc.input...)
			assert.Equal(t, tc.wantOutput, result)
		})
	}
}
