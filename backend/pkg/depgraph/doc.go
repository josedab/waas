// Package depgraph provides webhook dependency graph mapping and impact analysis.
//
// It automatically maps producer→consumer relationships by analyzing delivery
// history, builds adjacency graphs, and computes transitive closure for blast
// radius impact analysis.
package depgraph
