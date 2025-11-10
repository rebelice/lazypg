package models

// FilterOperator represents a filter comparison operator
type FilterOperator string

const (
	OpEqual          FilterOperator = "="
	OpNotEqual       FilterOperator = "!="
	OpGreaterThan    FilterOperator = ">"
	OpGreaterOrEqual FilterOperator = ">="
	OpLessThan       FilterOperator = "<"
	OpLessOrEqual    FilterOperator = "<="
	OpLike           FilterOperator = "LIKE"
	OpILike          FilterOperator = "ILIKE"
	OpIn             FilterOperator = "IN"
	OpNotIn          FilterOperator = "NOT IN"
	OpIsNull         FilterOperator = "IS NULL"
	OpIsNotNull      FilterOperator = "IS NOT NULL"
	OpContains       FilterOperator = "@>"  // JSONB contains
	OpContainedBy    FilterOperator = "<@"  // JSONB contained by
	OpHasKey         FilterOperator = "?"   // JSONB has key
	OpArrayOverlap   FilterOperator = "&&"  // Array overlap
)

// FilterCondition represents a single filter condition
type FilterCondition struct {
	Column   string
	Operator FilterOperator
	Value    interface{}
	Type     string // PostgreSQL type (text, integer, jsonb, etc.)
}

// FilterGroup represents a group of conditions with AND/OR logic
type FilterGroup struct {
	Conditions []FilterCondition
	Logic      string // "AND" or "OR"
	Groups     []FilterGroup
}

// Filter represents the complete filter state
type Filter struct {
	RootGroup FilterGroup
	TableName string
	Schema    string
}
