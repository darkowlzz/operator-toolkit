package object

import (
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"

	tdv1alpha1 "github.com/darkowlzz/operator-toolkit/testdata/api/v1alpha1"
)

func TestGetUnstructuredObject(t *testing.T) {
	// Create a scheme with testdata scheme info.
	scheme := runtime.NewScheme()
	assert.Nil(t, tdv1alpha1.AddToScheme(scheme))

	cases := []struct {
		name string
		obj  runtime.Object
		want *unstructured.Unstructured

		wantErr bool
	}{
		{
			name: "empty object",
			obj:  &tdv1alpha1.Game{},
			want: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": "app.example.com/v1alpha1",
					"kind":       "Game",
					"metadata": map[string]interface{}{
						"creationTimestamp": nil,
					},
					"spec":   map[string]interface{}{},
					"status": map[string]interface{}{},
				},
			},
		},
		{
			name: "object with metadata",
			obj: &tdv1alpha1.Game{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "zelda",
					Namespace: "switch",
				},
			},
			want: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": "app.example.com/v1alpha1",
					"kind":       "Game",
					"metadata": map[string]interface{}{
						"name":              "zelda",
						"namespace":         "switch",
						"creationTimestamp": nil,
					},
					"spec":   map[string]interface{}{},
					"status": map[string]interface{}{},
				},
			},
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			got, err := GetUnstructuredObject(scheme, tc.obj)
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

func TestIsInitialized(t *testing.T) {
	// Create a scheme with testdata scheme info.
	scheme := runtime.NewScheme()
	assert.Nil(t, tdv1alpha1.AddToScheme(scheme))

	cases := []struct {
		name    string
		obj     runtime.Object
		want    bool
		wantErr bool
	}{
		{
			name: "empty object",
			obj:  &tdv1alpha1.Game{},
			want: false,
		},
		{
			name: "no status",
			obj: &tdv1alpha1.Game{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "zelda",
					Namespace: "switch",
				},
			},
			want: false,
		},
		{
			name: "no status conditions",
			obj: &tdv1alpha1.Game{
				Status: tdv1alpha1.GameStatus{
					Conditions: []metav1.Condition{},
				},
			},
			want: false,
		},
		{
			name: "one status condition",
			obj: &tdv1alpha1.Game{
				Status: tdv1alpha1.GameStatus{
					Conditions: []metav1.Condition{
						{
							Type: "WaitingForPlayer",
						},
					},
				},
			},
			want: true,
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			got, err := IsInitialized(scheme, tc.obj)
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

func TestGetObjectStatus(t *testing.T) {
	cases := []struct {
		name    string
		obj     map[string]interface{}
		want    map[string]interface{}
		wantErr bool
	}{
		{
			name: "empty object",
			obj: map[string]interface{}{
				"apiVersion": "v1alpha1",
				"kind":       "Game",
				"metadata": map[string]interface{}{
					"creationTimestamp": nil,
				},
				"spec":   map[string]interface{}{},
				"status": map[string]interface{}{},
			},
			want: map[string]interface{}{},
		},
		{
			name: "no status",
			obj: map[string]interface{}{
				"apiVersion": "v1alpha1",
				"kind":       "Game",
				"metadata": map[string]interface{}{
					"creationTimestamp": nil,
				},
				"spec": map[string]interface{}{},
			},
			wantErr: true,
		},
		{
			name: "invalid status",
			obj: map[string]interface{}{
				"apiVersion": "v1alpha1",
				"kind":       "Game",
				"metadata": map[string]interface{}{
					"creationTimestamp": nil,
				},
				"spec":   map[string]interface{}{},
				"status": "should-not-be-a-string",
			},
			wantErr: true,
		},
		{
			name: "no status conditions",
			obj: map[string]interface{}{
				"apiVersion": "v1alpha1",
				"kind":       "Game",
				"metadata": map[string]interface{}{
					"creationTimestamp": nil,
				},
				"spec": map[string]interface{}{},
				"status": map[string]interface{}{
					"conditions": []map[string]interface{}{},
				},
			},
			want: map[string]interface{}{
				"conditions": []map[string]interface{}{},
			},
		},
		{
			name: "single status conditions",
			obj: map[string]interface{}{
				"apiVersion": "v1alpha1",
				"kind":       "Game",
				"metadata": map[string]interface{}{
					"creationTimestamp": nil,
				},
				"spec": map[string]interface{}{},
				"status": map[string]interface{}{
					"conditions": []map[string]interface{}{
						{
							"type": "WaitingForPlayer",
						},
					},
				},
			},
			want: map[string]interface{}{
				"conditions": []map[string]interface{}{
					{
						"type": "WaitingForPlayer",
					},
				},
			},
		},
		{
			name: "multiple status conditions",
			obj: map[string]interface{}{
				"apiVersion": "v1alpha1",
				"kind":       "Game",
				"metadata": map[string]interface{}{
					"creationTimestamp": nil,
				},
				"spec": map[string]interface{}{},
				"status": map[string]interface{}{
					"conditions": []map[string]interface{}{
						{
							"type": "WaitingForPlayer",
						},
						{
							"type": "Online",
						},
					},
				},
			},
			want: map[string]interface{}{
				"conditions": []map[string]interface{}{
					{
						"type": "WaitingForPlayer",
					},
					{
						"type": "Online",
					},
				},
			},
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			got, err := GetObjectStatus(tc.obj)
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

func TestStatusChanged(t *testing.T) {
	// Create a scheme with testdata scheme info.
	scheme := runtime.NewScheme()
	assert.Nil(t, tdv1alpha1.AddToScheme(scheme))

	cases := []struct {
		name string
		oldo runtime.Object
		newo runtime.Object
		want bool
	}{
		{
			name: "no status",
			oldo: &tdv1alpha1.Game{},
			newo: &tdv1alpha1.Game{},
			want: false,
		},
		{
			name: "no change",
			oldo: &tdv1alpha1.Game{
				Status: tdv1alpha1.GameStatus{
					Conditions: []metav1.Condition{
						{
							Type: "WaitingForPlayer",
						},
					},
				},
			},
			newo: &tdv1alpha1.Game{
				Status: tdv1alpha1.GameStatus{
					Conditions: []metav1.Condition{
						{
							Type: "WaitingForPlayer",
						},
					},
				},
			},
			want: false,
		},
		{
			name: "status added",
			oldo: &tdv1alpha1.Game{},
			newo: &tdv1alpha1.Game{
				Status: tdv1alpha1.GameStatus{
					Conditions: []metav1.Condition{
						{
							Type: "Booting",
						},
					},
				},
			},
			want: true,
		},
		{
			name: "status removed",
			oldo: &tdv1alpha1.Game{
				Status: tdv1alpha1.GameStatus{
					Conditions: []metav1.Condition{
						{
							Type: "SelfDestructing",
						},
					},
				},
			},
			newo: &tdv1alpha1.Game{},
			want: true,
		},
		{
			name: "add additional",
			oldo: &tdv1alpha1.Game{
				Status: tdv1alpha1.GameStatus{
					Conditions: []metav1.Condition{
						{
							Type: "WaitingForPlayer",
						},
					},
				},
			},
			newo: &tdv1alpha1.Game{
				Status: tdv1alpha1.GameStatus{
					Conditions: []metav1.Condition{
						{
							Type: "WaitingForPlayer",
						},
						{
							Type: "InstallingUpdate",
						},
					},
				},
			},
			want: true,
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			got, err := StatusChanged(scheme, tc.oldo, tc.newo)
			// Not testing error conditions as the type safety should not make
			// them possible.
			assert.NoError(t, err)
			assert.Equal(t, tc.want, got, "result")
		})
	}
}

func TestNestedFieldNoCopy(t *testing.T) {
	cases := []struct {
		name      string
		obj       map[string]interface{}
		fields    []string
		wantData  interface{}
		wantFound bool
		wantErr   bool
	}{
		{
			name:      "not found",
			obj:       map[string]interface{}{},
			fields:    []string{"apiVersion"},
			wantData:  nil,
			wantFound: false,
		},
		{
			name: "string",
			obj: map[string]interface{}{
				"apiVersion": "v1alpha1",
			},
			fields:    []string{"apiVersion"},
			wantData:  "v1alpha1",
			wantFound: true,
		},
		{
			name: "struct",
			obj: map[string]interface{}{
				"metadata": map[string]interface{}{
					"name":      "zelda",
					"namespace": "switch",
				},
			},
			fields: []string{"metadata"},
			wantData: map[string]interface{}{
				"name":      "zelda",
				"namespace": "switch",
			},
			wantFound: true,
		},
		{
			name: "nested string",
			obj: map[string]interface{}{
				"metadata": map[string]interface{}{
					"name": "zelda",
				},
			},
			fields:    []string{"metadata", "name"},
			wantData:  "zelda",
			wantFound: true,
		},
		{
			name: "bad nesting",
			obj: map[string]interface{}{
				"apiVersion": "v1alpha1",
				"metadata": map[string]interface{}{
					"name": "zelda",
				},
			},
			fields:    []string{"apiVersion", "metadata"},
			wantData:  nil,
			wantFound: false,
			wantErr:   true,
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			got, found, err := NestedFieldNoCopy(tc.obj, tc.fields...)
			if tc.wantErr {
				assert.Error(t, err)
				assert.False(t, found)
				assert.Nil(t, got)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tc.wantFound, found, "found")
			assert.Equal(t, tc.wantData, got, "result")
		})
	}
}
