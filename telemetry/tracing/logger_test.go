package tracing

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/attribute"
)

func TestKeyValues(t *testing.T) {
	cases := []struct {
		name       string
		input      []interface{}
		wantOutput []attribute.KeyValue
	}{
		{
			name:       "empty input",
			wantOutput: []attribute.KeyValue{},
		},
		{
			name:       "nil input",
			input:      nil,
			wantOutput: []attribute.KeyValue{},
		},
		{
			name:  "valid key-val",
			input: []interface{}{"apple", "ball", "cat", 4},
			wantOutput: []attribute.KeyValue{
				attribute.String("apple", "ball"),
				attribute.Int("cat", 4),
			},
		},
		{
			name:  "non-string key",
			input: []interface{}{1, 2},
			wantOutput: []attribute.KeyValue{
				attribute.Int(nonStringKey, 2),
			},
		},
		{
			name:       "invalid input",
			input:      []interface{}{"aa"},
			wantOutput: []attribute.KeyValue{},
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
