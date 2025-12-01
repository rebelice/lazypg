# lazypg

A fast, keyboard-driven Terminal User Interface (TUI) for PostgreSQL, inspired by [lazygit](https://github.com/jesseduffield/lazygit).

> **Status: Beta** - Core features are stable. See [Roadmap](docs/ROADMAP.md) for planned features.

## Features

- **Keyboard-First Navigation** - Vim-style keybindings (`hjkl`, `g/G`, etc.)
- **Command Palette** - Quick access to all features via `Ctrl+K`
- **Interactive Filtering** - Build complex WHERE clauses visually
- **JSONB Support** - View and navigate JSONB data with tree view
- **Query Favorites** - Save and organize frequently used queries
- **Auto-Discovery** - Automatically find local PostgreSQL instances
- **Mouse Support** - Click, scroll, and double-click interactions

## Installation

### From Source

```bash
git clone https://github.com/rebeliceyang/lazypg.git
cd lazypg
make build
./bin/lazypg
```

### Requirements

- Go 1.21+
- PostgreSQL 12+ (for connecting)

## Quick Start

1. Run `lazypg`
2. Press `c` to connect to a database
3. Navigate with `hjkl` or arrow keys
4. Press `Tab` to switch panels
5. Press `?` for help

## Key Bindings

| Key | Action |
|-----|--------|
| `hjkl` / Arrows | Navigate |
| `Tab` | Switch panels |
| `Enter` | Select / Expand |
| `Ctrl+K` | Command palette |
| `Ctrl+E` | SQL editor |
| `f` | Open filter builder |
| `j` | JSONB viewer (on JSONB cell) |
| `c` | Connect to database |
| `?` | Help |
| `q` | Quit |

See [documentation](docs/INDEX.md) for complete keyboard reference.

## Configuration

lazypg stores configuration in `~/.config/lazypg/`:

```yaml
# config.yaml
ui:
  theme: "default"
  panel_width_ratio: 25
  mouse_enabled: true
```

```yaml
# connections.yaml
connections:
  - name: "Local Dev"
    host: localhost
    port: 5432
    database: mydb
    user: postgres
```

## Documentation

- [Documentation Index](docs/INDEX.md)
- [Filtering Guide](docs/features/filtering.md)
- [JSONB Support](docs/features/jsonb.md)
- [Query Favorites](docs/features/favorites.md)
- [Development Guide](docs/DEVELOPMENT.md)

## Contributing

Contributions are welcome! Please read [DEVELOPMENT.md](docs/DEVELOPMENT.md) first.

```bash
make deps    # Install dependencies
make build   # Build binary
make test    # Run tests
make fmt     # Format code
```

## License

MIT License - see [LICENSE](LICENSE) for details.

## Acknowledgments

- Inspired by [lazygit](https://github.com/jesseduffield/lazygit)
- Built with [Bubble Tea](https://github.com/charmbracelet/bubbletea) and [Lip Gloss](https://github.com/charmbracelet/lipgloss)
- PostgreSQL driver: [pgx](https://github.com/jackc/pgx)
