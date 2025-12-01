# lazypg Documentation

Complete documentation for lazypg, a modern Terminal User Interface (TUI) client for PostgreSQL.

## Getting Started

- [README](../README.md) - Project overview, installation, and quick start
- [Development Guide](DEVELOPMENT.md) - Setup development environment and workflow
- [Roadmap](ROADMAP.md) - Planned features and improvements

## User Features

- [Interactive Filtering](features/filtering.md) - Build complex filters with type-aware operators
- [JSONB Support](features/jsonb.md) - View, navigate, and filter JSONB data
- [Query Favorites](features/favorites.md) - Save, organize, and execute frequently used queries

## Quick Reference

### Keyboard Shortcuts

#### Global
| Key | Action |
|-----|--------|
| `Ctrl+P` | Open command palette |
| `Tab` | Switch between panels |
| `?` | Show help |
| `q` | Quit |
| `Esc` | Close dialogs/cancel |

#### Navigation
| Key | Action |
|-----|--------|
| `j`/`k` or `↑`/`↓` | Move up/down |
| `h`/`l` or `←`/`→` | Move left/right |
| `g`/`G` | Jump to top/bottom |
| `Ctrl+D`/`Ctrl+U` | Page down/up |

#### Data Browsing
| Key | Action |
|-----|--------|
| `Enter` | Select/expand node |
| `c` | Open connection dialog |
| `r` | Refresh view |

#### Filtering
| Key | Action |
|-----|--------|
| `f` | Open filter builder |
| `Ctrl+F` | Quick filter from cell |
| `Ctrl+R` | Clear filters |

#### JSONB Viewer
| Key | Action |
|-----|--------|
| `j` | Open JSONB viewer |
| `1`/`2`/`3` | Switch view mode |

### Configuration Files

| File | Location | Purpose |
|------|----------|---------|
| `config.yaml` | `~/.config/lazypg/` | User settings |
| `connections.yaml` | `~/.config/lazypg/` | Saved connections |
| `favorites.yaml` | `~/.config/lazypg/` | Query favorites |

## Contributing

See [DEVELOPMENT.md](DEVELOPMENT.md) for development guidelines.

## License

MIT License - see [LICENSE](../LICENSE) for details.
