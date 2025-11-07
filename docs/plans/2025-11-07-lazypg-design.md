# lazypg - PostgreSQL TUI Client Design Document

**Date:** 2025-11-07
**Status:** Design Approved
**Author:** Design Brainstorming Session

## Executive Summary

lazypg is a modern Terminal User Interface (TUI) client for PostgreSQL, inspired by lazygit. It addresses the pain points of existing PostgreSQL clients by providing a keyboard-driven, intuitive interface that combines the speed of CLI tools with the usability of GUI applications.

**Core Value Propositions:**
- Keyboard-first workflow with full mouse support
- Command palette as unified entry point (like VS Code)
- Intelligent JSONB support with automatic path extraction
- Interactive filter builder that generates SQL
- Virtual scrolling for large datasets
- Declarative schema editing (SDL) with DDL generation

## 1. Technical Architecture

### 1.1 Technology Stack

- **Language:** Go 1.21+
- **TUI Framework:** Bubble Tea (bubbletea)
- **Styling:** Lipgloss
- **UI Components:** Bubbles
- **PostgreSQL Driver:** pgx v5
- **Configuration:** YAML
- **History Storage:** SQLite

### 1.2 Project Structure

```
lazypg/
├── cmd/lazypg/          # Main program entry
├── internal/
│   ├── app/             # Bubble Tea application logic
│   ├── ui/              # UI components (panels, views)
│   │   ├── panels/      # Left nav, right content, command palette
│   │   ├── views/       # Data view, structure view, query editor
│   │   └── components/  # Reusable UI components
│   ├── db/              # Database connection and queries
│   │   ├── connection/  # Connection pool management
│   │   ├── metadata/    # Schema metadata queries
│   │   └── discovery/   # Auto-discovery logic
│   ├── config/          # Configuration management
│   ├── filter/          # Interactive filter builder
│   ├── jsonb/           # JSONB processing and path extraction
│   ├── history/         # Query history management
│   └── models/          # Data models
├── config/              # Default configuration files
├── docs/                # Documentation
│   └── plans/           # Design and implementation plans
└── README.md
```

### 1.3 Architecture Patterns

**Bubble Tea Elm Architecture:**
- **Model:** Application state (connections, navigation, data cache)
- **Update:** Message handling (keyboard, mouse, database responses)
- **View:** UI rendering

**Key Design Principles:**
- Separation of concerns (UI / Business Logic / Data Access)
- Asynchronous operations (non-blocking UI)
- Caching strategy (metadata, query results)
- Error resilience (graceful degradation)

## 2. User Interface Design

### 2.1 Layout Structure

```
┌─────────────────────────────────────────────────────────┐
│ lazypg              postgres@localhost:5432/mydb    ⌘K │
├──────────────┬──────────────────────────────────────────┤
│              │  mydb > public > tables > users           │
│              │  ┌─ Data ─┬─ Structure ─┬─ Indexes ─┐    │
│ Databases    │  │                                   │    │
│ └─ mydb      │  │  id  │ name      │ created_at    │    │
│    └─ public │  │ ────┼───────────┼─────────────── │    │
│       ├─Tables│  │  1  │ Alice     │ 2024-01-01    │    │
│       │ ├─users│ │  2  │ Bob       │ 2024-01-02    │    │
│       │ ├─posts│ │                                   │    │
│       ├─Views │  │                                   │    │
│       ├─Funcs │  │                                   │    │
│       └─Seqs  │  └───────────────────────────────────┘    │
│              │  Rows 2/1,000 | Query: 45ms               │
├──────────────┴──────────────────────────────────────────┤
│ [e] Edit [f] Filter [r] Refresh [c] Copy  | ⌘K Command │
└─────────────────────────────────────────────────────────┘
```

**Three-panel layout:**
1. **Left Panel (25% width, adjustable):** Navigation tree
2. **Right Panel (75% width):** Content area with tabs
3. **Bottom Bar:** Contextual shortcuts and status

**Top Bar:** Connection info and command palette trigger

### 2.2 Command Palette (Core UX Innovation)

**Trigger:** Ctrl+K / Cmd+K from anywhere

**Appearance:**
```
┌─────────────────────────────────────────────────────────┐
│ ┌─────────────────────────────────────────────────────┐ │
│ │ > search tables_                                    │ │
│ │ ───────────────────────────────────────────────────  │ │
│ │  🔍 Tables: users (public)                          │ │
│ │  🔍 Tables: user_sessions (public)                  │ │
│ │  💾 Recent: SELECT * FROM users WHERE...           │ │
│ │  ⚡ Command: Connect to Database                    │ │
│ │  ⚡ Command: Execute Query                          │ │
│ └─────────────────────────────────────────────────────┘ │
```

**Unified search across:**
- Database objects (tables, views, functions, etc.)
- Commands (connect, settings, help, etc.)
- Query history
- Saved queries/favorites
- Documentation

**Prefix-based modes:**
- Default: Smart search (all sources)
- `>` : Command mode
- `?` : Help/documentation search
- `@` : Jump to object
- `#` : Tags/favorites
- Direct SQL: Quick query mode

**Smart ranking:**
- Recent usage
- Fuzzy match score
- Object type priority

### 2.3 Navigation Tree

**Hierarchy:**
```
Databases
└─ Database Name
   └─ Schema Name
      ├─ Tables
      │  └─ table_name
      ├─ Views
      ├─ Functions
      ├─ Sequences
      ├─ Types
      └─ Extensions
```

**Features:**
- Keyboard navigation (↑↓ / j/k)
- Expand/collapse (Enter / Space)
- Mouse click support
- Search within tree (/)
- Refresh (r)
- Visual indicators for object types

### 2.4 Content Area Tabs

Dynamic tabs based on selected object:

**For Tables:**
- Data: Table data with virtual scrolling
- Structure: Columns with types, nullability, defaults
- Indexes: Index definitions and sizes
- Constraints: PK, FK, CHECK, UNIQUE constraints

**For Views:**
- Data: View results
- Definition: CREATE VIEW SQL

**For Functions:**
- Definition: Function code
- Parameters: Input/output parameters

**Tab Navigation:**
- Number keys (1-9)
- Mouse click
- Ctrl+Tab / Ctrl+Shift+Tab

## 3. Core Features

### 3.1 Connection Management

**Multiple Connection Methods:**

1. **Command line:**
   ```bash
   lazypg postgres://user:pass@localhost:5432/mydb
   lazypg -h localhost -p 5432 -U user -d mydb
   ```

2. **Configuration file** (`~/.config/lazypg/connections.yaml`):
   ```yaml
   connections:
     - name: "Local Dev"
       host: localhost
       port: 5432
       database: mydb
       user: postgres
   ```

3. **Command palette:** ⌘K > "connect"

4. **Auto-discovery:**
   - Port scanning (5432-5435)
   - Environment variables (PGHOST, PGPORT, etc.)
   - `.pgpass` file parsing
   - `.pg_service.conf` parsing
   - Docker container detection
   - Unix socket detection

**Connection Pool:**
- pgx connection pooling
- Configurable pool size
- Automatic reconnection
- Connection health checks

### 3.2 Data Browsing with Virtual Scrolling

**Implementation Strategy:**

1. **Initial Load:**
   - Load first 100 rows (configurable)
   - Async query total row count
   - Display loading indicator

2. **Virtual Scrolling:**
   - Monitor scroll position
   - Pre-fetch upcoming data (buffer zone)
   - Cache loaded chunks (100 rows per chunk)
   - Max 1000 rows in memory

3. **Query Strategy:**
   - Use OFFSET + LIMIT for pagination
   - Consider cursors for very large tables
   - Support ORDER BY for consistent scrolling

4. **Large Table Handling:**
   - Warn when table > 1M rows
   - Suggest adding filters
   - Show estimated query time

**Table Features:**
- Column sorting (click header or shortcut)
- Auto-sizing columns
- Manual column resize
- Cell content preview (truncate long text)
- Row selection (single/multi)
- Copy operations (cell/row/selection)

### 3.3 Interactive Filter Builder

**Workflow:**

1. Press `f` to open filter builder
2. Add conditions by selecting:
   - Column (dropdown with search)
   - Operator (context-aware based on type)
   - Value (with validation)
3. Preview generated SQL
4. Apply to reload data

**Filter UI:**
```
┌─────────────────────────────────────────────────────────┐
│ 📋 Filters (2 active)                    [Apply] [Clear] │
│ ┌─────────────────────────────────────────────────────┐ │
│ │ 1. name     [contains ▾]  [alice___]          [x]   │ │
│ │ 2. created_at [>= ▾]      [2024-01-01]        [x]   │ │
│ │                                          [+ Add]     │ │
│ └─────────────────────────────────────────────────────┘ │
│ Generated SQL: WHERE name ILIKE '%alice%' AND           │
│                created_at >= '2024-01-01'               │
└─────────────────────────────────────────────────────────┘
```

**Operators by Type:**
- **Text:** =, !=, LIKE, ILIKE, ~ (regex), IS NULL
- **Number:** =, !=, >, <, >=, <=, BETWEEN, IN
- **Date:** =, !=, >, <, >=, <=, BETWEEN
- **Boolean:** IS TRUE, IS FALSE, IS NULL
- **Array:** @> (contains), <@ (contained), && (overlaps)
- **JSONB:** @>, ?, ?|, ?&, ->>, #>> (with path)

**Quick Filter from Cell:**
- Select cell, press `f`: Add `column = value`
- `Shift+F`: Add `column != value`
- `Alt+F`: Show operator menu
- Multi-select support: Generate IN clause

**Filter Stack:**
- Multiple conditions with AND logic
- Support OR by switching connectors
- Visual indication of active filters
- One-click clear individual or all filters

### 3.4 JSONB/JSON Support

**Display in Table:**
- Compressed single-line view
- Syntax highlighting
- Truncate with `...` for long values
- Expand icon `⏎`

**Expanded View (Enter or double-click):**

**Three modes:**

1. **Formatted Mode:**
   - Pretty-printed JSON
   - Syntax highlighting
   - Collapsible sections
   - Copy/search support

2. **Tree Mode:**
   ```
   📦 metadata
   ├─ 🔢 age: 25
   ├─ 📋 tags (array, 2 items)
   │  ├─ [0]: "dev"
   │  └─ [1]: "golang"
   └─ 📦 settings
      ├─ 🔤 theme: "dark"
      └─ ☑️  notifications: true
   ```
   - Interactive navigation
   - Show JSON Path for selected node
   - Quick filter from node

3. **Query Mode:**
   - Input JSON Path expression
   - Real-time result display
   - Support PostgreSQL JSONB operators

**JSONB Filtering:**

**Path Auto-Extraction:**
- Sample first 1000 rows
- Extract all unique JSON paths
- Infer types from values
- Show coverage percentage
- Cache per table+column

**Filter UI for JSONB:**
```
┌─────────────────────────────────────────────────────────┐
│ Path:  [$.age                              ▾]           │
│        ┌──────────────────────────────────┐             │
│        │ 🔍 Search paths...                │             │
│        ├──────────────────────────────────┤             │
│        │ ✓ $.age              (number)    │             │
│        │   $.tags             (array)     │             │
│        │   $.settings.theme   (string)    │             │
│        │   $.city             (string, 40%)│             │
│        └──────────────────────────────────┘             │
│ Operator: [= ▾]                                         │
│ Value:    [25]                                          │
│ Preview: metadata->>'age'::int = 25                     │
└─────────────────────────────────────────────────────────┘
```

**Benefits:**
- No need to memorize JSON structure
- Visual path selection
- Type-aware operators
- Example values as hints

### 3.5 Query Execution

**Two Query Modes:**

**1. Quick Query (Ctrl+P):**
- Single-line input at bottom
- For simple queries
- SQL syntax highlighting
- Autocomplete (tables, columns, keywords)
- History navigation (↑↓)
- Press Enter to execute
- Ctrl+E to switch to full editor

**2. Query Editor (Ctrl+Shift+P):**

```
┌─────────────────────────────────────────────────────────┐
│ Query Editor                    [x] Close [▶] Run (F5)  │
├─────────────────────────────────────────────────────────┤
│  1  SELECT u.id,                                         │
│  2         u.name,                                       │
│  3         u.metadata->>'age' as age                     │
│  4  FROM users u                                         │
│  5  WHERE u.created_at > NOW() - INTERVAL '7 days'      │
│  6  ORDER BY u.created_at DESC;                         │
├─────────────────────────────────────────────────────────┤
│ 📊 Results (156 rows in 23ms)                           │
│  id  │ name      │ age                                   │
│ ────┼───────────┼──────────────────────────────────     │
│  1   │ Alice     │ 25                                    │
└─────────────────────────────────────────────────────────┘
```

**Editor Features:**
- Line numbers
- Syntax highlighting
- Autocomplete (keywords, tables, columns, functions)
- Multi-statement support (semicolon-separated)
- Execute selection (highlight + F5)
- SQL formatting (Ctrl+/)
- Split view (editor + results)

**Result Display:**
- Same table view as data browsing
- Virtual scrolling for large results
- Export options (CSV, JSON, SQL)
- Copy functionality
- Execution stats (rows, duration)

**Error Handling:**
- Clear error messages
- Line number highlighting (if available)
- Suggestions for common mistakes
- Link to documentation

### 3.6 Query History & Favorites

**History Storage:**

SQLite database (`~/.config/lazypg/history.db`):
```sql
CREATE TABLE query_history (
  id INTEGER PRIMARY KEY,
  connection_name TEXT,
  database_name TEXT,
  query TEXT,
  executed_at TIMESTAMP,
  duration_ms INTEGER,
  rows_affected INTEGER,
  success BOOLEAN,
  error_message TEXT
);
```

**History View (Ctrl+H):**
- Grouped by time (Today, Yesterday, Last 7 days, Older)
- Success/failure indicators
- Search and filter
- View details
- Re-run queries
- Save to favorites

**Favorites:**

YAML storage (`~/.config/lazypg/favorites.yaml`):
```yaml
favorites:
  - name: "Active users last 7 days"
    description: "Get all active users"
    tags: ["users", "analytics"]
    query: |
      SELECT ...
```

**Favorites View (Ctrl+B):**
- Organized by tags
- Search by name/description/tags
- Quick execution
- Edit/delete management
- Export/import favorites

**Command Palette Integration:**
- Favorites appear in search results
- Marked with ⭐ icon
- Quick access by name

### 3.7 Schema Viewing

**Structure Tab for Tables:**

Display:
- Column name, type, nullable, default, comment
- Primary key indicators
- Foreign key references
- Constraints

**Indexes Tab:**
- Index name, type (btree, gin, etc.)
- Indexed columns
- Unique/non-unique
- Index size
- Full definition SQL

**Constraints Tab:**
- Constraint name and type
- Definition
- Referenced tables (for FK)

**Other Object Types:**
- **Views:** Column list + VIEW definition SQL
- **Functions:** Parameters, return type, language, function body
- **Sequences:** Current value, increment, min/max, cycle

**Actions:**
- Copy DDL (c)
- Edit schema (e) - SDL mode
- Drop object (d) - with confirmation
- Refresh (r)

### 3.8 SDL Mode & DDL Generation

*Note: This is a post-MVP feature, included in design for completeness*

**Concept:**
Users edit a simplified schema representation, and lazypg generates the necessary DDL.

**SDL Syntax (simplified):**
```
TABLE users
  id serial
  name varchar(255)
  email text
  age int
  created_at timestamp

INDEX users_email_idx ON users(email) UNIQUE
```

**Edit Flow:**
1. Press `e` on table
2. Left panel: current schema (read-only)
3. Right panel: editable SDL
4. Modify as desired
5. Press "Preview" to see generated DDL
6. Review changes and warnings
7. Press "Apply" to execute

**DDL Generation (Diff Algorithm):**
- Compare current vs modified schema
- Generate ALTER TABLE statements
- Order operations safely
- Detect renames (similarity matching)
- Provide warnings for destructive changes
- Execute in transaction (rollback on error)

**Safety Features:**
- Confirm destructive operations (DROP COLUMN)
- Warn about potential downtime
- Show estimated execution time
- Dry-run option (preview only)

## 4. Configuration System

### 4.1 Configuration Files

**Location:** `~/.config/lazypg/`

**Main Config (`config.yaml`):**
```yaml
general:
  auto_connect_last: true
  confirm_destructive_ops: true
  default_limit: 100

ui:
  theme: "default"
  mouse_enabled: true
  panel_width_ratio: 0.25

editor:
  tab_size: 2
  auto_complete: true

data:
  virtual_scroll_buffer: 100
  max_cell_display_length: 100
  jsonb_auto_format: true

history:
  enabled: true
  max_entries: 1000
  persist: true
```

**Keybindings (`keybindings.yaml`):**
```yaml
global:
  command_palette: ["ctrl+k", "cmd+k"]
  quick_query: ["ctrl+p"]
  quit: ["ctrl+q", "q"]

navigation:
  move_up: ["up", "k"]
  move_down: ["down", "j"]

data_view:
  filter: ["f", "/"]
  copy_cell: ["c"]
```

**Themes (`themes/default.yaml`):**
```yaml
name: "Default"
colors:
  background: "#1e1e1e"
  foreground: "#d4d4d4"
  border_focused: "#007acc"
  keyword: "#569cd6"
  string: "#ce9178"
```

### 4.2 Settings UI

Command palette: `>settings`

- Grouped by category (General, UI, Editor, etc.)
- Live preview for theme changes
- Validation for invalid values
- Reset to defaults
- Export/import configuration

### 4.3 Keybinding Customization

- Visual keybinding editor
- Conflict detection
- Record mode (press keys to assign)
- Multiple bindings per action
- Reset individual or all bindings

## 5. Error Handling & Resilience

### 5.1 Error Categories

**Connection Errors:**
- Clear messaging
- Actionable suggestions
- Retry mechanism
- Auto-reconnect with exponential backoff

**Query Errors:**
- Syntax error highlighting
- Spelling suggestions
- Type mismatch explanations
- Link to relevant docs

**Permission Errors:**
- Show current user and required privilege
- Suggest contacting admin or switching connection

**Data Errors:**
- Constraint violation details
- Suggest corrections

### 5.2 Notification System

**Toast Notifications (right-top corner):**

- **Success:** Auto-dismiss after 3s
- **Info:** Auto-dismiss after 3s
- **Warning:** Auto-dismiss after 5s, closable
- **Error:** Persistent until closed

**Notification Queue:**
- Max 3 visible at once
- Stack vertically
- Merge similar notifications

### 5.3 Connection Resilience

**Reconnection Strategy:**
- Detect connection loss
- Exponential backoff: 1s, 2s, 4s, 8s, 16s
- Max 5 retries
- Show progress UI
- Restore context on success

### 5.4 Logging

**Log Files (`~/.config/lazypg/logs/`):**
- `app.log` - General application logs
- `queries.log` - All executed queries
- `errors.log` - Error-level logs

**Log Viewer:**
- Command palette: `>logs`
- Filter by level
- Search logs
- Copy/export

### 5.5 Crash Recovery

**Session Persistence:**
- Save state on graceful exit
- Detect unexpected shutdown
- Offer to restore last session
- Option to view crash report

## 6. MVP Scope & Implementation Plan

### 6.1 MVP Feature Set

**Included in MVP:**
✅ Connection management (CLI, config, auto-discovery)
✅ Database navigation tree
✅ Table data browsing with virtual scrolling
✅ Schema viewing (Structure, Indexes, Constraints)
✅ Command palette
✅ Quick query and query editor
✅ Interactive filter builder
✅ JSONB support (display, expand, path extraction, filtering)
✅ Query history
✅ Query favorites
✅ Configuration system
✅ Basic theme support
✅ Error handling and notifications
✅ Help system

**Post-MVP:**
❌ SDL editing and DDL generation
❌ Data editing (INSERT/UPDATE/DELETE via UI)
❌ Advanced query editor features (autocomplete, multi-tab)
❌ SQL formatter
❌ Docker auto-discovery
❌ Multiple simultaneous connections
❌ Database creation/management via UI
❌ Performance monitoring
❌ Query plan analysis (EXPLAIN)
❌ Data import

### 6.2 Development Phases

**Phase 1: Foundation (2-3 weeks)**
- Bubble Tea app skeleton
- Multi-panel layout
- Basic navigation
- Configuration loading
- Theme system

**Phase 2: Connection & Discovery (2-3 weeks)**
- pgx integration
- Connection pool
- Auto-discovery
- Connection manager UI
- Metadata queries

**Phase 3: Data Browsing (3-4 weeks)**
- Navigation tree implementation
- Table data view
- Virtual scrolling
- Structure/Indexes/Constraints tabs
- Copy functionality

**Phase 4: Command Palette & Query (2-3 weeks)**
- Command palette UI
- Smart search
- Quick query mode
- Query editor
- Result display
- Syntax highlighting

**Phase 5: Filtering (2 weeks)**
- Interactive filter UI
- Type-aware operators
- SQL generation
- Quick filter from cell
- Filter preview

**Phase 6: JSONB Support (2 weeks)**
- JSONB formatting
- Three-mode viewer (Formatted/Tree/Query)
- Path extraction algorithm
- JSONB filtering integration

**Phase 7: History (1 week)**
- SQLite storage
- History UI
- Search and filtering
- Re-execution

**Phase 8: Favorites & Polish (1-2 weeks)**
- Favorites YAML storage
- Favorites UI
- Export functionality
- Help system
- Error handling refinement
- Documentation

**Total Estimated Time: 15-20 weeks**

### 6.3 Success Criteria

**MVP is successful if:**
1. Users can connect to PostgreSQL easily (local auto-discovery works)
2. Navigation is intuitive (tree + command palette)
3. Browsing large tables is smooth (virtual scrolling performs well)
4. Filtering is faster than writing SQL manually
5. JSONB inspection is significantly better than psql/pgcli
6. Query execution is reliable and fast
7. No data loss or corruption bugs
8. Crash-free for common workflows

**Key Metrics:**
- Time to connect and browse first table < 30 seconds
- Filter creation < 1 minute (vs 2-3 min writing SQL)
- JSONB path selection < 30 seconds (vs 5 min manual inspection)
- Virtual scroll latency < 100ms
- Zero critical bugs in 2 weeks of testing

## 7. Future Enhancements (Post-MVP)

### 7.1 Advanced Features
- **Schema diff:** Compare two databases
- **Migration generator:** Track schema changes over time
- **Query builder:** Visual query construction
- **EXPLAIN visualization:** Graphical query plan
- **Performance monitoring:** Real-time connection/query stats
- **Backup/restore:** Database dump/restore UI
- **Extension management:** Install/configure extensions
- **Role/permission management:** User administration

### 7.2 Integrations
- **Git integration:** Commit query changes
- **Cloud providers:** Connect to AWS RDS, Google Cloud SQL, etc.
- **SSH tunneling:** Connect through bastion hosts
- **Multiple DB support:** Extend to MySQL, SQLite (different tool or modes)

### 7.3 Collaboration Features
- **Shared favorites:** Team query library
- **Query comments:** Annotate saved queries
- **Connection sharing:** Team connection configs

### 7.4 Developer Experience
- **Plugin system:** Allow extensions
- **Scripting:** Automate tasks with scripts
- **API mode:** Use as library in other tools

## 8. Technical Challenges & Mitigations

### 8.1 Virtual Scrolling Performance
**Challenge:** Smooth scrolling with large datasets
**Mitigation:**
- Efficient caching strategy
- Pre-fetch buffer
- Use cursors for very large tables
- Profile and optimize hot paths

### 8.2 JSONB Path Extraction
**Challenge:** Analyzing diverse JSON structures efficiently
**Mitigation:**
- Limit sampling (first 1000 rows)
- Async background processing
- Cache results per table+column
- Efficient JSON parsing (using Go stdlib)

### 8.3 Connection Resilience
**Challenge:** Handling network issues gracefully
**Mitigation:**
- Robust reconnection logic
- Clear user feedback
- Transaction safety (rollback on disconnect)
- Timeout configuration

### 8.4 TUI Complexity
**Challenge:** Rich UI in terminal constraints
**Mitigation:**
- Use proven framework (Bubble Tea)
- Progressive enhancement (mouse optional)
- Thorough testing on different terminals
- Graceful degradation for limited terminals

### 8.5 Cross-platform Compatibility
**Challenge:** Works on macOS, Linux, Windows
**Mitigation:**
- Test on all platforms early
- Use Go's cross-platform APIs
- Conditional code for platform-specific features
- CI/CD with multi-platform builds

## 9. Documentation Plan

### 9.1 User Documentation
- **README:** Quick start, installation, basic usage
- **User Guide:** Comprehensive feature documentation
- **Keyboard Reference:** Printable cheat sheet
- **FAQ:** Common questions and troubleshooting
- **Video Tutorials:** Screen recordings for key features

### 9.2 Developer Documentation
- **Architecture Overview:** System design
- **Contribution Guide:** How to contribute
- **Code Standards:** Style guide, best practices
- **API Documentation:** GoDoc comments
- **Development Setup:** Local development environment

### 9.3 In-App Help
- Help command (? or F1)
- Context-sensitive help
- Command palette help mode (?)
- Tooltip hints

## 10. Conclusion

lazypg aims to fill the gap between simple CLI tools and heavy GUI clients for PostgreSQL. By combining keyboard-driven efficiency with modern UX patterns (command palette, smart search, visual feedback), it provides a powerful yet approachable tool for daily database work.

The MVP focuses on core workflows: connecting, browsing, querying, and filtering data, with special attention to JSONB handling. The modular architecture allows for future expansion while keeping the initial scope achievable.

**Next Steps:**
1. ✅ Design approval
2. Initialize git repository
3. Create detailed implementation plan
4. Set up development environment
5. Begin Phase 1 implementation

---

**Design Status:** ✅ Approved
**Ready for Implementation Planning:** Yes
