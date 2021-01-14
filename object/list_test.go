package object

import (
	"testing"

	tdv1alpha1 "github.com/darkowlzz/operator-toolkit/testdata/api/v1alpha1"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestNamespacedNames(t *testing.T) {

	cases := []struct {
		name      string
		instances client.ObjectList
		want      []types.NamespacedName
		wantErr   bool
	}{
		{
			name: "name and ns",
			instances: &tdv1alpha1.GameList{
				Items: []tdv1alpha1.Game{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "zelda",
							Namespace: "switch",
						},
					},
				},
			},
			want: []types.NamespacedName{
				{
					Name:      "zelda",
					Namespace: "switch",
				},
			},
		},
		{
			name: "name only",
			instances: &tdv1alpha1.GameList{
				Items: []tdv1alpha1.Game{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "hangman",
						},
					},
				},
			},
			want: []types.NamespacedName{
				{
					Name:      "hangman",
					Namespace: "",
				},
			},
		},
		{
			name: "mixed",
			instances: &tdv1alpha1.GameList{
				Items: []tdv1alpha1.Game{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "zelda",
							Namespace: "switch",
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "hangman",
						},
					},
				},
			},
			want: []types.NamespacedName{
				{
					Name:      "zelda",
					Namespace: "switch",
				},
				{
					Name:      "hangman",
					Namespace: "",
				},
			},
		},
		{
			name:      "empty",
			instances: &tdv1alpha1.GameList{},
			want:      []types.NamespacedName{},
		},
		{
			name: "no items",
			instances: &tdv1alpha1.GameList{
				Items: []tdv1alpha1.Game{},
			},
			want: []types.NamespacedName{},
		},
		{
			name:      "nil",
			instances: nil,
			want:      nil,
			wantErr:   true,
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			got, err := NamespacedNames(tc.instances)
			if tc.wantErr {
				assert.Error(t, err)
				assert.Nil(t, got)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tc.want, got, "result")
		})
	}
}

func TestNamespacedNamesDiff(t *testing.T) {
	cases := []struct {
		name string
		a    []types.NamespacedName
		b    []types.NamespacedName
		want []types.NamespacedName
	}{
		{
			name: "same",
			a: []types.NamespacedName{
				{
					Name:      "zelda",
					Namespace: "switch",
				},
			},
			b: []types.NamespacedName{
				{
					Name:      "zelda",
					Namespace: "switch",
				},
			},
			want: []types.NamespacedName{},
		},
		{
			name: "missing in b",
			a: []types.NamespacedName{
				{
					Name:      "zelda",
					Namespace: "switch",
				},
			},
			b: []types.NamespacedName{},
			want: []types.NamespacedName{
				{
					Name:      "zelda",
					Namespace: "switch",
				},
			},
		},
		{
			name: "missing in a",
			a:    []types.NamespacedName{},
			b: []types.NamespacedName{
				{
					Name:      "zelda",
					Namespace: "switch",
				},
			},
			want: []types.NamespacedName{}, // Only care about items from a missing in b.
		},
		{
			name: "multiple",
			a: []types.NamespacedName{
				{
					Name:      "zelda",
					Namespace: "switch",
				},
				{
					Name:      "overwatch",
					Namespace: "blizzard",
				},
				{
					Name:      "lol",
					Namespace: "blizzard",
				},
			},
			b: []types.NamespacedName{
				{
					Name:      "zelda",
					Namespace: "switch",
				},
				{
					Name:      "lol",
					Namespace: "blizzard",
				},
			},
			want: []types.NamespacedName{
				{
					Name:      "overwatch",
					Namespace: "blizzard",
				},
			},
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			got := NamespacedNamesDiff(tc.a, tc.b)
			assert.Equal(t, tc.want, got, "result")
		})
	}
}

func TestClientObjects(t *testing.T) {
	// Create a scheme with testdata scheme info.
	scheme := runtime.NewScheme()
	assert.Nil(t, tdv1alpha1.AddToScheme(scheme))

	type want struct {
		name      string
		namespace string
	}

	tests := []struct {
		name    string
		objs    []runtime.Object
		want    []want
		wantErr bool
	}{
		{
			name: "name and ns",
			objs: []runtime.Object{
				&tdv1alpha1.Game{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "zelda",
						Namespace: "switch",
					},
				},
			},
			want: []want{
				{
					name:      "zelda",
					namespace: "switch",
				},
			},
		},
		{
			name: "list type",
			objs: []runtime.Object{
				&tdv1alpha1.GameList{
					ListMeta: metav1.ListMeta{},
					Items: []tdv1alpha1.Game{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "zelda",
								Namespace: "switch",
							},
						},
						{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "overwatch",
								Namespace: "blizzard",
							},
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "unregistered type",
			objs: []runtime.Object{
				&corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "zelda",
						Namespace: "switch",
					},
				},
			},
			wantErr: true,
		},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			got, err := ClientObjects(scheme, tc.objs)
			if tc.wantErr {
				assert.Error(t, err)
				assert.Nil(t, got)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, len(tc.objs), len(got))
			for i, obj := range got {
				assert.Equal(t, tc.want[i].name, obj.GetName())
				assert.Equal(t, tc.want[i].namespace, obj.GetNamespace())
			}

		})
	}
}
