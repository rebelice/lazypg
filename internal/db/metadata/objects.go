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

// FunctionSource represents the source code of a function/procedure
type FunctionSource struct {
	Name       string
	Schema     string
	Arguments  string
	ReturnType string
	Language   string
	Source     string
	Definition string // Full CREATE FUNCTION statement
}

// SequenceDetails represents detailed sequence information including current value
type SequenceDetails struct {
	Schema       string
	Name         string
	CurrentValue int64
	StartValue   int64
	MinValue     int64
	MaxValue     int64
	Increment    int64
	Cycle        bool
	Owner        string
}

// ExtensionDetails represents detailed extension information
type ExtensionDetails struct {
	Name        string
	Version     string
	Schema      string
	Description string
}

// CompositeTypeDetails represents detailed composite type information
type CompositeTypeDetails struct {
	Schema     string
	Name       string
	Attributes []TypeAttribute
}

// TypeAttribute represents an attribute of a composite type
type TypeAttribute struct {
	Name     string
	Type     string
	Position int
}

// DomainTypeDetails represents detailed domain type information
type DomainTypeDetails struct {
	Schema      string
	Name        string
	BaseType    string
	Default     string
	NotNull     bool
	Constraints []string
}

// GetFunctionSource returns the source code of a function, procedure, or trigger function
func GetFunctionSource(ctx context.Context, pool *connection.Pool, schema, name, args string) (*FunctionSource, error) {
	query := `
		SELECT
			p.proname,
			n.nspname,
			pg_get_function_identity_arguments(p.oid) as args,
			pg_get_function_result(p.oid) as return_type,
			l.lanname as language,
			p.prosrc as source,
			pg_get_functiondef(p.oid) as definition
		FROM pg_proc p
		JOIN pg_namespace n ON p.pronamespace = n.oid
		JOIN pg_language l ON p.prolang = l.oid
		WHERE n.nspname = $1 AND p.proname = $2
		  AND pg_get_function_identity_arguments(p.oid) = $3;
	`

	rows, err := pool.Query(ctx, query, schema, name, args)
	if err != nil {
		return nil, err
	}

	if len(rows) == 0 {
		return nil, fmt.Errorf("function %s.%s(%s) not found", schema, name, args)
	}

	row := rows[0]
	return &FunctionSource{
		Name:       toString(row["proname"]),
		Schema:     toString(row["nspname"]),
		Arguments:  toString(row["args"]),
		ReturnType: toString(row["return_type"]),
		Language:   toString(row["language"]),
		Source:     toString(row["source"]),
		Definition: toString(row["definition"]),
	}, nil
}

// GetTriggerFunctionSource returns the source code of a trigger function (no args version)
func GetTriggerFunctionSource(ctx context.Context, pool *connection.Pool, schema, name string) (*FunctionSource, error) {
	query := `
		SELECT
			p.proname,
			n.nspname,
			'' as args,
			'trigger' as return_type,
			l.lanname as language,
			p.prosrc as source,
			pg_get_functiondef(p.oid) as definition
		FROM pg_proc p
		JOIN pg_namespace n ON p.pronamespace = n.oid
		JOIN pg_language l ON p.prolang = l.oid
		WHERE n.nspname = $1 AND p.proname = $2
		  AND p.prorettype = 'trigger'::regtype;
	`

	rows, err := pool.Query(ctx, query, schema, name)
	if err != nil {
		return nil, err
	}

	if len(rows) == 0 {
		return nil, fmt.Errorf("trigger function %s.%s not found", schema, name)
	}

	row := rows[0]
	return &FunctionSource{
		Name:       toString(row["proname"]),
		Schema:     toString(row["nspname"]),
		Arguments:  toString(row["args"]),
		ReturnType: toString(row["return_type"]),
		Language:   toString(row["language"]),
		Source:     toString(row["source"]),
		Definition: toString(row["definition"]),
	}, nil
}

// GetSequenceDetails returns detailed information about a sequence including current value
func GetSequenceDetails(ctx context.Context, pool *connection.Pool, schema, name string) (*SequenceDetails, error) {
	// First get the sequence properties
	query := `
		SELECT
			schemaname,
			sequencename,
			start_value,
			min_value,
			max_value,
			increment_by,
			cycle,
			sequenceowner
		FROM pg_sequences
		WHERE schemaname = $1 AND sequencename = $2;
	`

	rows, err := pool.Query(ctx, query, schema, name)
	if err != nil {
		return nil, err
	}

	if len(rows) == 0 {
		return nil, fmt.Errorf("sequence %s.%s not found", schema, name)
	}

	row := rows[0]
	details := &SequenceDetails{
		Schema:     toString(row["schemaname"]),
		Name:       toString(row["sequencename"]),
		StartValue: toInt64(row["start_value"]),
		MinValue:   toInt64(row["min_value"]),
		MaxValue:   toInt64(row["max_value"]),
		Increment:  toInt64(row["increment_by"]),
		Cycle:      toBool(row["cycle"]),
		Owner:      toString(row["sequenceowner"]),
	}

	// Get current value using last_value from the sequence itself
	// This requires querying the sequence directly
	lastValueQuery := fmt.Sprintf(`SELECT last_value FROM "%s"."%s"`, schema, name)
	lastValueRows, err := pool.Query(ctx, lastValueQuery)
	if err == nil && len(lastValueRows) > 0 {
		details.CurrentValue = toInt64(lastValueRows[0]["last_value"])
	}

	return details, nil
}

// GetExtensionDetails returns detailed information about an extension
func GetExtensionDetails(ctx context.Context, pool *connection.Pool, name string) (*ExtensionDetails, error) {
	query := `
		SELECT
			e.extname,
			e.extversion,
			n.nspname as schema,
			c.description
		FROM pg_extension e
		JOIN pg_namespace n ON e.extnamespace = n.oid
		LEFT JOIN pg_description c ON c.objoid = e.oid AND c.classoid = 'pg_extension'::regclass
		WHERE e.extname = $1;
	`

	rows, err := pool.Query(ctx, query, name)
	if err != nil {
		return nil, err
	}

	if len(rows) == 0 {
		return nil, fmt.Errorf("extension %s not found", name)
	}

	row := rows[0]
	return &ExtensionDetails{
		Name:        toString(row["extname"]),
		Version:     toString(row["extversion"]),
		Schema:      toString(row["schema"]),
		Description: toString(row["description"]),
	}, nil
}

// GetCompositeTypeDetails returns detailed information about a composite type
func GetCompositeTypeDetails(ctx context.Context, pool *connection.Pool, schema, name string) (*CompositeTypeDetails, error) {
	query := `
		SELECT
			a.attname,
			pg_catalog.format_type(a.atttypid, a.atttypmod) as type,
			a.attnum as position
		FROM pg_type t
		JOIN pg_namespace n ON t.typnamespace = n.oid
		JOIN pg_attribute a ON a.attrelid = t.typrelid
		WHERE n.nspname = $1
		  AND t.typname = $2
		  AND a.attnum > 0
		  AND NOT a.attisdropped
		ORDER BY a.attnum;
	`

	rows, err := pool.Query(ctx, query, schema, name)
	if err != nil {
		return nil, err
	}

	details := &CompositeTypeDetails{
		Schema:     schema,
		Name:       name,
		Attributes: make([]TypeAttribute, 0, len(rows)),
	}

	for _, row := range rows {
		details.Attributes = append(details.Attributes, TypeAttribute{
			Name:     toString(row["attname"]),
			Type:     toString(row["type"]),
			Position: int(toInt64(row["position"])),
		})
	}

	return details, nil
}

// GetDomainTypeDetails returns detailed information about a domain type
func GetDomainTypeDetails(ctx context.Context, pool *connection.Pool, schema, name string) (*DomainTypeDetails, error) {
	query := `
		SELECT
			t.typname,
			pg_catalog.format_type(t.typbasetype, t.typtypmod) as basetype,
			t.typdefault as default_value,
			t.typnotnull as not_null
		FROM pg_type t
		JOIN pg_namespace n ON t.typnamespace = n.oid
		WHERE n.nspname = $1
		  AND t.typname = $2
		  AND t.typtype = 'd';
	`

	rows, err := pool.Query(ctx, query, schema, name)
	if err != nil {
		return nil, err
	}

	if len(rows) == 0 {
		return nil, fmt.Errorf("domain type %s.%s not found", schema, name)
	}

	row := rows[0]
	details := &DomainTypeDetails{
		Schema:   schema,
		Name:     toString(row["typname"]),
		BaseType: toString(row["basetype"]),
		Default:  toString(row["default_value"]),
		NotNull:  toBool(row["not_null"]),
	}

	// Get domain constraints
	constraintQuery := `
		SELECT pg_get_constraintdef(c.oid) as constraint_def
		FROM pg_constraint c
		JOIN pg_type t ON c.contypid = t.oid
		JOIN pg_namespace n ON t.typnamespace = n.oid
		WHERE n.nspname = $1 AND t.typname = $2;
	`

	constraintRows, err := pool.Query(ctx, constraintQuery, schema, name)
	if err == nil {
		for _, cr := range constraintRows {
			details.Constraints = append(details.Constraints, toString(cr["constraint_def"]))
		}
	}

	return details, nil
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
