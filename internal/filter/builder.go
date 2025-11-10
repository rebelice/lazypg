package filter

import (
	"fmt"
	"strings"

	"github.com/rebeliceyang/lazypg/internal/models"
)

// Builder generates SQL WHERE clauses from Filter models
type Builder struct{}

// NewBuilder creates a new filter builder
func NewBuilder() *Builder {
	return &Builder{}
}

// BuildWhere generates a WHERE clause from a Filter
func (b *Builder) BuildWhere(filter models.Filter) (string, []interface{}, error) {
	if len(filter.RootGroup.Conditions) == 0 && len(filter.RootGroup.Groups) == 0 {
		return "", nil, nil
	}

	clause, args, err := b.buildGroup(filter.RootGroup, 1)
	if err != nil {
		return "", nil, err
	}

	return "WHERE " + clause, args, nil
}

// buildGroup recursively builds a filter group
func (b *Builder) buildGroup(group models.FilterGroup, paramIndex int) (string, []interface{}, error) {
	var clauses []string
	var args []interface{}
	currentParam := paramIndex

	// Build conditions
	for _, cond := range group.Conditions {
		clause, condArgs, err := b.buildCondition(cond, currentParam)
		if err != nil {
			return "", nil, err
		}
		clauses = append(clauses, clause)
		args = append(args, condArgs...)
		currentParam += len(condArgs)
	}

	// Build sub-groups
	for _, subGroup := range group.Groups {
		clause, groupArgs, err := b.buildGroup(subGroup, currentParam)
		if err != nil {
			return "", nil, err
		}
		clauses = append(clauses, "("+clause+")")
		args = append(args, groupArgs...)
		currentParam += len(groupArgs)
	}

	logic := group.Logic
	if logic == "" {
		logic = "AND"
	}

	return strings.Join(clauses, " "+logic+" "), args, nil
}

// buildCondition builds a single filter condition
func (b *Builder) buildCondition(cond models.FilterCondition, paramIndex int) (string, []interface{}, error) {
	column := cond.Column

	switch cond.Operator {
	case models.OpIsNull:
		return fmt.Sprintf("%s IS NULL", column), nil, nil
	case models.OpIsNotNull:
		return fmt.Sprintf("%s IS NOT NULL", column), nil, nil
	case models.OpEqual, models.OpNotEqual, models.OpGreaterThan, models.OpGreaterOrEqual,
		models.OpLessThan, models.OpLessOrEqual:
		return fmt.Sprintf("%s %s $%d", column, cond.Operator, paramIndex), []interface{}{cond.Value}, nil
	case models.OpLike, models.OpILike:
		return fmt.Sprintf("%s %s $%d", column, cond.Operator, paramIndex), []interface{}{cond.Value}, nil
	case models.OpIn, models.OpNotIn:
		// For IN/NOT IN, value should be a slice
		return fmt.Sprintf("%s %s ($%d)", column, cond.Operator, paramIndex), []interface{}{cond.Value}, nil
	case models.OpContains, models.OpContainedBy, models.OpHasKey:
		// JSONB operators
		return fmt.Sprintf("%s %s $%d", column, cond.Operator, paramIndex), []interface{}{cond.Value}, nil
	case models.OpArrayOverlap:
		return fmt.Sprintf("%s && $%d", column, paramIndex), []interface{}{cond.Value}, nil
	default:
		return "", nil, fmt.Errorf("unsupported operator: %s", cond.Operator)
	}
}

// GetOperatorsForType returns available operators for a given PostgreSQL type
func GetOperatorsForType(dataType string) []models.FilterOperator {
	switch {
	case strings.Contains(dataType, "int") || strings.Contains(dataType, "numeric") ||
		strings.Contains(dataType, "real") || strings.Contains(dataType, "double"):
		return []models.FilterOperator{
			models.OpEqual, models.OpNotEqual,
			models.OpGreaterThan, models.OpGreaterOrEqual,
			models.OpLessThan, models.OpLessOrEqual,
			models.OpIsNull, models.OpIsNotNull,
		}
	case strings.Contains(dataType, "char") || strings.Contains(dataType, "text"):
		return []models.FilterOperator{
			models.OpEqual, models.OpNotEqual,
			models.OpLike, models.OpILike,
			models.OpIsNull, models.OpIsNotNull,
		}
	case strings.Contains(dataType, "jsonb"):
		return []models.FilterOperator{
			models.OpEqual, models.OpNotEqual,
			models.OpContains, models.OpContainedBy, models.OpHasKey,
			models.OpIsNull, models.OpIsNotNull,
		}
	case strings.Contains(dataType, "ARRAY"):
		return []models.FilterOperator{
			models.OpEqual, models.OpNotEqual,
			models.OpArrayOverlap, models.OpContains, models.OpContainedBy,
			models.OpIsNull, models.OpIsNotNull,
		}
	case strings.Contains(dataType, "bool"):
		return []models.FilterOperator{
			models.OpEqual, models.OpNotEqual,
			models.OpIsNull, models.OpIsNotNull,
		}
	case strings.Contains(dataType, "date") || strings.Contains(dataType, "time"):
		return []models.FilterOperator{
			models.OpEqual, models.OpNotEqual,
			models.OpGreaterThan, models.OpGreaterOrEqual,
			models.OpLessThan, models.OpLessOrEqual,
			models.OpIsNull, models.OpIsNotNull,
		}
	default:
		return []models.FilterOperator{
			models.OpEqual, models.OpNotEqual,
			models.OpIsNull, models.OpIsNotNull,
		}
	}
}
