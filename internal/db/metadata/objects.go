package metadata

import (
	"context"
	"fmt"

	"github.com/rebelice/lazypg/internal/db/connection"
)

// MaterializedView represents a PostgreSQL materialized view
type MaterializedView struct {
	Schema string
	Name   string
}

// Function represents a PostgreSQL function
type Function struct {
	Schema    string
	Name      string
	Arguments string // e.g., "(id integer, name text)"
}

// Procedure represents a PostgreSQL procedure (PG11+)
type Procedure struct {
	Schema    string
	Name      string
	Arguments string
}

// TriggerFunction represents a PostgreSQL trigger function
type TriggerFunction struct {
	Schema string
	Name   string
}

// Sequence represents a PostgreSQL sequence
type Sequence struct {
	Schema     string
	Name       string
	StartValue int64
	MinValue   int64
	MaxValue   int64
	Increment  int64
	Cycle      bool
}

// Index represents a PostgreSQL index
type Index struct {
	Schema     string
	Table      string
	Name       string
	Definition string
}

// Trigger represents a PostgreSQL trigger
type Trigger struct {
	Schema     string
	Table      string
	Name       string
	Definition string
}

// Extension represents a PostgreSQL extension
type Extension struct {
	Name    string
	Version string
	Schema  string
}

// CompositeType represents a PostgreSQL composite type
type CompositeType struct {
	Schema string
	Name   string
}

// EnumType represents a PostgreSQL enum type
type EnumType struct {
	Schema string
	Name   string
	Labels []string
}

// DomainType represents a PostgreSQL domain type
type DomainType struct {
	Schema   string
	Name     string
	BaseType string
}

// RangeType represents a PostgreSQL range type
type RangeType struct {
	Schema  string
	Name    string
	Subtype string
}

// ListMaterializedViews returns all materialized views in a schema
func ListMaterializedViews(ctx context.Context, pool *connection.Pool, schema string) ([]MaterializedView, error) {
	query := `
		SELECT schemaname, matviewname
		FROM pg_matviews
		WHERE schemaname = $1
		ORDER BY matviewname;
	`

	rows, err := pool.Query(ctx, query, schema)
	if err != nil {
		return nil, err
	}

	views := make([]MaterializedView, 0, len(rows))
	for _, row := range rows {
		views = append(views, MaterializedView{
			Schema: toString(row["schemaname"]),
			Name:   toString(row["matviewname"]),
		})
	}

	return views, nil
}

// ListFunctions returns all regular functions in a schema (excluding trigger functions and procedures)
func ListFunctions(ctx context.Context, pool *connection.Pool, schema string) ([]Function, error) {
	query := `
		SELECT p.proname, pg_get_function_identity_arguments(p.oid) as args
		FROM pg_proc p
		JOIN pg_namespace n ON p.pronamespace = n.oid
		WHERE n.nspname = $1
		  AND p.prokind = 'f'
		  AND p.prorettype != 'trigger'::regtype
		ORDER BY p.proname;
	`

	rows, err := pool.Query(ctx, query, schema)
	if err != nil {
		return nil, err
	}

	functions := make([]Function, 0, len(rows))
	for _, row := range rows {
		functions = append(functions, Function{
			Schema:    schema,
			Name:      toString(row["proname"]),
			Arguments: toString(row["args"]),
		})
	}

	return functions, nil
}

// ListProcedures returns all procedures in a schema (PG11+)
func ListProcedures(ctx context.Context, pool *connection.Pool, schema string) ([]Procedure, error) {
	// Check if prokind column exists (PG11+)
	query := `
		SELECT p.proname, pg_get_function_identity_arguments(p.oid) as args
		FROM pg_proc p
		JOIN pg_namespace n ON p.pronamespace = n.oid
		WHERE n.nspname = $1
		  AND p.prokind = 'p'
		ORDER BY p.proname;
	`

	rows, err := pool.Query(ctx, query, schema)
	if err != nil {
		// If prokind doesn't exist (PG10 or earlier), return empty list
		return []Procedure{}, nil
	}

	procedures := make([]Procedure, 0, len(rows))
	for _, row := range rows {
		procedures = append(procedures, Procedure{
			Schema:    schema,
			Name:      toString(row["proname"]),
			Arguments: toString(row["args"]),
		})
	}

	return procedures, nil
}

// ListTriggerFunctions returns all trigger functions in a schema
func ListTriggerFunctions(ctx context.Context, pool *connection.Pool, schema string) ([]TriggerFunction, error) {
	query := `
		SELECT p.proname
		FROM pg_proc p
		JOIN pg_namespace n ON p.pronamespace = n.oid
		WHERE n.nspname = $1
		  AND p.prorettype = 'trigger'::regtype
		ORDER BY p.proname;
	`

	rows, err := pool.Query(ctx, query, schema)
	if err != nil {
		return nil, err
	}

	functions := make([]TriggerFunction, 0, len(rows))
	for _, row := range rows {
		functions = append(functions, TriggerFunction{
			Schema: schema,
			Name:   toString(row["proname"]),
		})
	}

	return functions, nil
}

// ListSequences returns all sequences in a schema
func ListSequences(ctx context.Context, pool *connection.Pool, schema string) ([]Sequence, error) {
	query := `
		SELECT sequencename, start_value, min_value, max_value, increment_by, cycle
		FROM pg_sequences
		WHERE schemaname = $1
		ORDER BY sequencename;
	`

	rows, err := pool.Query(ctx, query, schema)
	if err != nil {
		return nil, err
	}

	sequences := make([]Sequence, 0, len(rows))
	for _, row := range rows {
		sequences = append(sequences, Sequence{
			Schema:     schema,
			Name:       toString(row["sequencename"]),
			StartValue: toInt64(row["start_value"]),
			MinValue:   toInt64(row["min_value"]),
			MaxValue:   toInt64(row["max_value"]),
			Increment:  toInt64(row["increment_by"]),
			Cycle:      toBool(row["cycle"]),
		})
	}

	return sequences, nil
}

// ListTableIndexes returns all indexes for a specific table
func ListTableIndexes(ctx context.Context, pool *connection.Pool, schema, table string) ([]Index, error) {
	query := `
		SELECT indexname, indexdef
		FROM pg_indexes
		WHERE schemaname = $1 AND tablename = $2
		ORDER BY indexname;
	`

	rows, err := pool.Query(ctx, query, schema, table)
	if err != nil {
		return nil, err
	}

	indexes := make([]Index, 0, len(rows))
	for _, row := range rows {
		indexes = append(indexes, Index{
			Schema:     schema,
			Table:      table,
			Name:       toString(row["indexname"]),
			Definition: toString(row["indexdef"]),
		})
	}

	return indexes, nil
}

// ListTableTriggers returns all triggers for a specific table
func ListTableTriggers(ctx context.Context, pool *connection.Pool, schema, table string) ([]Trigger, error) {
	query := `
		SELECT t.tgname, pg_get_triggerdef(t.oid) as definition
		FROM pg_trigger t
		JOIN pg_class c ON t.tgrelid = c.oid
		JOIN pg_namespace n ON c.relnamespace = n.oid
		WHERE n.nspname = $1 AND c.relname = $2
		  AND NOT t.tgisinternal
		ORDER BY t.tgname;
	`

	rows, err := pool.Query(ctx, query, schema, table)
	if err != nil {
		return nil, err
	}

	triggers := make([]Trigger, 0, len(rows))
	for _, row := range rows {
		triggers = append(triggers, Trigger{
			Schema:     schema,
			Table:      table,
			Name:       toString(row["tgname"]),
			Definition: toString(row["definition"]),
		})
	}

	return triggers, nil
}

// ListExtensions returns all extensions in the database
func ListExtensions(ctx context.Context, pool *connection.Pool) ([]Extension, error) {
	query := `
		SELECT e.extname, e.extversion, n.nspname as schema
		FROM pg_extension e
		JOIN pg_namespace n ON e.extnamespace = n.oid
		ORDER BY e.extname;
	`

	rows, err := pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}

	extensions := make([]Extension, 0, len(rows))
	for _, row := range rows {
		extensions = append(extensions, Extension{
			Name:    toString(row["extname"]),
			Version: toString(row["extversion"]),
			Schema:  toString(row["schema"]),
		})
	}

	return extensions, nil
}

// ListCompositeTypes returns all composite types in a schema
func ListCompositeTypes(ctx context.Context, pool *connection.Pool, schema string) ([]CompositeType, error) {
	query := `
		SELECT t.typname
		FROM pg_type t
		JOIN pg_namespace n ON t.typnamespace = n.oid
		LEFT JOIN pg_class c ON t.typrelid = c.oid
		WHERE n.nspname = $1
		  AND t.typtype = 'c'
		  AND (c.relkind IS NULL OR c.relkind = 'c')
		ORDER BY t.typname;
	`

	rows, err := pool.Query(ctx, query, schema)
	if err != nil {
		return nil, err
	}

	types := make([]CompositeType, 0, len(rows))
	for _, row := range rows {
		types = append(types, CompositeType{
			Schema: schema,
			Name:   toString(row["typname"]),
		})
	}

	return types, nil
}

// ListEnumTypes returns all enum types in a schema
func ListEnumTypes(ctx context.Context, pool *connection.Pool, schema string) ([]EnumType, error) {
	query := `
		SELECT t.typname,
		       array_agg(e.enumlabel ORDER BY e.enumsortorder) as labels
		FROM pg_type t
		JOIN pg_namespace n ON t.typnamespace = n.oid
		JOIN pg_enum e ON t.oid = e.enumtypid
		WHERE n.nspname = $1
		  AND t.typtype = 'e'
		GROUP BY t.typname
		ORDER BY t.typname;
	`

	rows, err := pool.Query(ctx, query, schema)
	if err != nil {
		return nil, err
	}

	types := make([]EnumType, 0, len(rows))
	for _, row := range rows {
		types = append(types, EnumType{
			Schema: schema,
			Name:   toString(row["typname"]),
			Labels: toStringSlice(row["labels"]),
		})
	}

	return types, nil
}

// ListDomainTypes returns all domain types in a schema
func ListDomainTypes(ctx context.Context, pool *connection.Pool, schema string) ([]DomainType, error) {
	query := `
		SELECT t.typname, pg_catalog.format_type(t.typbasetype, t.typtypmod) as basetype
		FROM pg_type t
		JOIN pg_namespace n ON t.typnamespace = n.oid
		WHERE n.nspname = $1
		  AND t.typtype = 'd'
		ORDER BY t.typname;
	`

	rows, err := pool.Query(ctx, query, schema)
	if err != nil {
		return nil, err
	}

	types := make([]DomainType, 0, len(rows))
	for _, row := range rows {
		types = append(types, DomainType{
			Schema:   schema,
			Name:     toString(row["typname"]),
			BaseType: toString(row["basetype"]),
		})
	}

	return types, nil
}

// ListRangeTypes returns all range types in a schema
func ListRangeTypes(ctx context.Context, pool *connection.Pool, schema string) ([]RangeType, error) {
	query := `
		SELECT t.typname, st.typname as subtype
		FROM pg_type t
		JOIN pg_namespace n ON t.typnamespace = n.oid
		JOIN pg_range r ON t.oid = r.rngtypid
		JOIN pg_type st ON r.rngsubtype = st.oid
		WHERE n.nspname = $1
		  AND t.typtype = 'r'
		ORDER BY t.typname;
	`

	rows, err := pool.Query(ctx, query, schema)
	if err != nil {
		return nil, err
	}

	types := make([]RangeType, 0, len(rows))
	for _, row := range rows {
		types = append(types, RangeType{
			Schema:  schema,
			Name:    toString(row["typname"]),
			Subtype: toString(row["subtype"]),
		})
	}

	return types, nil
}

// Helper functions for type conversion
func toInt64(v interface{}) int64 {
	if v == nil {
		return 0
	}
	switch val := v.(type) {
	case int64:
		return val
	case int32:
		return int64(val)
	case int:
		return int64(val)
	case float64:
		return int64(val)
	default:
		return 0
	}
}

func toStringSlice(v interface{}) []string {
	if v == nil {
		return []string{}
	}
	switch val := v.(type) {
	case []string:
		return val
	case []interface{}:
		result := make([]string, len(val))
		for i, item := range val {
			result[i] = fmt.Sprintf("%v", item)
		}
		return result
	default:
		return []string{}
	}
}
