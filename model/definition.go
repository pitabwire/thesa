package model

// DomainDefinition is the root structure of a definition file. Each file
// declares one domain's pages, forms, commands, workflows, searches, and lookups.
type DomainDefinition struct {
	Domain     string               `yaml:"domain"     json:"domain"`
	Version    string               `yaml:"version"    json:"version"`
	Navigation NavigationDefinition `yaml:"navigation" json:"navigation"`
	Pages      []PageDefinition     `yaml:"pages"      json:"pages,omitempty"`
	Forms      []FormDefinition     `yaml:"forms"      json:"forms,omitempty"`
	Commands   []CommandDefinition  `yaml:"commands"    json:"commands,omitempty"`
	Workflows  []WorkflowDefinition `yaml:"workflows"   json:"workflows,omitempty"`
	Searches   []SearchDefinition   `yaml:"searches"    json:"searches,omitempty"`
	Lookups    []LookupDefinition   `yaml:"lookups"     json:"lookups,omitempty"`

	// Checksum is computed at load time and not part of the YAML.
	Checksum string `yaml:"-" json:"-"`
	// SourceFile records the originating file path.
	SourceFile string `yaml:"-" json:"-"`
}

// NavigationDefinition describes a domain's menu entry.
type NavigationDefinition struct {
	Label        string                      `yaml:"label"        json:"label"`
	Icon         string                      `yaml:"icon"         json:"icon"`
	Order        int                         `yaml:"order"        json:"order"`
	Capabilities []string                    `yaml:"capabilities" json:"capabilities"`
	Children     []NavigationChildDefinition `yaml:"children"     json:"children"`
}

// NavigationChildDefinition describes a child navigation item in the menu.
type NavigationChildDefinition struct {
	Label        string          `yaml:"label"        json:"label"`
	Icon         string          `yaml:"icon"         json:"icon,omitempty"`
	Route        string          `yaml:"route"        json:"route"`
	PageID       string          `yaml:"page_id"      json:"page_id"`
	Capabilities []string        `yaml:"capabilities" json:"capabilities"`
	Order        int             `yaml:"order"        json:"order"`
	Badge        *BadgeDefinition `yaml:"badge"        json:"badge,omitempty"`
}

// BadgeDefinition describes a count badge on a navigation item.
type BadgeDefinition struct {
	OperationID string `yaml:"operation_id" json:"operation_id"`
	Field       string `yaml:"field"        json:"field"`
	Style       string `yaml:"style"        json:"style"`
}

// PageDefinition describes a page visible in the UI.
type PageDefinition struct {
	ID              string              `yaml:"id"               json:"id"`
	Title           string              `yaml:"title"            json:"title"`
	Route           string              `yaml:"route"            json:"route"`
	Layout          string              `yaml:"layout"           json:"layout"`
	Capabilities    []string            `yaml:"capabilities"     json:"capabilities"`
	RefreshInterval int                 `yaml:"refresh_interval" json:"refresh_interval,omitempty"`
	Breadcrumb      []BreadcrumbItem    `yaml:"breadcrumb"       json:"breadcrumb,omitempty"`
	Table           *TableDefinition    `yaml:"table"            json:"table,omitempty"`
	Sections        []SectionDefinition `yaml:"sections"         json:"sections,omitempty"`
	Actions         []ActionDefinition  `yaml:"actions"          json:"actions,omitempty"`
}

// BreadcrumbItem is a single entry in a breadcrumb trail.
type BreadcrumbItem struct {
	Label string `yaml:"label" json:"label"`
	Route string `yaml:"route" json:"route,omitempty"`
}

// TableDefinition describes a data table within a list page.
type TableDefinition struct {
	DataSource  DataSourceDefinition `yaml:"data_source"  json:"data_source"`
	Columns     []ColumnDefinition   `yaml:"columns"      json:"columns"`
	Filters     []FilterDefinition   `yaml:"filters"      json:"filters,omitempty"`
	RowActions  []ActionDefinition   `yaml:"row_actions"  json:"row_actions,omitempty"`
	BulkActions []ActionDefinition   `yaml:"bulk_actions" json:"bulk_actions,omitempty"`
	DefaultSort string               `yaml:"default_sort" json:"default_sort,omitempty"`
	SortDir     string               `yaml:"sort_dir"     json:"sort_dir,omitempty"`
	PageSize    int                  `yaml:"page_size"    json:"page_size,omitempty"`
	Selectable  bool                 `yaml:"selectable"   json:"selectable,omitempty"`
}

// DataSourceDefinition describes how to fetch data from a backend service.
type DataSourceDefinition struct {
	OperationID string                `yaml:"operation_id" json:"operation_id,omitempty"`
	ServiceID   string                `yaml:"service_id"   json:"service_id,omitempty"`
	Handler     string                `yaml:"handler"      json:"handler,omitempty"`
	Mapping     ResponseMappingDefinition `yaml:"mapping"      json:"mapping"`
}

// ResponseMappingDefinition describes how to transform a backend response.
type ResponseMappingDefinition struct {
	ItemsPath string            `yaml:"items_path" json:"items_path"`
	TotalPath string            `yaml:"total_path" json:"total_path,omitempty"`
	FieldMap  map[string]string `yaml:"field_map"  json:"field_map,omitempty"`
}

// ColumnDefinition describes a table column.
type ColumnDefinition struct {
	Field      string            `yaml:"field"      json:"field"`
	Label      string            `yaml:"label"      json:"label"`
	Type       string            `yaml:"type"       json:"type"`
	Sortable   bool              `yaml:"sortable"   json:"sortable,omitempty"`
	Filterable bool              `yaml:"filterable" json:"filterable,omitempty"`
	Visible    string            `yaml:"visible"    json:"visible,omitempty"`
	Format     string            `yaml:"format"     json:"format,omitempty"`
	Width      string            `yaml:"width"      json:"width,omitempty"`
	Link       *LinkDefinition   `yaml:"link"       json:"link,omitempty"`
	StatusMap  map[string]string `yaml:"status_map" json:"status_map,omitempty"`
}

// LinkDefinition describes a clickable link within a table cell.
type LinkDefinition struct {
	Route  string            `yaml:"route"  json:"route"`
	Params map[string]string `yaml:"params" json:"params,omitempty"`
}

// FilterDefinition describes a filter control above a table.
type FilterDefinition struct {
	Field    string                  `yaml:"field"    json:"field"`
	Label    string                  `yaml:"label"    json:"label"`
	Type     string                  `yaml:"type"     json:"type"`
	Operator string                  `yaml:"operator" json:"operator"`
	Options  *FilterOptionsDefinition `yaml:"options"  json:"options,omitempty"`
	Default  string                  `yaml:"default"  json:"default,omitempty"`
}

// FilterOptionsDefinition describes options for select/multi-select filters.
type FilterOptionsDefinition struct {
	LookupID string              `yaml:"lookup_id" json:"lookup_id,omitempty"`
	Static   []StaticOption      `yaml:"static"    json:"static,omitempty"`
}

// StaticOption is a label/value pair for dropdowns and filters.
type StaticOption struct {
	Label string `yaml:"label" json:"label"`
	Value string `yaml:"value" json:"value"`
}

// SectionDefinition describes a section within a page or form.
type SectionDefinition struct {
	ID           string            `yaml:"id"           json:"id"`
	Title        string            `yaml:"title"        json:"title"`
	Layout       string            `yaml:"layout"       json:"layout"`
	Columns      int               `yaml:"columns"      json:"columns,omitempty"`
	Capabilities []string          `yaml:"capabilities" json:"capabilities,omitempty"`
	Collapsible  bool              `yaml:"collapsible"  json:"collapsible,omitempty"`
	Collapsed    bool              `yaml:"collapsed"    json:"collapsed,omitempty"`
	Fields       []FieldDefinition `yaml:"fields"       json:"fields"`
}

// FieldDefinition describes a single field in a section or form.
type FieldDefinition struct {
	Field       string                  `yaml:"field"       json:"field"`
	Label       string                  `yaml:"label"       json:"label"`
	Type        string                  `yaml:"type"        json:"type"`
	ReadOnly    string                  `yaml:"read_only"   json:"read_only,omitempty"`
	Required    bool                    `yaml:"required"    json:"required,omitempty"`
	Validation  *ValidationDefinition   `yaml:"validation"  json:"validation,omitempty"`
	Lookup      *LookupRefDefinition    `yaml:"lookup"      json:"lookup,omitempty"`
	Visibility  string                  `yaml:"visibility"  json:"visibility,omitempty"`
	Format      string                  `yaml:"format"      json:"format,omitempty"`
	Placeholder string                  `yaml:"placeholder" json:"placeholder,omitempty"`
	HelpText    string                  `yaml:"help_text"   json:"help_text,omitempty"`
	Span        int                     `yaml:"span"        json:"span,omitempty"`
	DependsOn   []FieldDependency       `yaml:"depends_on"  json:"depends_on,omitempty"`
}

// ValidationDefinition describes validation rules for a field.
type ValidationDefinition struct {
	MinLength *int    `yaml:"min_length" json:"min_length,omitempty"`
	MaxLength *int    `yaml:"max_length" json:"max_length,omitempty"`
	Min       *float64 `yaml:"min"        json:"min,omitempty"`
	Max       *float64 `yaml:"max"        json:"max,omitempty"`
	Pattern   string  `yaml:"pattern"    json:"pattern,omitempty"`
	Message   string  `yaml:"message"    json:"message,omitempty"`
}

// LookupRefDefinition references a LookupDefinition or provides inline options.
type LookupRefDefinition struct {
	LookupID string         `yaml:"lookup_id" json:"lookup_id,omitempty"`
	Static   []StaticOption `yaml:"static"    json:"static,omitempty"`
}

// FieldDependency describes a dependency between fields.
type FieldDependency struct {
	Field     string `yaml:"field"     json:"field"`
	Condition string `yaml:"condition" json:"condition"`
	Value     string `yaml:"value"     json:"value,omitempty"`
}

// FormDefinition describes an input form.
type FormDefinition struct {
	ID             string              `yaml:"id"              json:"id"`
	Title          string              `yaml:"title"           json:"title"`
	Capabilities   []string            `yaml:"capabilities"    json:"capabilities"`
	SubmitCommand  string              `yaml:"submit_command"  json:"submit_command"`
	LoadSource     *DataSourceDefinition `yaml:"load_source"     json:"load_source,omitempty"`
	SuccessRoute   string              `yaml:"success_route"   json:"success_route,omitempty"`
	SuccessMessage string              `yaml:"success_message" json:"success_message,omitempty"`
	Sections       []SectionDefinition `yaml:"sections"        json:"sections"`
}

// ActionDefinition describes a UI action (button, menu item).
type ActionDefinition struct {
	ID           string                   `yaml:"id"           json:"id"`
	Label        string                   `yaml:"label"        json:"label"`
	Icon         string                   `yaml:"icon"         json:"icon,omitempty"`
	Style        string                   `yaml:"style"        json:"style,omitempty"`
	Capabilities []string                 `yaml:"capabilities" json:"capabilities"`
	Type         string                   `yaml:"type"         json:"type"`
	CommandID    string                   `yaml:"command_id"   json:"command_id,omitempty"`
	NavigateTo   string                   `yaml:"navigate_to"  json:"navigate_to,omitempty"`
	WorkflowID   string                   `yaml:"workflow_id"  json:"workflow_id,omitempty"`
	FormID       string                   `yaml:"form_id"      json:"form_id,omitempty"`
	Confirmation *ConfirmationDefinition  `yaml:"confirmation" json:"confirmation,omitempty"`
	Conditions   []ConditionDefinition    `yaml:"conditions"   json:"conditions,omitempty"`
	Params       map[string]string        `yaml:"params"       json:"params,omitempty"`
}

// ConfirmationDefinition describes a confirmation dialog.
type ConfirmationDefinition struct {
	Title   string `yaml:"title"   json:"title"`
	Message string `yaml:"message" json:"message"`
	Confirm string `yaml:"confirm" json:"confirm"`
	Cancel  string `yaml:"cancel"  json:"cancel,omitempty"`
	Style   string `yaml:"style"   json:"style,omitempty"`
}

// ConditionDefinition describes a data-dependent visibility/enablement rule.
type ConditionDefinition struct {
	Field    string `yaml:"field"    json:"field"`
	Operator string `yaml:"operator" json:"operator"`
	Value    any    `yaml:"value"    json:"value,omitempty"`
	Effect   string `yaml:"effect"   json:"effect"`
}

// CommandDefinition describes a mutable operation.
type CommandDefinition struct {
	ID           string              `yaml:"id"           json:"id"`
	Capabilities []string            `yaml:"capabilities" json:"capabilities"`
	Operation    OperationBinding    `yaml:"operation"    json:"operation"`
	Input        InputMapping        `yaml:"input"        json:"input"`
	Output       OutputMapping       `yaml:"output"       json:"output"`
	Idempotency  *IdempotencyConfig  `yaml:"idempotency"  json:"idempotency,omitempty"`
	RateLimit    *RateLimitConfig    `yaml:"rate_limit"   json:"rate_limit,omitempty"`
}

// OperationBinding describes the backend operation to invoke.
type OperationBinding struct {
	Type        string `yaml:"type"         json:"type"`
	OperationID string `yaml:"operation_id" json:"operation_id,omitempty"`
	ServiceID   string `yaml:"service_id"   json:"service_id,omitempty"`
	Handler     string `yaml:"handler"      json:"handler,omitempty"`
}

// InputMapping describes how to map frontend input to a backend request.
type InputMapping struct {
	PathParams      map[string]string `yaml:"path_params"       json:"path_params,omitempty"`
	QueryParams     map[string]string `yaml:"query_params"      json:"query_params,omitempty"`
	HeaderParams    map[string]string `yaml:"header_params"     json:"header_params,omitempty"`
	BodyMapping     string            `yaml:"body_mapping"      json:"body_mapping"`
	BodyTemplate    map[string]string `yaml:"body_template"     json:"body_template,omitempty"`
	FieldProjection map[string]string `yaml:"field_projection"  json:"field_projection,omitempty"`
}

// OutputMapping describes how to transform a backend response for the frontend.
type OutputMapping struct {
	Type           string            `yaml:"type"            json:"type"`
	Fields         map[string]string `yaml:"fields"          json:"fields,omitempty"`
	ErrorMap       map[string]string `yaml:"error_map"       json:"error_map,omitempty"`
	SuccessMessage string            `yaml:"success_message" json:"success_message,omitempty"`
}

// IdempotencyConfig describes idempotency settings for a command.
type IdempotencyConfig struct {
	KeySource string `yaml:"key_source" json:"key_source"`
	TTL       string `yaml:"ttl"        json:"ttl"`
}

// RateLimitConfig describes rate limiting for a command.
type RateLimitConfig struct {
	MaxRequests int    `yaml:"max_requests" json:"max_requests"`
	Window      string `yaml:"window"       json:"window"`
	Scope       string `yaml:"scope"        json:"scope"`
}

// WorkflowDefinition describes a multi-step process.
type WorkflowDefinition struct {
	ID          string                 `yaml:"id"           json:"id"`
	Name        string                 `yaml:"name"         json:"name"`
	Capabilities []string              `yaml:"capabilities" json:"capabilities"`
	InitialStep string                 `yaml:"initial_step" json:"initial_step"`
	Timeout     string                 `yaml:"timeout"      json:"timeout,omitempty"`
	OnTimeout   string                 `yaml:"on_timeout"   json:"on_timeout,omitempty"`
	Steps       []StepDefinition       `yaml:"steps"        json:"steps"`
	Transitions []TransitionDefinition `yaml:"transitions"  json:"transitions"`
}

// StepDefinition describes a single step in a workflow.
type StepDefinition struct {
	ID           string           `yaml:"id"           json:"id"`
	Name         string           `yaml:"name"         json:"name"`
	Type         string           `yaml:"type"         json:"type"`
	Capabilities []string         `yaml:"capabilities" json:"capabilities,omitempty"`
	FormID       string           `yaml:"form_id"      json:"form_id,omitempty"`
	Operation    *OperationBinding `yaml:"operation"    json:"operation,omitempty"`
	Input        *InputMapping    `yaml:"input"        json:"input,omitempty"`
	Output       *OutputMapping   `yaml:"output"       json:"output,omitempty"`
	Timeout      string           `yaml:"timeout"      json:"timeout,omitempty"`
	OnTimeout    string           `yaml:"on_timeout"   json:"on_timeout,omitempty"`
	Assignee     *AssigneeConfig  `yaml:"assignee"     json:"assignee,omitempty"`
}

// AssigneeConfig describes who is responsible for a workflow step.
type AssigneeConfig struct {
	Type  string `yaml:"type"  json:"type"`
	Value string `yaml:"value" json:"value"`
}

// TransitionDefinition describes a transition between workflow steps.
type TransitionDefinition struct {
	From      string `yaml:"from"      json:"from"`
	To        string `yaml:"to"        json:"to"`
	Event     string `yaml:"event"     json:"event"`
	Condition string `yaml:"condition" json:"condition,omitempty"`
	Guard     string `yaml:"guard"     json:"guard,omitempty"`
}

// SearchDefinition describes a search provider for global search.
type SearchDefinition struct {
	ID           string              `yaml:"id"             json:"id"`
	Domain       string              `yaml:"domain"         json:"domain"`
	Capabilities []string            `yaml:"capabilities"   json:"capabilities"`
	Operation    OperationBinding    `yaml:"operation"      json:"operation"`
	ResultMapping SearchResultMapping `yaml:"result_mapping" json:"result_mapping"`
	Weight       int                 `yaml:"weight"         json:"weight,omitempty"`
	MaxResults   int                 `yaml:"max_results"    json:"max_results,omitempty"`
}

// SearchResultMapping describes how to map backend search results to UI results.
type SearchResultMapping struct {
	ItemsPath     string `yaml:"items_path"     json:"items_path"`
	TitleField    string `yaml:"title_field"    json:"title_field"`
	SubtitleField string `yaml:"subtitle_field" json:"subtitle_field,omitempty"`
	CategoryField string `yaml:"category_field" json:"category_field,omitempty"`
	IconField     string `yaml:"icon_field"     json:"icon_field,omitempty"`
	Route         string `yaml:"route"          json:"route"`
	IDField       string `yaml:"id_field"       json:"id_field"`
}

// LookupDefinition describes a lookup provider for dropdowns and reference fields.
type LookupDefinition struct {
	ID          string           `yaml:"id"           json:"id"`
	Operation   OperationBinding `yaml:"operation"    json:"operation"`
	LabelField  string           `yaml:"label_field"  json:"label_field"`
	ValueField  string           `yaml:"value_field"  json:"value_field"`
	SearchField string           `yaml:"search_field" json:"search_field,omitempty"`
	Cache       *CacheConfig     `yaml:"cache"        json:"cache,omitempty"`
}

// CacheConfig describes caching settings for a lookup.
type CacheConfig struct {
	TTL   string `yaml:"ttl"   json:"ttl"`
	Scope string `yaml:"scope" json:"scope"`
}
