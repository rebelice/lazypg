# Data View 行号显示与 Vim 跳转设计

## 概述

为 TableView 添加行号列，支持绝对/相对行号切换，以及完整的 Vim 风格跳转。

## 功能特性

- 行号列显示在数据最左侧
- 默认显示绝对行号，`Ctrl+n` 切换相对行号模式
- 支持 `{number}G`、`{number}j`、`{number}k`、`gg`、`G` 跳转
- 当前行行号高亮显示

## 视觉设计

### 绝对行号模式（默认）

```
   1 │ id   │ name     │ data
   2 │ 130  │ TESTUSER │ {...}
  >3 │ 1058 │ bbdata.. │ {...}   <- 当前行，行号高亮
   4 │ 1134 │ UTF8...  │ {...}
```

### 相对行号模式

```
   2 │ id   │ name     │ data
   1 │ 130  │ TESTUSER │ {...}
  >3 │ 1058 │ bbdata.. │ {...}   <- 显示绝对行号
   1 │ 1134 │ UTF8...  │ {...}
   2 │ 1129 │ danny... │ {...}
```

### 行号列宽度

根据总行数动态计算（如 134 行需要 3 位 + 1 空格 + 分隔符）

## Vim 跳转实现

### 数字前缀输入机制

- 用户输入数字时累积到 `pendingCount` 缓冲区（如输入 `4` `2` 得到 42）
- 输入动作键（`G`/`j`/`k`）时执行跳转并清空缓冲区
- 超时 1.5 秒或按其他键自动清空
- 在状态栏显示当前输入的数字（如 `42_`）

### 支持的跳转命令

| 命令 | 动作 |
|------|------|
| `gg` | 跳转到第一行 |
| `G` | 跳转到最后一行 |
| `{n}G` | 跳转到第 n 行 |
| `{n}j` | 向下移动 n 行 |
| `{n}k` | 向上移动 n 行 |

### 边界处理

- 超出范围时跳转到最近的有效行（第一行或最后一行）
- 数据需要从服务器加载时，触发数据加载

## 代码结构

### TableView 新增字段

```go
type TableView struct {
    // ... 现有字段 ...

    // 行号显示
    ShowLineNumbers   bool  // 是否显示行号（默认 true）
    RelativeNumbers   bool  // 是否相对行号模式（默认 false）

    // Vim 跳转
    PendingCount      string    // 数字前缀缓冲区（如 "42"）
    PendingCountTime  time.Time // 上次输入时间，用于超时清空
}
```

### 修改的函数

- `View()` - 在渲染行前添加行号列
- `renderRow()` - 新增 `renderLineNumber()` 调用
- `calculateColumnWidths()` - 预留行号列宽度

### 新增函数

- `renderLineNumber(rowIndex int) string` - 渲染单个行号
- `getLineNumberWidth() int` - 计算行号列宽度
- `ToggleRelativeNumbers()` - 切换相对/绝对行号
- `HandleVimMotion(key string) bool` - 处理 Vim 跳转输入
- `ExecuteJump(count int, motion string)` - 执行跳转

## 快捷键映射

| 按键 | 动作 | 条件 |
|------|------|------|
| `Ctrl+n` | 切换相对/绝对行号 | Data tab 激活时 |
| `0-9` | 累积数字前缀 | Data tab 激活时 |
| `g` | 等待第二个 `g`（用于 `gg`） | Data tab 激活时 |
| `G` | 执行跳转（末行或第 n 行） | Data tab 激活时 |
| `j`/`k` | 移动 n 行或 1 行 | Data tab 激活时 |

## 状态栏提示

- 有数字前缀时显示：`42_` 表示等待动作
- 输入 `g` 后显示：`g_` 表示等待第二个 `g`

---

*设计日期: 2025-01-28*
