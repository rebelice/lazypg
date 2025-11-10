package metadata

import (
	"context"
	"fmt"

	"github.com/rebeliceyang/lazypg/internal/db/connection"
)

// TableData represents paginated table data
type TableData struct {
	Columns   []string
	Rows      [][]string
	TotalRows int64
}

// QueryTableData fetches paginated table data
func QueryTableData(ctx context.Context, pool *connection.Pool, schema, table string, offset, limit int) (*TableData, error) {
	// First get total count
	countQuery := fmt.Sprintf("SELECT COUNT(*) as count FROM %s.%s", schema, table)
	countRow, err := pool.QueryRow(ctx, countQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to count rows: %w", err)
	}

	totalRows := int64(0)
	if count, ok := countRow["count"].(int64); ok {
		totalRows = count
	}

	// Query paginated data with columns in order
	query := fmt.Sprintf("SELECT * FROM %s.%s LIMIT %d OFFSET %d", schema, table, limit, offset)
	result, err := pool.QueryWithColumns(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query table data: %w", err)
	}

	if len(result.Rows) == 0 {
		return &TableData{
			Columns:   result.Columns,
			Rows:      [][]string{},
			TotalRows: totalRows,
		}, nil
	}

	columns := result.Columns

	// Convert rows to string slices
	data := make([][]string, len(result.Rows))
	for i, row := range result.Rows {
		rowData := make([]string, len(columns))
		for j, col := range columns {
			val := row[col]
			if val == nil {
				rowData[j] = "NULL"
			} else {
				rowData[j] = fmt.Sprintf("%v", val)
			}
		}
		data[i] = rowData
	}

	return &TableData{
		Columns:   columns,
		Rows:      data,
		TotalRows: totalRows,
	}, nil
}
