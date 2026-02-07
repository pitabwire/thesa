package model

// NavigationTree is the top-level navigation structure returned to the frontend.
type NavigationTree struct {
	Items []NavigationNode `json:"items"`
}

// NavigationNode is a single node in the navigation tree.
type NavigationNode struct {
	ID       string           `json:"id"`
	Label    string           `json:"label"`
	Icon     string           `json:"icon"`
	Route    string           `json:"route,omitempty"`
	Children []NavigationNode `json:"children"`
	Badge    *BadgeDescriptor `json:"badge,omitempty"`
}

// BadgeDescriptor describes a count badge on a navigation item.
type BadgeDescriptor struct {
	Count int    `json:"count"`
	Style string `json:"style"`
}

// PageDescriptor is the resolved page sent to the frontend.
type PageDescriptor struct {
	ID              string              `json:"id"`
	Title           string              `json:"title"`
	Route           string              `json:"route"`
	Layout          string              `json:"layout"`
	RefreshInterval int                 `json:"refresh_interval"`
	Breadcrumb      []BreadcrumbDescriptor `json:"breadcrumb,omitempty"`
	Table           *TableDescriptor    `json:"table,omitempty"`
	Sections        []SectionDescriptor `json:"sections,omitempty"`
	Actions         []ActionDescriptor  `json:"actions,omitempty"`
}

// BreadcrumbDescriptor is a single breadcrumb entry.
type BreadcrumbDescriptor struct {
	Label string `json:"label"`
	Route string `json:"route,omitempty"`
}

// TableDescriptor is the resolved table metadata sent to the frontend.
type TableDescriptor struct {
	Columns      []ColumnDescriptor `json:"columns"`
	Filters      []FilterDescriptor `json:"filters,omitempty"`
	RowActions   []ActionDescriptor `json:"row_actions,omitempty"`
	BulkActions  []ActionDescriptor `json:"bulk_actions,omitempty"`
	DataEndpoint string             `json:"data_endpoint"`
	DefaultSort  string             `json:"default_sort,omitempty"`
	SortDir      string             `json:"sort_dir,omitempty"`
	PageSize     int                `json:"page_size"`
	Selectable   bool               `json:"selectable"`
}

// ColumnDescriptor describes a visible table column.
type ColumnDescriptor struct {
	Field     string            `json:"field"`
	Label     string            `json:"label"`
	Type      string            `json:"type"`
	Sortable  bool              `json:"sortable"`
	Format    string            `json:"format,omitempty"`
	Width     string            `json:"width,omitempty"`
	Link      *LinkDescriptor   `json:"link,omitempty"`
	StatusMap map[string]string `json:"status_map,omitempty"`
}

// LinkDescriptor describes a clickable link.
type LinkDescriptor struct {
	Route  string            `json:"route"`
	Params map[string]string `json:"params,omitempty"`
}

// FilterDescriptor describes a resolved filter control.
type FilterDescriptor struct {
	Field    string             `json:"field"`
	Label    string             `json:"label"`
	Type     string             `json:"type"`
	Operator string             `json:"operator"`
	Options  []OptionDescriptor `json:"options,omitempty"`
	Default  any                `json:"default,omitempty"`
}

// OptionDescriptor is a resolved option for dropdowns and filters.
type OptionDescriptor struct {
	Label string `json:"label"`
	Value string `json:"value"`
	Icon  string `json:"icon,omitempty"`
}

// FormDescriptor is the resolved form sent to the frontend.
type FormDescriptor struct {
	ID             string              `json:"id"`
	Title          string              `json:"title"`
	Sections       []SectionDescriptor `json:"sections"`
	Actions        []ActionDescriptor  `json:"actions,omitempty"`
	SubmitEndpoint string              `json:"submit_endpoint"`
	SuccessRoute   string              `json:"success_route,omitempty"`
	SuccessMessage string              `json:"success_message,omitempty"`
}

// SectionDescriptor is a resolved section.
type SectionDescriptor struct {
	ID          string            `json:"id"`
	Title       string            `json:"title"`
	Layout      string            `json:"layout"`
	Columns     int               `json:"columns,omitempty"`
	Collapsible bool              `json:"collapsible"`
	Collapsed   bool              `json:"collapsed"`
	Fields      []FieldDescriptor `json:"fields"`
}

// FieldDescriptor is a resolved field sent to the frontend.
type FieldDescriptor struct {
	Field       string                       `json:"field"`
	Label       string                       `json:"label"`
	Type        string                       `json:"type"`
	ReadOnly    bool                         `json:"read_only"`
	Required    bool                         `json:"required"`
	Validation  *ValidationDescriptor        `json:"validation,omitempty"`
	Options     []OptionDescriptor           `json:"options,omitempty"`
	Format      string                       `json:"format,omitempty"`
	Placeholder string                       `json:"placeholder,omitempty"`
	HelpText    string                       `json:"help_text,omitempty"`
	Span        int                          `json:"span,omitempty"`
	Value       any                          `json:"value,omitempty"`
	DependsOn   []FieldDependencyDescriptor  `json:"depends_on,omitempty"`
}

// ValidationDescriptor describes client-side validation rules.
type ValidationDescriptor struct {
	MinLength *int     `json:"min_length,omitempty"`
	MaxLength *int     `json:"max_length,omitempty"`
	Min       *float64 `json:"min,omitempty"`
	Max       *float64 `json:"max,omitempty"`
	Pattern   string   `json:"pattern,omitempty"`
	Message   string   `json:"message,omitempty"`
}

// FieldDependencyDescriptor describes a client-side field dependency.
type FieldDependencyDescriptor struct {
	Field     string `json:"field"`
	Condition string `json:"condition"`
	Value     string `json:"value,omitempty"`
}

// ActionDescriptor is a resolved action sent to the frontend.
type ActionDescriptor struct {
	ID           string                  `json:"id"`
	Label        string                  `json:"label"`
	Icon         string                  `json:"icon,omitempty"`
	Style        string                  `json:"style,omitempty"`
	Type         string                  `json:"type"`
	Enabled      bool                    `json:"enabled"`
	Visible      bool                    `json:"visible"`
	CommandID    string                  `json:"command_id,omitempty"`
	NavigateTo   string                  `json:"navigate_to,omitempty"`
	WorkflowID   string                  `json:"workflow_id,omitempty"`
	FormID       string                  `json:"form_id,omitempty"`
	Confirmation *ConfirmationDescriptor `json:"confirmation,omitempty"`
	Conditions   []ConditionDescriptor   `json:"conditions,omitempty"`
	Params       map[string]string       `json:"params,omitempty"`
}

// ConfirmationDescriptor describes a confirmation dialog.
type ConfirmationDescriptor struct {
	Title   string `json:"title"`
	Message string `json:"message"`
	Confirm string `json:"confirm"`
	Cancel  string `json:"cancel,omitempty"`
	Style   string `json:"style,omitempty"`
}

// ConditionDescriptor describes a client-side data-dependent condition.
type ConditionDescriptor struct {
	Field    string `json:"field"`
	Operator string `json:"operator"`
	Value    any    `json:"value,omitempty"`
	Effect   string `json:"effect"`
}

// WorkflowDescriptor is the resolved workflow instance sent to the frontend.
type WorkflowDescriptor struct {
	ID          string           `json:"id"`
	WorkflowID  string           `json:"workflow_id"`
	Name        string           `json:"name"`
	Status      string           `json:"status"`
	CurrentStep *StepDescriptor  `json:"current_step,omitempty"`
	Steps       []StepSummary    `json:"steps"`
	History     []HistoryEntry   `json:"history,omitempty"`
}

// StepDescriptor describes the current active step.
type StepDescriptor struct {
	ID      string             `json:"id"`
	Name    string             `json:"name"`
	Type    string             `json:"type"`
	Status  string             `json:"status"`
	Form    *FormDescriptor    `json:"form,omitempty"`
	Actions []ActionDescriptor `json:"actions,omitempty"`
}

// StepSummary is a summary of a workflow step shown in the progress indicator.
type StepSummary struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Type   string `json:"type"`
	Status string `json:"status"`
}

// HistoryEntry is an audit trail entry for a workflow.
type HistoryEntry struct {
	StepName  string `json:"step_name"`
	Event     string `json:"event"`
	Actor     string `json:"actor"`
	Timestamp string `json:"timestamp"`
	Comment   string `json:"comment,omitempty"`
}

// DataResponse is the standardized data response for list pages.
type DataResponse struct {
	Data DataPayload    `json:"data"`
	Meta map[string]any `json:"meta,omitempty"`
}

// DataPayload contains the items and pagination for a data response.
type DataPayload struct {
	Items      []map[string]any `json:"items"`
	TotalCount int              `json:"total_count"`
	Page       int              `json:"page"`
	PageSize   int              `json:"page_size"`
}

// CommandResponse is the response from executing a command.
type CommandResponse struct {
	Success bool            `json:"success"`
	Message string          `json:"message,omitempty"`
	Result  map[string]any  `json:"result,omitempty"`
	Errors  []FieldError    `json:"errors,omitempty"`
}

// SearchResponse is the response from a global search query.
type SearchResponse struct {
	Data SearchPayload  `json:"data"`
	Meta map[string]any `json:"meta,omitempty"`
}

// SearchPayload contains the search results.
type SearchPayload struct {
	Results    []SearchResult `json:"results"`
	TotalCount int            `json:"total_count"`
	Query      string         `json:"query"`
}

// SearchResult is a single search result item.
type SearchResult struct {
	ID       string  `json:"id"`
	Title    string  `json:"title"`
	Subtitle string  `json:"subtitle,omitempty"`
	Category string  `json:"category"`
	Icon     string  `json:"icon,omitempty"`
	Route    string  `json:"route"`
	Score    float64 `json:"score"`
}

// LookupResponse is the response from a lookup endpoint.
type LookupResponse struct {
	Data LookupPayload  `json:"data"`
	Meta map[string]any `json:"meta,omitempty"`
}

// LookupPayload contains the lookup options.
type LookupPayload struct {
	Options []OptionDescriptor `json:"options"`
}
