package utils

import (
	"io"
	"os"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
)

type Table struct {
	writer table.Writer
}

func NewTable(headers []string) *Table {
	t := &Table{
		writer: table.NewWriter(),
	}

	// Set default output to standard output
	t.writer.SetOutputMirror(os.Stdout)

	// Enable auto index
	t.writer.SetAutoIndex(true)

	// Set table headers
	headerRow := make(table.Row, len(headers))
	for i, h := range headers {
		headerRow[i] = h
	}
	t.writer.AppendHeader(headerRow)

	// Set table style
	t.writer.SetStyle(table.StyleLight)

	// Configure column properties
	configs := make([]table.ColumnConfig, len(headers))
	for i, header := range headers {
		var colors text.Colors
		switch header {
		case "名称": // Name
			colors = text.Colors{text.FgHiBlue}
		case "运营商": // Service Provider
			colors = text.Colors{text.FgHiYellow}
		case "节点类型": // Node Type
			colors = text.Colors{text.FgHiCyan}
		case "节点ID":
			colors = text.Colors{text.FgHiMagenta}
		default:
			colors = text.Colors{text.FgWhite}
		}

		configs[i] = table.ColumnConfig{
			Name:         header,
			Colors:       colors,
			ColorsHeader: text.Colors{text.Bold, colors[0]},
			Align:        text.AlignLeft,
			VAlign:       text.VAlignMiddle,
			WidthMax:     50,
		}
	}
	t.writer.SetColumnConfigs(configs)

	return t
}

// SetOutput sets the output destination
func (t *Table) SetOutput(w io.Writer) {
	t.writer.SetOutputMirror(w)
}

// AddRow adds a row of data
func (t *Table) AddRow(row []string) {
	tableRow := make(table.Row, len(row))
	for i, cell := range row {
		tableRow[i] = cell
	}
	t.writer.AppendRow(tableRow)
}

// AddSeparator adds a separator row
func (t *Table) AddSeparator() {
	t.writer.AppendSeparator()
}

// SetPageSize sets the number of rows displayed per page
func (t *Table) SetPageSize(size int) {
	t.writer.SetPageSize(size)
}

// EnableAutoMerge enables automatic cell merging
func (t *Table) EnableAutoMerge() {
	t.writer.SetAutoIndex(true)
	t.writer.SetAllowedRowLength(100)
}

// SortBy sorts by specified columns
func (t *Table) SortBy(columnNames []string) {
	sortBy := make([]table.SortBy, len(columnNames))
	for i, name := range columnNames {
		sortBy[i] = table.SortBy{Name: name, Mode: table.Asc}
	}
	t.writer.SortBy(sortBy)
}

// Print renders the table
func (t *Table) Print() {
	t.writer.Render()
}

// RenderHTML outputs HTML format
func (t *Table) RenderHTML() string {
	return t.writer.RenderHTML()
}

// RenderMarkdown outputs Markdown format
func (t *Table) RenderMarkdown() string {
	return t.writer.RenderMarkdown()
}

// RenderCSV outputs CSV format
func (t *Table) RenderCSV() string {
	return t.writer.RenderCSV()
}
