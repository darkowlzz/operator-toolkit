package rbac

import (
	"fmt"
	"io"

	"github.com/pkg/errors"
	rbacv1 "k8s.io/api/rbac/v1"
	"sigs.k8s.io/yaml"
)

const (
	groupResourceSeparator = "_"
	yamlSeparator          = "\n---\n"
)

// Result marshals and writes the observed RBAC rules into a given Writer. It
// also writes any observed error into a given error writer.
func Result(c *RBACClient, manifestWriter io.Writer, errorWriter io.Writer) error {
	// Reorder the rules.
	c.Role.Rules = reorderRules(c.Role.Rules)
	c.ClusterRole.Rules = reorderRules(c.ClusterRole.Rules)

	// Roles to parse.
	roles := []interface{}{c.Role, c.ClusterRole}

	// Iterate through the roles and write them as yaml manifests.
	for _, role := range roles {
		if err := WriteResult(role, manifestWriter); err != nil {
			return err
		}
	}

	// Write the errors to the error writer.
	if errorWriter != nil && len(c.errors) > 0 {
		_, err := errorWriter.Write([]byte("Errors during RBACClient recording:\n"))
		if err != nil {
			return errors.Wrap(err, "failed to write to RBACClient errorWriter")
		}

		for _, cErr := range c.errors {
			_, err := errorWriter.Write([]byte(cErr.Error()))
			if err != nil {
				return errors.Wrap(err, "failed to write RBACClient recorder errors")
			}
		}
	}

	return nil
}

// WriteResult writes a given result into a given writer.
func WriteResult(role interface{}, writer io.Writer) error {
	m, err := yaml.Marshal(role)
	if err != nil {
		return errors.Wrap(err, "failed to marshal role")
	}

	_, err = writer.Write([]byte(yamlSeparator))
	if err != nil {
		return errors.Wrap(err, "failed to write yaml separator")
	}

	_, err = writer.Write(m)
	if err != nil {
		return errors.Wrap(err, "failed to write RBAC manifest")
	}

	return nil
}

// reorderRules reorders the rules and groups them together where possible.
func reorderRules(rules []rbacv1.PolicyRule) []rbacv1.PolicyRule {
	result := []rbacv1.PolicyRule{}

	// rulesByGroupResource contains the policy rules ordered based on their
	// group and resource names.
	rulesByGroupResource := map[string]rbacv1.PolicyRule{}

	// groupResourceNameList is a list of all the group-resource. This is used
	// to read the rulesByGroupResource map in a deterministic order.
	groupResourceNameList := []string{}

	for _, rule := range rules {
		// The rules collected by the rbac recorder contains only one group.
		group := rule.APIGroups[0]
		for _, resource := range rule.Resources {
			groupResourceName := fmt.Sprintf("%s%s%s", group, groupResourceSeparator, resource)
			if prule, exists := rulesByGroupResource[groupResourceName]; exists {
				// Append the verb.
				for _, verb := range rule.Verbs {
					// Check if the verb already exists.
					if !contains(prule.Verbs, verb) {
						prule.Verbs = append(prule.Verbs, verb)
						rulesByGroupResource[groupResourceName] = prule
					}
				}
			} else {
				// Add the new group-resource in the group-resource name list.
				groupResourceNameList = append(groupResourceNameList, groupResourceName)
				// Create a new policy rule for the current rule.
				rulesByGroupResource[groupResourceName] = rbacv1.PolicyRule{
					APIGroups: []string{group},
					Resources: []string{resource},
					Verbs:     rule.Verbs,
				}
			}
		}
	}

	// Conver the map of group resource into a list of result policy rules.
	for _, groupResourceName := range groupResourceNameList {
		policyRule := rulesByGroupResource[groupResourceName]
		result = append(result, policyRule)
	}

	return result
}

func contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}
