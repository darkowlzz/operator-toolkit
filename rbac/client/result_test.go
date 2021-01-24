package client

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	rbacv1 "k8s.io/api/rbac/v1"
)

func TestReorderRules(t *testing.T) {
	rules := []rbacv1.PolicyRule{
		{
			APIGroups: []string{"app.example.com"},
			Resources: []string{"database"},
			Verbs:     []string{"get"},
		},
		{
			APIGroups: []string{"app.example.com"},
			Resources: []string{"database"},
			Verbs:     []string{"create"},
		},
		{
			APIGroups: []string{"app.example.com"},
			Resources: []string{"database"},
			Verbs:     []string{"list"},
		},
		{
			APIGroups: []string{"app.example.com"},
			Resources: []string{"database/status"},
			Verbs:     []string{"update"},
		},
		{
			APIGroups: []string{"app.example.com"},
			Resources: []string{"database/status"},
			Verbs:     []string{"patch"},
		},
		{
			APIGroups: []string{"api.example.com"},
			Resources: []string{"auth"},
			Verbs:     []string{"get"},
		},
		{
			APIGroups: []string{"api.example.com"},
			Resources: []string{"auth"},
			Verbs:     []string{"list"},
		},
	}

	wantResult := `
---
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  creationTimestamp: null
  name: test-role
rules:
- apiGroups:
  - app.example.com
  resources:
  - database
  verbs:
  - get
  - create
  - list
- apiGroups:
  - app.example.com
  resources:
  - database/status
  verbs:
  - update
  - patch
- apiGroups:
  - api.example.com
  resources:
  - auth
  verbs:
  - get
  - list
`

	orderedRules := reorderRules(rules)

	// Create a new rule, add the rules result to it and write the result.
	role := newRole("test-role")
	role.Rules = orderedRules

	var res bytes.Buffer

	assert.Nil(t, WriteResult(role, &res))

	assert.Equal(t, wantResult, res.String())
}
