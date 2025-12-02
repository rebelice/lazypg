# PostgreSQL Database Objects Tree Design

## Overview

Extend lazypg's left sidebar to display all major PostgreSQL database objects beyond just Tables. This design follows the pgAdmin/DBeaver patterns for organizing database objects in a hierarchical tree structure.

## Current State

Currently supported:
- Databases
- Schemas
- Tables (with Columns)
- Views (model exists but not fully implemented)

## Target State

### Tree Structure

```
Database: mydb
├── Extensions
│   └── uuid-ossp, pg_trgm, postgis...
└── Schemas
    └── public
        ├── Tables
        │   └── users
        │       ├── Columns
        │       │   └── id, name, email...
        │       ├── Indexes
        │       │   └── users_pkey, idx_users_email...
        │       └── Triggers
        │           └── update_timestamp...
        ├── Views
        │   └── active_users, order_summary...
        ├── Materialized Views
        │   └── monthly_stats, user_metrics...
        ├── Functions
        │   └── get_user(id), calculate_total()...
        ├── Procedures
        │   └── process_order(id), cleanup_logs()...
        ├── Trigger Functions
        │   └── update_modified_column()...
        ├── Sequences
        │   └── users_id_seq, orders_id_seq...
        └── Types
            ├── Composite Types
            │   └── address_type, money_type...
            ├── Enum Types
            │   └── order_status, user_role...
            ├── Domain Types
            │   └── email_domain, phone_domain...
            └── Range Types
                └── date_range, int_range...
```

### Design Decisions

1. **Organization Style**: DBeaver-style grouping by object type
2. **Indexes/Triggers Location**: Under their parent Table (not at schema level)
3. **Functions Separation**: Three separate folders (pgAdmin style)
   - Functions (regular functions)
   - Procedures (PostgreSQL 11+)
   - Trigger Functions (functions returning trigger type)
4. **Extensions Location**: Database level (not schema level)
5. **Types Grouping**: Subdivided by type category
6. **Empty Folders**: Hidden (not displayed if no objects exist)

## Right Panel Display

| Object Type | Display Content |
|-------------|-----------------|
| Tables | Data rows (current behavior) |
| Views | Data rows (SELECT from view) |
| Materialized Views | Data rows (SELECT from view) |
| Functions | Source code with syntax highlighting |
| Procedures | Source code with syntax highlighting |
| Trigger Functions | Source code with syntax highlighting |
| Sequences | Properties table (current value, increment, min, max, cycle) |
| Indexes | DDL definition |
| Triggers | DDL definition |
| Types | Type definition |
| Extensions | Extension info (version, schema, description) |

## New TreeNodeTypes Required

```go
const (
    // Existing
    TreeNodeTypeRoot       TreeNodeType = "root"
    TreeNodeTypeDatabase   TreeNodeType = "database"
    TreeNodeTypeSchema     TreeNodeType = "schema"
    TreeNodeTypeTableGroup TreeNodeType = "table_group"
    TreeNodeTypeTable      TreeNodeType = "table"
    TreeNodeTypeColumn     TreeNodeType = "column"
    TreeNodeTypeViewGroup  TreeNodeType = "view_group"
    TreeNodeTypeView       TreeNodeType = "view"

    // New - Groups
    TreeNodeTypeMaterializedViewGroup TreeNodeType = "materialized_view_group"
    TreeNodeTypeFunctionGroup         TreeNodeType = "function_group"
    TreeNodeTypeProcedureGroup        TreeNodeType = "procedure_group"
    TreeNodeTypeTriggerFunctionGroup  TreeNodeType = "trigger_function_group"
    TreeNodeTypeSequenceGroup         TreeNodeType = "sequence_group"
    TreeNodeTypeTypeGroup             TreeNodeType = "type_group"
    TreeNodeTypeExtensionGroup        TreeNodeType = "extension_group"
    TreeNodeTypeIndexGroup            TreeNodeType = "index_group"
    TreeNodeTypeTriggerGroup          TreeNodeType = "trigger_group"

    // New - Type subcategories
    TreeNodeTypeCompositeTypeGroup TreeNodeType = "composite_type_group"
    TreeNodeTypeEnumTypeGroup      TreeNodeType = "enum_type_group"
    TreeNodeTypeDomainTypeGroup    TreeNodeType = "domain_type_group"
    TreeNodeTypeRangeTypeGroup     TreeNodeType = "range_type_group"

    // New - Leaf nodes
    TreeNodeTypeMaterializedView TreeNodeType = "materialized_view"
    TreeNodeTypeFunction         TreeNodeType = "function"
    TreeNodeTypeProcedure        TreeNodeType = "procedure"
    TreeNodeTypeTriggerFunction  TreeNodeType = "trigger_function"
    TreeNodeTypeSequence         TreeNodeType = "sequence"
    TreeNodeTypeIndex            TreeNodeType = "index"
    TreeNodeTypeTrigger          TreeNodeType = "trigger"
    TreeNodeTypeExtension        TreeNodeType = "extension"
    TreeNodeTypeCompositeType    TreeNodeType = "composite_type"
    TreeNodeTypeEnumType         TreeNodeType = "enum_type"
    TreeNodeTypeDomainType       TreeNodeType = "domain_type"
    TreeNodeTypeRangeType        TreeNodeType = "range_type"
)
```

## Database Queries Required

### Extensions
```sql
SELECT extname, extversion, n.nspname as schema
FROM pg_extension e
JOIN pg_namespace n ON e.extnamespace = n.oid
ORDER BY extname;
```

### Materialized Views
```sql
SELECT schemaname, matviewname, matviewowner
FROM pg_matviews
WHERE schemaname = $1
ORDER BY matviewname;
```

### Functions (regular, excluding triggers and procedures)
```sql
SELECT p.proname, pg_get_function_identity_arguments(p.oid) as args
FROM pg_proc p
JOIN pg_namespace n ON p.pronamespace = n.oid
WHERE n.nspname = $1
  AND p.prokind = 'f'
  AND p.prorettype != 'trigger'::regtype
ORDER BY p.proname;
```

### Procedures (PostgreSQL 11+)
```sql
SELECT p.proname, pg_get_function_identity_arguments(p.oid) as args
FROM pg_proc p
JOIN pg_namespace n ON p.pronamespace = n.oid
WHERE n.nspname = $1
  AND p.prokind = 'p'
ORDER BY p.proname;
```

### Trigger Functions
```sql
SELECT p.proname
FROM pg_proc p
JOIN pg_namespace n ON p.pronamespace = n.oid
WHERE n.nspname = $1
  AND p.prorettype = 'trigger'::regtype
ORDER BY p.proname;
```

### Sequences
```sql
SELECT sequencename, start_value, min_value, max_value, increment_by, cycle
FROM pg_sequences
WHERE schemaname = $1
ORDER BY sequencename;
```

### Indexes (for a specific table)
```sql
SELECT indexname, indexdef
FROM pg_indexes
WHERE schemaname = $1 AND tablename = $2
ORDER BY indexname;
```

### Triggers (for a specific table)
```sql
SELECT tgname, pg_get_triggerdef(t.oid) as definition
FROM pg_trigger t
JOIN pg_class c ON t.tgrelid = c.oid
JOIN pg_namespace n ON c.relnamespace = n.oid
WHERE n.nspname = $1 AND c.relname = $2
  AND NOT t.tgisinternal
ORDER BY tgname;
```

### Types - Composite
```sql
SELECT t.typname
FROM pg_type t
JOIN pg_namespace n ON t.typnamespace = n.oid
WHERE n.nspname = $1
  AND t.typtype = 'c'
  AND t.typrelid != 0
  AND NOT EXISTS (SELECT 1 FROM pg_class c WHERE c.oid = t.typrelid AND c.relkind IN ('r', 'v', 'm'))
ORDER BY t.typname;
```

### Types - Enum
```sql
SELECT t.typname
FROM pg_type t
JOIN pg_namespace n ON t.typnamespace = n.oid
WHERE n.nspname = $1
  AND t.typtype = 'e'
ORDER BY t.typname;
```

### Types - Domain
```sql
SELECT t.typname
FROM pg_type t
JOIN pg_namespace n ON t.typnamespace = n.oid
WHERE n.nspname = $1
  AND t.typtype = 'd'
ORDER BY t.typname;
```

### Types - Range
```sql
SELECT t.typname
FROM pg_type t
JOIN pg_namespace n ON t.typnamespace = n.oid
WHERE n.nspname = $1
  AND t.typtype = 'r'
ORDER BY t.typname;
```

## Icon Mapping (Terminal Unicode)

| Object Type | Icon | Description |
|-------------|------|-------------|
| Database | 󰆼 | Database icon |
| Schema | 󰙅 | Folder structure |
| Table | 󰓫 | Table grid |
| View | 󰈈 | Eye/view |
| Materialized View | 󰈈 | Eye with cache indicator |
| Function | 󰊕 | Function symbol |
| Procedure | 󰜎 | Procedure/process |
| Trigger Function | 󱓞 | Trigger + function |
| Sequence | 󰔡 | Sequential numbers |
| Index | 󰛤 | Index/sort |
| Trigger | 󱐋 | Lightning/trigger |
| Extension | 󰏖 | Plugin/extension |
| Type | 󰜁 | Type definition |
| Column | 󰠵 | Column indicator |

## Implementation Phases

### Phase 1: Core Infrastructure
- Add new TreeNodeTypes to models
- Update tree building logic
- Implement lazy loading for new object types

### Phase 2: Metadata Queries
- Add query functions for each object type
- Handle PostgreSQL version differences (e.g., procedures in PG11+)

### Phase 3: Tree View Updates
- Update icons for new node types
- Implement expand/collapse for new groups
- Handle selection and navigation

### Phase 4: Right Panel Display
- Function/Procedure source code viewer with syntax highlighting
- Sequence properties display
- Index/Trigger DDL display
- Type definition display

### Phase 5: Polish
- Performance optimization (lazy loading)
- Error handling for missing objects
- Empty folder hiding logic

## References

- [pgAdmin Tree Control](https://www.pgadmin.org/docs/pgadmin4/development/tree_control.html)
- [DBeaver Database Navigator](https://dbeaver.com/docs/dbeaver/Database-Navigator/)
- [DataGrip Database Explorer](https://www.jetbrains.com/help/datagrip/database-explorer.html)
- [PostgreSQL System Catalogs](https://www.postgresql.org/docs/current/catalogs.html)
