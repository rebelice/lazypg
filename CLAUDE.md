# Development Notes for AI Assistants

This document contains critical lessons learned and best practices for AI assistants working on this project.

## Critical: Bubble Tea / Lipgloss Width Calculation

**⚠️ ALWAYS account for padding and borders when calculating content width!**

### The Problem

When using `lipgloss` borders with `Width()` or `MaxWidth()`, the border and padding are rendered **outside** the content area, causing the total rendered width to exceed expectations and result in cut-off borders.

### Width Calculation Formula

```
Total Rendered Width = Content Width + Padding (left + right) + Border (left + right)
```

### Concrete Example

```go
containerStyle := lipgloss.NewStyle().
    Border(lipgloss.RoundedBorder()).  // +2 chars (1 left + 1 right)
    Padding(1, 2).                     // +4 chars (2 left + 2 right)
    MaxWidth(76)                       // Content max width

// Total rendered width = 76 + 4 + 2 = 82 chars
// This WILL overflow on an 80-char terminal!
```

### Safe Content Width Calculation

For a bordered container with padding that must fit within terminal width:

```go
// Target: Fit in 80-char terminal
// With Border(2) + Padding(4) = 6 extra chars
// Safe MaxWidth = 80 - 6 - safety_margin(4-10)
MaxWidth(70)  // Leaves 10-char safety margin for emojis, unicode, etc.
```

### Best Practices

1. **Use MaxWidth() instead of Width()**
   - `Width()` forces exact content width (can cause overflow)
   - `MaxWidth()` constrains maximum width (allows content to shrink)

2. **Conservative content width calculation**
   ```go
   safeContentWidth = targetWidth - padding - border - safetyMargin(8-10)
   ```

3. **Test with longest possible text**
   - Help text, titles, status messages
   - Include emojis (can be 2+ chars wide in terminal)
   - Account for Unicode characters

4. **Add width comments**
   ```go
   // Keep under 68 chars to fit MaxWidth(76) with Padding(1,2)
   helpText := "Short text here"
   ```

5. **Common pitfalls to avoid**
   - ❌ Setting `Width(80)` on bordered container
   - ❌ Not accounting for emoji width
   - ❌ Ignoring padding in calculations
   - ❌ Using exact measurements without safety margin

### Real Example from This Project

**Before (Border Cut Off):**
```go
containerStyle := lipgloss.NewStyle().
    Border(lipgloss.RoundedBorder()).
    Padding(1, 2).
    MaxWidth(76)

// Help text: 78 chars
sections = append(sections, "↑↓: Select  │  Tab: Switch  │  Enter: Connect  │  m: Manual  │  Esc: Cancel")
// Result: Right border cut off (76 + 4 + 2 = 82 chars total)
```

**After (Fixed):**
```go
containerStyle := lipgloss.NewStyle().
    Border(lipgloss.RoundedBorder()).
    Padding(1, 2).
    MaxWidth(76)

// Help text: 62 chars (14 char safety margin)
sections = append(sections, "↑↓: Navigate │ Tab: Switch │ Enter: Connect │ m: Manual")
// Result: Fits perfectly (76 + 4 + 2 = 82 → reduced to 72 effective width)
```

### Quick Reference Table

| Terminal Width | Border | Padding | Safe MaxWidth | Safe Content |
|----------------|--------|---------|---------------|--------------|
| 80             | 2      | 4       | 70            | 66           |
| 100            | 2      | 4       | 90            | 86           |
| 120            | 2      | 4       | 110           | 106          |

**Formula:** Safe Content = Terminal Width - Border(2) - Padding(4) - Safety Margin(4-10)

---

## Other Best Practices

### Code Organization

- Keep UI components in `internal/ui/components/`
- Keep business logic in `internal/` packages
- Use models package for shared data structures

### Error Handling

- Always log errors but don't crash on non-critical failures
- Provide user-friendly error messages in UI overlays
- Use `log.Printf("Warning: ...")` for non-fatal errors

### Testing

- Test UI changes at 80-char terminal width (most common)
- Test with actual PostgreSQL instances
- Verify password storage works across restarts

---

*Last updated: 2025-01-11*
