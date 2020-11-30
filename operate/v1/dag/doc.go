// Package dag helps create a directed asyclic graph(DAG) of the operands in an
// operator based on the relationship between the operands. It helps order the
// execution sequence of the operands. Since the DAG doesn't change usually, it
// is usually created when the operator initializes and the same graph is
// reused.
package dag
