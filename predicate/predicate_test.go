package predicate

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
)

func TestFinalizerChangedPredicate(t *testing.T) {
	getConfigMapWithFinalizers := func(f []string) *corev1.ConfigMap {
		return &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Finalizers: f,
			},
		}
	}

	tests := []struct {
		name   string
		oldObj client.Object
		newObj client.Object
		want   bool
	}{
		{
			name:   "old object nil",
			oldObj: nil,
			newObj: getConfigMapWithFinalizers([]string{"foo"}),
			want:   false,
		},
		{
			name:   "new object nil",
			oldObj: getConfigMapWithFinalizers([]string{"foo"}),
			newObj: nil,
			want:   false,
		},
		{
			name:   "same finalizers",
			oldObj: getConfigMapWithFinalizers([]string{"foo", "bar"}),
			newObj: getConfigMapWithFinalizers([]string{"foo", "bar"}),
			want:   false,
		},
		{
			name:   "different finalizers",
			oldObj: getConfigMapWithFinalizers([]string{"foo", "bar"}),
			newObj: getConfigMapWithFinalizers([]string{"foo", "bar", "baz"}),
			want:   true,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			f := FinalizerChangedPredicate{}
			e := event.UpdateEvent{
				ObjectOld: tc.oldObj,
				ObjectNew: tc.newObj,
			}
			assert.Equal(t, tc.want, f.Update(e))
		})
	}
}
