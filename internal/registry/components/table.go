//go:build vangoui

package ui

import (
	"github.com/vango-go/vango/pkg/vdom"
)

// TableOption configures a Table component.
type TableOption func(*tableConfig)

type tableConfig struct {
	className string
	children  []any
}

// TableClass adds additional CSS classes.
func TableClass(className string) TableOption {
	return func(c *tableConfig) {
		c.className = className
	}
}

// TableChildren sets the table children.
func TableChildren(children ...any) TableOption {
	return func(c *tableConfig) {
		c.children = children
	}
}

// Table renders a table element.
func Table(opts ...TableOption) *VNode {
	cfg := tableConfig{}
	for _, opt := range opts {
		opt(&cfg)
	}

	classes := CN("w-full caption-bottom text-sm", cfg.className)

	attrs := []any{Class(classes)}
	attrs = append(attrs, cfg.children...)

	return Div(
		Class("relative w-full overflow-auto"),
		El("table", attrs...),
	)
}

// TableHeaderOption configures a TableHeader component.
type TableHeaderOption func(*tableHeaderConfig)

type tableHeaderConfig struct {
	className string
	children  []any
}

// TableHeaderClass adds additional CSS classes.
func TableHeaderClass(className string) TableHeaderOption {
	return func(c *tableHeaderConfig) {
		c.className = className
	}
}

// TableHeaderChildren sets the header children.
func TableHeaderChildren(children ...any) TableHeaderOption {
	return func(c *tableHeaderConfig) {
		c.children = children
	}
}

// TableHeader renders a table header section.
func TableHeader(opts ...TableHeaderOption) *VNode {
	cfg := tableHeaderConfig{}
	for _, opt := range opts {
		opt(&cfg)
	}

	classes := CN("[&_tr]:border-b", cfg.className)

	attrs := []any{Class(classes)}
	attrs = append(attrs, cfg.children...)

	return El("thead", attrs...)
}

// TableBodyOption configures a TableBody component.
type TableBodyOption func(*tableBodyConfig)

type tableBodyConfig struct {
	className string
	children  []any
}

// TableBodyClass adds additional CSS classes.
func TableBodyClass(className string) TableBodyOption {
	return func(c *tableBodyConfig) {
		c.className = className
	}
}

// TableBodyChildren sets the body children.
func TableBodyChildren(children ...any) TableBodyOption {
	return func(c *tableBodyConfig) {
		c.children = children
	}
}

// TableBody renders a table body section.
func TableBody(opts ...TableBodyOption) *VNode {
	cfg := tableBodyConfig{}
	for _, opt := range opts {
		opt(&cfg)
	}

	classes := CN("[&_tr:last-child]:border-0", cfg.className)

	attrs := []any{Class(classes)}
	attrs = append(attrs, cfg.children...)

	return El("tbody", attrs...)
}

// TableFooterOption configures a TableFooter component.
type TableFooterOption func(*tableFooterConfig)

type tableFooterConfig struct {
	className string
	children  []any
}

// TableFooterClass adds additional CSS classes.
func TableFooterClass(className string) TableFooterOption {
	return func(c *tableFooterConfig) {
		c.className = className
	}
}

// TableFooterChildren sets the footer children.
func TableFooterChildren(children ...any) TableFooterOption {
	return func(c *tableFooterConfig) {
		c.children = children
	}
}

// TableFooter renders a table footer section.
func TableFooter(opts ...TableFooterOption) *VNode {
	cfg := tableFooterConfig{}
	for _, opt := range opts {
		opt(&cfg)
	}

	classes := CN("border-t bg-muted/50 font-medium [&>tr]:last:border-b-0", cfg.className)

	attrs := []any{Class(classes)}
	attrs = append(attrs, cfg.children...)

	return El("tfoot", attrs...)
}

// TableRowOption configures a TableRow component.
type TableRowOption func(*tableRowConfig)

type tableRowConfig struct {
	className string
	selected  bool
	children  []any
}

// TableRowClass adds additional CSS classes.
func TableRowClass(className string) TableRowOption {
	return func(c *tableRowConfig) {
		c.className = className
	}
}

// TableRowSelected sets the selected state.
func TableRowSelected(selected bool) TableRowOption {
	return func(c *tableRowConfig) {
		c.selected = selected
	}
}

// TableRowChildren sets the row children.
func TableRowChildren(children ...any) TableRowOption {
	return func(c *tableRowConfig) {
		c.children = children
	}
}

// TableRow renders a table row.
func TableRow(opts ...TableRowOption) *VNode {
	cfg := tableRowConfig{}
	for _, opt := range opts {
		opt(&cfg)
	}

	selectedClass := ""
	if cfg.selected {
		selectedClass = "bg-muted/50"
	}

	classes := CN(
		"border-b transition-colors hover:bg-muted/50 data-[state=selected]:bg-muted",
		selectedClass,
		cfg.className,
	)

	attrs := []any{Class(classes)}
	if cfg.selected {
		attrs = append(attrs, Data("state", "selected"))
	}
	attrs = append(attrs, cfg.children...)

	return El("tr", attrs...)
}

// TableHeadOption configures a TableHead component.
type TableHeadOption func(*tableHeadConfig)

type tableHeadConfig struct {
	className string
	sortable  bool
	sorted    string // "asc", "desc", or ""
	onSort    func()
	children  []any
}

// TableHeadClass adds additional CSS classes.
func TableHeadClass(className string) TableHeadOption {
	return func(c *tableHeadConfig) {
		c.className = className
	}
}

// TableHeadSortable makes the column sortable.
func TableHeadSortable(sortable bool) TableHeadOption {
	return func(c *tableHeadConfig) {
		c.sortable = sortable
	}
}

// TableHeadSorted sets the sort direction.
func TableHeadSorted(direction string) TableHeadOption {
	return func(c *tableHeadConfig) {
		c.sorted = direction
	}
}

// TableHeadOnSort sets the sort handler.
func TableHeadOnSort(handler func()) TableHeadOption {
	return func(c *tableHeadConfig) {
		c.onSort = handler
	}
}

// TableHeadChildren sets the header cell children.
func TableHeadChildren(children ...any) TableHeadOption {
	return func(c *tableHeadConfig) {
		c.children = children
	}
}

// TableHead renders a table header cell.
func TableHead(opts ...TableHeadOption) *VNode {
	cfg := tableHeadConfig{}
	for _, opt := range opts {
		opt(&cfg)
	}

	classes := CN(
		"h-12 px-4 text-left align-middle font-medium text-muted-foreground [&:has([role=checkbox])]:pr-0",
		cfg.className,
	)

	attrs := []any{Class(classes)}

	if cfg.sortable {
		attrs = append(attrs, Class("cursor-pointer select-none"))
		if cfg.onSort != nil {
			attrs = append(attrs, OnClick(cfg.onSort))
		}
	}

	// Add sort indicator
	var sortIndicator *VNode
	if cfg.sorted == "asc" {
		sortIndicator = El("svg",
			Class("ml-2 h-4 w-4 inline"),
			Attr{Key: "xmlns", Value: "http://www.w3.org/2000/svg"},
			Attr{Key: "viewBox", Value: "0 0 24 24"},
			Attr{Key: "fill", Value: "none"},
			Attr{Key: "stroke", Value: "currentColor"},
			Attr{Key: "stroke-width", Value: "2"},
			El("polyline", Attr{Key: "points", Value: "18 15 12 9 6 15"}),
		)
	} else if cfg.sorted == "desc" {
		sortIndicator = El("svg",
			Class("ml-2 h-4 w-4 inline"),
			Attr{Key: "xmlns", Value: "http://www.w3.org/2000/svg"},
			Attr{Key: "viewBox", Value: "0 0 24 24"},
			Attr{Key: "fill", Value: "none"},
			Attr{Key: "stroke", Value: "currentColor"},
			Attr{Key: "stroke-width", Value: "2"},
			El("polyline", Attr{Key: "points", Value: "6 9 12 15 18 9"}),
		)
	}

	attrs = append(attrs, cfg.children...)
	if sortIndicator != nil {
		attrs = append(attrs, sortIndicator)
	}

	return El("th", attrs...)
}

// TableCellOption configures a TableCell component.
type TableCellOption func(*tableCellConfig)

type tableCellConfig struct {
	className string
	children  []any
}

// TableCellClass adds additional CSS classes.
func TableCellClass(className string) TableCellOption {
	return func(c *tableCellConfig) {
		c.className = className
	}
}

// TableCellChildren sets the cell children.
func TableCellChildren(children ...any) TableCellOption {
	return func(c *tableCellConfig) {
		c.children = children
	}
}

// TableCell renders a table cell.
func TableCell(opts ...TableCellOption) *VNode {
	cfg := tableCellConfig{}
	for _, opt := range opts {
		opt(&cfg)
	}

	classes := CN("p-4 align-middle [&:has([role=checkbox])]:pr-0", cfg.className)

	attrs := []any{Class(classes)}
	attrs = append(attrs, cfg.children...)

	return El("td", attrs...)
}

// TableCaption renders a table caption.
func TableCaption(text string, className string) *VNode {
	classes := CN("mt-4 text-sm text-muted-foreground", className)
	return El("caption", Class(classes), Text(text))
}
