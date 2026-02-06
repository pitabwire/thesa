# 23 — Glossary and Appendices

This document serves as the reference glossary for all Thesa terminology, provides
JSON Schema references for automated validation, collects frequently asked questions,
and indexes all decision records.

---

## Glossary

### Core Concepts

| Term | Definition | First Introduced |
|------|-----------|-----------------|
| **Thesa** | The Backend-For-Frontend (BFF) system described by this architecture. Serves as the sole entry point between frontends and backend services. | [00 — Overview](00-overview.md) |
| **BFF (Backend-For-Frontend)** | An architectural pattern where a dedicated backend layer sits between the frontend and domain services. The BFF tailors responses specifically for the frontend's needs. | [00 — Overview](00-overview.md) |
| **Server-Driven UI** | A pattern where the server sends descriptors that describe what the UI should render, rather than the frontend deciding its own layout and behavior. The frontend is a generic renderer. | [01 — Principles](01-principles-and-invariants.md) (P1) |
| **Definition** | A YAML configuration file that declares how backend operations are exposed to the UI. Definitions are authored by domain teams and describe pages, forms, commands, workflows, and searches. Definitions are internal — never sent to the frontend. | [05 — Definitions](05-ui-exposure-definitions.md) |
| **Descriptor** | A resolved, capability-filtered data structure derived from a definition. Descriptors are the JSON payloads sent to the frontend. They contain only what the current user is allowed to see and interact with. | [08 — Descriptor Model](08-ui-descriptor-model.md) |
| **Domain** | A bounded context representing a business area (e.g., Orders, Inventory, Customers). Each domain has its own definition files, capability namespace, and backend services. | [02 — Topology](02-system-topology.md) |
| **Domain Definition** | The top-level YAML structure that groups all pages, forms, commands, workflows, searches, and lookups for a single domain. | [06 — Schema Reference](06-definition-schema-reference.md) |

### Identity and Tenancy

| Term | Definition | First Introduced |
|------|-----------|-----------------|
| **Tenant** | The top-level organizational boundary for data isolation. A tenant represents a company, organization, or account. All data is scoped to a tenant. Tenant ID is extracted exclusively from the verified JWT, never from request input. | [02 — Topology](02-system-topology.md) |
| **Partition** | A workspace, environment, or subdivision within a tenant (e.g., "us-production", "eu-staging"). Enables multi-region or multi-environment support within a single tenant. | [02 — Topology](02-system-topology.md) |
| **Subject** | The authenticated user making a request. Identified by a Subject ID extracted from the JWT `sub` claim. | [04 — Identity](04-request-context-and-identity.md) |
| **RequestContext** | A value object constructed from the verified JWT and request headers. Contains subject ID, tenant ID, partition ID, roles, claims, session ID, correlation ID, locale, timezone, and tracing identifiers. Immutable after construction. Passed through the entire request lifecycle. | [04 — Identity](04-request-context-and-identity.md) |
| **Claim** | A key-value pair from the JWT token containing identity assertions (e.g., email, roles, tenant ID). Claims are extracted via configurable claim paths. | [04 — Identity](04-request-context-and-identity.md) |

### Authorization

| Term | Definition | First Introduced |
|------|-----------|-----------------|
| **Capability** | A UI-level permission string in the format `namespace:resource:action` (e.g., `orders:list:view`, `orders:cancel:execute`). Capabilities determine what elements appear in descriptors and what commands a user can execute. | [07 — Capabilities](07-capabilities-and-permissions.md) |
| **CapabilitySet** | A map of capability strings to boolean values, representing all permissions for a given request context. Type alias: `map[string]bool`. | [07 — Capabilities](07-capabilities-and-permissions.md) |
| **CapabilityResolver** | The component that resolves a RequestContext into a CapabilitySet. Uses caching to avoid repeated resolution for the same subject/tenant/partition. | [07 — Capabilities](07-capabilities-and-permissions.md) |
| **PolicyEvaluator** | The backend implementation that actually computes capabilities from roles, attributes, and tenant configuration. Implementations include static (config file), OPA (external engine), and database-backed. | [07 — Capabilities](07-capabilities-and-permissions.md) |
| **Capability Namespace** | The hierarchical prefix of a capability string, typically matching the domain name. Each domain owns its namespace (e.g., `orders:*`). Wildcards enable hierarchical grants. | [07 — Capabilities](07-capabilities-and-permissions.md) |

### UI Components

| Term | Definition | First Introduced |
|------|-----------|-----------------|
| **Page** | A screen in the frontend application. Can display tables (list pages), detail views, or composite layouts. Defined by a PageDefinition and resolved to a PageDescriptor. | [06 — Schema Reference](06-definition-schema-reference.md) |
| **Form** | A data entry screen with sections and fields. Used for creating or editing resources. Defined by a FormDefinition and resolved to a FormDescriptor. Can be pre-populated with existing data. | [06 — Schema Reference](06-definition-schema-reference.md) |
| **Table** | A component within a page that displays paginated, sortable, filterable rows of data. Contains columns, filters, row actions, and bulk actions. | [06 — Schema Reference](06-definition-schema-reference.md) |
| **Column** | A single data field displayed in a table. Has a type (text, number, currency, badge, datetime, etc.) that tells the frontend how to render it. | [06 — Schema Reference](06-definition-schema-reference.md) |
| **Filter** | A search/filter control on a table page. Types include text, select, multi-select, date range, number range, and boolean. | [06 — Schema Reference](06-definition-schema-reference.md) |
| **Section** | A grouping of fields within a page or form. Provides visual organization with optional titles and layout hints. | [06 — Schema Reference](06-definition-schema-reference.md) |
| **Field** | A single data element within a section. On detail pages, fields are read-only displays. On forms, fields are input controls. Types include text, number, select, textarea, date, datetime, checkbox, toggle, file, rich text, hidden, static, and custom. | [06 — Schema Reference](06-definition-schema-reference.md) |
| **Action** | An interactive element (button, link) that the user can trigger. Types include command (invokes a backend operation), navigate (routes to a new page), workflow (starts or advances a workflow), link (opens external URL), and modal (opens a form in a dialog). | [06 — Schema Reference](06-definition-schema-reference.md) |
| **Badge** | A small visual indicator on a navigation item or table cell. Shows a count, status, or label with a color style (info, warning, danger, success). | [08 — Descriptor Model](08-ui-descriptor-model.md) |
| **Breadcrumb** | A navigation trail showing the user's position in the page hierarchy. Assembled from the page definition and its parent domain. | [08 — Descriptor Model](08-ui-descriptor-model.md) |
| **NavigationTree** | The hierarchical menu structure sent to the frontend. Built from all domain definitions, filtered by capabilities, sorted by order. | [08 — Descriptor Model](08-ui-descriptor-model.md) |

### Commands and Workflows

| Term | Definition | First Introduced |
|------|-----------|-----------------|
| **Command** | A named, authorized mutation operation invoked via `POST /ui/commands/{commandId}`. Commands go through a full pipeline: resolve → authorize → map → validate → invoke → translate. | [10 — Commands](10-command-and-action-model.md) |
| **CommandInput** | The request body for a command execution. Contains `input` (user-provided payload), `route_params` (current route parameters), and `idempotency_key` (optional deduplication key). | [10 — Commands](10-command-and-action-model.md) |
| **CommandResponse** | The response from a command execution. Contains `success` (boolean), `message`, `result` (projected response data), and `errors` (field-level errors on failure). | [10 — Commands](10-command-and-action-model.md) |
| **Idempotency Key** | A unique string provided by the client to ensure that duplicate command submissions produce the same result without re-executing the backend operation. Keys have a configurable TTL. | [10 — Commands](10-command-and-action-model.md) |
| **Workflow** | A multi-step, stateful process managed by the BFF. Workflows are definition-driven: the state machine is declared in YAML, and the engine executes it. Examples: approval processes, onboarding flows, dispute resolution. | [11 — Workflow Engine](11-workflow-engine.md) |
| **WorkflowInstance** | A runtime instance of a workflow. Contains the current step, accumulated state, status (active/completed/failed/cancelled), and tenant isolation fields. Persisted in the WorkflowStore. | [11 — Workflow Engine](11-workflow-engine.md) |
| **WorkflowEvent** | An append-only audit record of something that happened in a workflow. Events include step_entered, step_completed, step_failed, transition, timeout, and workflow_completed. | [11 — Workflow Engine](11-workflow-engine.md) |
| **Step** | A single stage within a workflow. Types: action (requires user input), approval (approve/reject semantics), system (automatic backend call), wait (pause for duration/event), notification (fire-and-forget), terminal (end state). | [11 — Workflow Engine](11-workflow-engine.md) |
| **Transition** | A directional link between two steps, triggered by an event. Transitions can have conditions evaluated against workflow state. | [11 — Workflow Engine](11-workflow-engine.md) |

### Backend Invocation

| Term | Definition | First Introduced |
|------|-----------|-----------------|
| **Operation Binding** | A reference to a backend operation. Contains the invocation type (openapi or sdk), service ID, and operation ID (or handler name). Links a definition element to a concrete backend call. | [06 — Schema Reference](06-definition-schema-reference.md) |
| **OperationInvoker** | The interface for invoking backend operations. Has two implementations: OpenAPIOperationInvoker (dynamic HTTP) and SDKOperationInvoker (typed clients). | [13 — API Mapping](13-api-mapping-and-invocation.md) |
| **InvokerRegistry** | Holds all OperationInvoker implementations and dispatches based on the binding's type field. | [18 — Interfaces](18-core-abstractions-and-interfaces.md) |
| **OpenAPIIndex** | An in-memory index of all loaded OpenAPI specifications. Provides operation lookup by (serviceId, operationId), request schema validation, and parameter metadata. | [13 — API Mapping](13-api-mapping-and-invocation.md) |
| **InvocationInput** | The data needed to invoke a backend operation: path parameters, query parameters, headers, and body. | [18 — Interfaces](18-core-abstractions-and-interfaces.md) |
| **InvocationResult** | The result of a backend invocation: HTTP status code, parsed response body, and response headers. | [18 — Interfaces](18-core-abstractions-and-interfaces.md) |
| **SDK Handler** | A typed Go function that handles a specific backend operation using a pre-compiled client (gRPC, Connect, custom). Registered by name in the SDKInvokerRegistry. | [13 — API Mapping](13-api-mapping-and-invocation.md) |

### Mapping and Transformation

| Term | Definition | First Introduced |
|------|-----------|-----------------|
| **InputMapping** | Rules that transform UI input into backend request parameters. Specifies how to resolve path params, query params, headers, and request body from the command input, route params, and request context. | [10 — Commands](10-command-and-action-model.md) |
| **OutputMapping** | Rules that transform backend response data into the UI-safe shape returned to the frontend. Used in commands to project response fields. | [10 — Commands](10-command-and-action-model.md) |
| **ResponseMapping** | Rules that extract and rename fields from backend list/detail responses. Specifies items_path, total_path, and field_map for data source responses. Used in page data and form data fetching. | [13 — API Mapping](13-api-mapping-and-invocation.md) |
| **Field Map** | A dictionary mapping UI field names to backend field names (e.g., `order_number: "orderNumber"`). Enables the adapter layer to absorb backend naming changes without frontend impact. | [14 — Contract Stability](14-schema-and-contract-stability.md) |
| **Body Mapping Strategy** | The method used to construct the backend request body from command input. Three strategies: passthrough (send as-is), template (merge into a template), projection (map individual fields). | [10 — Commands](10-command-and-action-model.md) |
| **Source Expression** | A restricted dot-path reference used in input mappings. Prefixes: `input.` (from user payload), `route.` (from route params), `context.` (from RequestContext), `workflow.` (from workflow state). Not a general-purpose expression language. | [10 — Commands](10-command-and-action-model.md) |

### Infrastructure

| Term | Definition | First Introduced |
|------|-----------|-----------------|
| **Circuit Breaker** | A per-service resilience pattern that prevents cascading failures. States: Closed (healthy, passing requests), Open (failing fast, not calling backend), Half-Open (testing with a single probe request). | [16 — Observability](16-observability-and-reliability.md) |
| **Error Envelope** | The standard JSON structure for all error responses. Contains `code` (machine-readable), `message` (human-readable), `details` (field-level errors), and `trace_id` (for debugging). | [15 — Error Handling](15-error-handling-and-validation.md) |
| **FieldError** | A field-level validation error within the error envelope's details array. Contains `field` (UI field name), `code` (error type), and `message` (human-readable). | [15 — Error Handling](15-error-handling-and-validation.md) |
| **Error Map** | A per-command configuration that translates backend error codes to user-friendly messages. Defined in the command definition. Unmapped codes receive a generic message. | [15 — Error Handling](15-error-handling-and-validation.md) |
| **Adapter Layer** | The conceptual layer formed by definitions (field maps, input/output mappings, response mappings) that insulates the frontend contract from backend changes. When a backend evolves, only the definition changes — not the frontend. | [14 — Contract Stability](14-schema-and-contract-stability.md) |
| **Hot Reload** | The ability to update definition files at runtime without restarting the BFF process. Uses filesystem watching with debouncing. The definition registry is atomically replaced. | [05 — Definitions](05-ui-exposure-definitions.md) |
| **Atomic Pointer Swap** | The concurrency mechanism used by the DefinitionRegistry. A new immutable snapshot is built and stored via `atomic.Pointer.Store()`. In-flight reads on the old snapshot complete safely. No locks needed for reads. | [18 — Interfaces](18-core-abstractions-and-interfaces.md) |

### Observability

| Term | Definition | First Introduced |
|------|-----------|-----------------|
| **Correlation ID** | A unique identifier that ties together all operations for a single frontend user action. Propagated from the frontend via `X-Correlation-Id` header or generated if missing. | [04 — Identity](04-request-context-and-identity.md) |
| **Trace ID** | An OpenTelemetry distributed trace identifier. Propagated to backend services via the `traceparent` header. Enables end-to-end tracing across the frontend, BFF, and backends. | [16 — Observability](16-observability-and-reliability.md) |
| **Span** | A unit of work within a distributed trace. The BFF creates spans for HTTP handling, capability resolution, input mapping, backend invocation, and output mapping. | [16 — Observability](16-observability-and-reliability.md) |

---

## Appendix A: Definition File JSON Schema

This appendix defines the JSON Schema for validating definition YAML files.
Domain teams should use this schema in their CI pipelines to validate
definitions before deployment.

### Schema: DomainDefinition

```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "title": "Thesa Domain Definition",
  "description": "Root schema for a Thesa UI exposure definition file",
  "type": "object",
  "required": ["domain"],
  "properties": {
    "domain": {
      "type": "object",
      "required": ["id", "name"],
      "properties": {
        "id": {
          "type": "string",
          "pattern": "^[a-z][a-z0-9_]*$",
          "description": "Unique domain identifier (lowercase, alphanumeric, underscores)"
        },
        "name": {
          "type": "string",
          "minLength": 1,
          "description": "Human-readable domain name"
        },
        "description": {
          "type": "string"
        },
        "icon": {
          "type": "string",
          "description": "Material icon name for navigation"
        },
        "order": {
          "type": "integer",
          "minimum": 0,
          "description": "Sort order in navigation menu"
        },
        "capability": {
          "type": "string",
          "pattern": "^[a-z][a-z0-9_]*:[a-z][a-z0-9_]*(:[a-z][a-z0-9_]*)?$",
          "description": "Required capability to see this domain in navigation"
        },
        "navigation": { "$ref": "#/$defs/NavigationDefinition" },
        "pages": {
          "type": "array",
          "items": { "$ref": "#/$defs/PageDefinition" }
        },
        "forms": {
          "type": "array",
          "items": { "$ref": "#/$defs/FormDefinition" }
        },
        "commands": {
          "type": "array",
          "items": { "$ref": "#/$defs/CommandDefinition" }
        },
        "workflows": {
          "type": "array",
          "items": { "$ref": "#/$defs/WorkflowDefinition" }
        },
        "searches": {
          "type": "array",
          "items": { "$ref": "#/$defs/SearchDefinition" }
        },
        "lookups": {
          "type": "array",
          "items": { "$ref": "#/$defs/LookupDefinition" }
        }
      }
    }
  },
  "$defs": {
    "CapabilityString": {
      "type": "string",
      "pattern": "^[a-z][a-z0-9_]*(:[a-z*][a-z0-9_]*)*(:[a-z*][a-z0-9_]*)?$",
      "description": "Capability in format namespace:resource:action with optional wildcards"
    },
    "OperationBinding": {
      "type": "object",
      "required": ["type"],
      "properties": {
        "type": {
          "type": "string",
          "enum": ["openapi", "sdk"]
        },
        "service_id": {
          "type": "string",
          "description": "Required for type: openapi"
        },
        "operation_id": {
          "type": "string",
          "description": "Required for type: openapi"
        },
        "handler": {
          "type": "string",
          "description": "Required for type: sdk"
        }
      },
      "if": { "properties": { "type": { "const": "openapi" } } },
      "then": { "required": ["service_id", "operation_id"] },
      "else": {
        "if": { "properties": { "type": { "const": "sdk" } } },
        "then": { "required": ["handler"] }
      }
    },
    "NavigationDefinition": {
      "type": "object",
      "properties": {
        "items": {
          "type": "array",
          "items": {
            "type": "object",
            "required": ["id", "label", "route"],
            "properties": {
              "id": { "type": "string" },
              "label": { "type": "string" },
              "icon": { "type": "string" },
              "route": { "type": "string" },
              "order": { "type": "integer" },
              "capability": { "$ref": "#/$defs/CapabilityString" },
              "badge": { "$ref": "#/$defs/OperationBinding" }
            }
          }
        }
      }
    },
    "PageDefinition": {
      "type": "object",
      "required": ["id", "title", "layout"],
      "properties": {
        "id": {
          "type": "string",
          "pattern": "^[a-z][a-z0-9_]*(\\.[a-z][a-z0-9_]*)*$"
        },
        "title": { "type": "string", "minLength": 1 },
        "layout": {
          "type": "string",
          "enum": ["table", "detail", "composite", "dashboard"]
        },
        "capability": { "$ref": "#/$defs/CapabilityString" },
        "table": { "$ref": "#/$defs/TableDefinition" },
        "sections": {
          "type": "array",
          "items": { "$ref": "#/$defs/SectionDefinition" }
        },
        "actions": {
          "type": "array",
          "items": { "$ref": "#/$defs/ActionDefinition" }
        },
        "data_source": { "$ref": "#/$defs/DataSourceDefinition" }
      }
    },
    "TableDefinition": {
      "type": "object",
      "required": ["columns", "data_source"],
      "properties": {
        "columns": {
          "type": "array",
          "minItems": 1,
          "items": { "$ref": "#/$defs/ColumnDefinition" }
        },
        "filters": {
          "type": "array",
          "items": { "$ref": "#/$defs/FilterDefinition" }
        },
        "row_actions": {
          "type": "array",
          "items": { "$ref": "#/$defs/ActionDefinition" }
        },
        "bulk_actions": {
          "type": "array",
          "items": { "$ref": "#/$defs/ActionDefinition" }
        },
        "data_source": { "$ref": "#/$defs/DataSourceDefinition" },
        "page_size": { "type": "integer", "minimum": 1, "maximum": 200 },
        "default_sort": { "type": "string" },
        "default_sort_dir": { "type": "string", "enum": ["asc", "desc"] }
      }
    },
    "ColumnDefinition": {
      "type": "object",
      "required": ["id", "label", "type"],
      "properties": {
        "id": { "type": "string" },
        "label": { "type": "string" },
        "type": {
          "type": "string",
          "enum": ["text", "number", "currency", "percentage", "badge",
                   "datetime", "date", "boolean", "link", "image", "custom"]
        },
        "sortable": { "type": "boolean", "default": false },
        "width": { "type": "string" },
        "capability": { "$ref": "#/$defs/CapabilityString" },
        "source": { "type": "string", "description": "Backend field path (dot notation)" }
      }
    },
    "FilterDefinition": {
      "type": "object",
      "required": ["id", "label", "type"],
      "properties": {
        "id": { "type": "string" },
        "label": { "type": "string" },
        "type": {
          "type": "string",
          "enum": ["text", "select", "multi_select", "date_range",
                   "number_range", "boolean", "lookup"]
        },
        "options": {
          "type": "array",
          "items": {
            "type": "object",
            "required": ["value", "label"],
            "properties": {
              "value": { "type": "string" },
              "label": { "type": "string" }
            }
          }
        },
        "lookup_id": { "type": "string" },
        "operators": {
          "type": "array",
          "items": {
            "type": "string",
            "enum": ["eq", "neq", "gt", "gte", "lt", "lte", "contains",
                     "starts_with", "in", "between"]
          }
        }
      }
    },
    "SectionDefinition": {
      "type": "object",
      "required": ["fields"],
      "properties": {
        "id": { "type": "string" },
        "title": { "type": "string" },
        "capability": { "$ref": "#/$defs/CapabilityString" },
        "collapsible": { "type": "boolean" },
        "columns": { "type": "integer", "minimum": 1, "maximum": 4 },
        "fields": {
          "type": "array",
          "minItems": 1,
          "items": { "$ref": "#/$defs/FieldDefinition" }
        }
      }
    },
    "FieldDefinition": {
      "type": "object",
      "required": ["id", "label", "type"],
      "properties": {
        "id": { "type": "string" },
        "label": { "type": "string" },
        "type": {
          "type": "string",
          "enum": ["text", "number", "currency", "select", "multi_select",
                   "textarea", "date", "datetime", "checkbox", "toggle",
                   "file", "rich_text", "hidden", "static", "custom"]
        },
        "required": { "type": "boolean" },
        "read_only": { "type": "boolean" },
        "read_only_capability": { "$ref": "#/$defs/CapabilityString" },
        "capability": { "$ref": "#/$defs/CapabilityString" },
        "placeholder": { "type": "string" },
        "help_text": { "type": "string" },
        "default_value": {},
        "source": { "type": "string" },
        "options": {
          "type": "array",
          "items": {
            "type": "object",
            "required": ["value", "label"],
            "properties": {
              "value": { "type": "string" },
              "label": { "type": "string" }
            }
          }
        },
        "lookup_id": { "type": "string" },
        "validation": {
          "type": "object",
          "properties": {
            "min_length": { "type": "integer" },
            "max_length": { "type": "integer" },
            "min": { "type": "number" },
            "max": { "type": "number" },
            "pattern": { "type": "string" },
            "pattern_message": { "type": "string" }
          }
        },
        "conditions": {
          "type": "array",
          "items": { "$ref": "#/$defs/ConditionDefinition" }
        }
      }
    },
    "ActionDefinition": {
      "type": "object",
      "required": ["id", "label", "type"],
      "properties": {
        "id": { "type": "string" },
        "label": { "type": "string" },
        "icon": { "type": "string" },
        "style": {
          "type": "string",
          "enum": ["primary", "secondary", "danger", "warning", "ghost"]
        },
        "type": {
          "type": "string",
          "enum": ["command", "navigate", "workflow", "link", "modal"]
        },
        "capability": { "$ref": "#/$defs/CapabilityString" },
        "command_id": { "type": "string" },
        "navigate_to": { "type": "string" },
        "workflow_id": { "type": "string" },
        "href": { "type": "string" },
        "form_id": { "type": "string" },
        "confirmation": {
          "type": "object",
          "required": ["title", "message"],
          "properties": {
            "title": { "type": "string" },
            "message": { "type": "string" },
            "confirm_label": { "type": "string" },
            "cancel_label": { "type": "string" },
            "style": { "type": "string", "enum": ["info", "warning", "danger"] }
          }
        },
        "conditions": {
          "type": "array",
          "items": { "$ref": "#/$defs/ConditionDefinition" }
        }
      }
    },
    "CommandDefinition": {
      "type": "object",
      "required": ["id", "operation"],
      "properties": {
        "id": {
          "type": "string",
          "pattern": "^[a-z][a-z0-9_]*(\\.[a-z][a-z0-9_]*)*$"
        },
        "capability": { "$ref": "#/$defs/CapabilityString" },
        "operation": { "$ref": "#/$defs/OperationBinding" },
        "input_mapping": {
          "type": "object",
          "properties": {
            "path_params": {
              "type": "object",
              "additionalProperties": { "type": "string" }
            },
            "query_params": {
              "type": "object",
              "additionalProperties": { "type": "string" }
            },
            "headers": {
              "type": "object",
              "additionalProperties": { "type": "string" }
            },
            "body_mapping": {
              "type": "string",
              "enum": ["passthrough", "template", "projection"]
            },
            "body_template": { "type": "object" },
            "field_projection": {
              "type": "object",
              "additionalProperties": { "type": "string" }
            }
          }
        },
        "output_mapping": {
          "type": "object",
          "properties": {
            "field_projection": {
              "type": "object",
              "additionalProperties": { "type": "string" }
            },
            "success_message": { "type": "string" }
          }
        },
        "error_map": {
          "type": "object",
          "additionalProperties": { "type": "string" }
        },
        "idempotency": {
          "type": "object",
          "properties": {
            "enabled": { "type": "boolean" },
            "ttl": { "type": "string" },
            "key_source": {
              "type": "string",
              "enum": ["client", "generated", "input_hash"]
            }
          }
        },
        "rate_limit": {
          "type": "object",
          "properties": {
            "requests_per_minute": { "type": "integer" },
            "scope": {
              "type": "string",
              "enum": ["subject", "tenant", "global"]
            }
          }
        }
      }
    },
    "WorkflowDefinition": {
      "type": "object",
      "required": ["id", "name", "initial_step", "steps"],
      "properties": {
        "id": {
          "type": "string",
          "pattern": "^[a-z][a-z0-9_]*(\\.[a-z][a-z0-9_]*)*$"
        },
        "name": { "type": "string" },
        "capability": { "$ref": "#/$defs/CapabilityString" },
        "initial_step": { "type": "string" },
        "timeout": { "type": "string" },
        "steps": {
          "type": "array",
          "minItems": 1,
          "items": { "$ref": "#/$defs/StepDefinition" }
        }
      }
    },
    "StepDefinition": {
      "type": "object",
      "required": ["id", "name", "type"],
      "properties": {
        "id": { "type": "string" },
        "name": { "type": "string" },
        "type": {
          "type": "string",
          "enum": ["action", "approval", "system", "wait", "notification", "terminal"]
        },
        "capability": { "$ref": "#/$defs/CapabilityString" },
        "form_id": { "type": "string" },
        "operation": { "$ref": "#/$defs/OperationBinding" },
        "input_mapping": { "type": "object" },
        "output_mapping": { "type": "object" },
        "timeout": { "type": "string" },
        "transitions": {
          "type": "array",
          "items": {
            "type": "object",
            "required": ["event", "target"],
            "properties": {
              "event": { "type": "string" },
              "target": { "type": "string" },
              "condition": { "type": "string" }
            }
          }
        }
      }
    },
    "SearchDefinition": {
      "type": "object",
      "required": ["id", "operation", "result_mapping"],
      "properties": {
        "id": {
          "type": "string",
          "pattern": "^[a-z][a-z0-9_]*(\\.[a-z][a-z0-9_]*)*$"
        },
        "capability": { "$ref": "#/$defs/CapabilityString" },
        "operation": { "$ref": "#/$defs/OperationBinding" },
        "weight": { "type": "number", "minimum": 0, "maximum": 10 },
        "result_mapping": {
          "type": "object",
          "required": ["items_path", "title_field", "route_template"],
          "properties": {
            "items_path": { "type": "string" },
            "title_field": { "type": "string" },
            "subtitle_field": { "type": "string" },
            "badge_field": { "type": "string" },
            "route_template": { "type": "string" },
            "icon": { "type": "string" }
          }
        }
      }
    },
    "LookupDefinition": {
      "type": "object",
      "required": ["id"],
      "properties": {
        "id": {
          "type": "string",
          "pattern": "^[a-z][a-z0-9_]*(\\.[a-z][a-z0-9_]*)*$"
        },
        "type": {
          "type": "string",
          "enum": ["static", "dynamic"]
        },
        "options": {
          "type": "array",
          "items": {
            "type": "object",
            "required": ["value", "label"],
            "properties": {
              "value": { "type": "string" },
              "label": { "type": "string" }
            }
          }
        },
        "operation": { "$ref": "#/$defs/OperationBinding" },
        "mapping": {
          "type": "object",
          "properties": {
            "items_path": { "type": "string" },
            "value_field": { "type": "string" },
            "label_field": { "type": "string" }
          }
        },
        "cache": {
          "type": "object",
          "properties": {
            "ttl": { "type": "string" },
            "scope": {
              "type": "string",
              "enum": ["global", "tenant", "partition"]
            }
          }
        }
      }
    },
    "DataSourceDefinition": {
      "type": "object",
      "required": ["operation_id", "service_id"],
      "properties": {
        "operation_id": { "type": "string" },
        "service_id": { "type": "string" },
        "mapping": {
          "type": "object",
          "properties": {
            "items_path": { "type": "string" },
            "total_path": { "type": "string" },
            "field_map": {
              "type": "object",
              "additionalProperties": { "type": "string" }
            }
          }
        }
      }
    },
    "ConditionDefinition": {
      "type": "object",
      "required": ["field", "operator", "value"],
      "properties": {
        "field": { "type": "string" },
        "operator": {
          "type": "string",
          "enum": ["eq", "neq", "gt", "gte", "lt", "lte", "in", "not_in",
                   "contains", "is_empty", "is_not_empty"]
        },
        "value": {},
        "action": {
          "type": "string",
          "enum": ["show", "hide", "enable", "disable", "require"]
        }
      }
    }
  }
}
```

### Using the Schema

**In CI (with ajv-cli or similar):**

```bash
# Validate a definition file
ajv validate -s thesa-definition-schema.json -d definitions/orders/definition.yaml

# Validate all definition files
for f in definitions/*/definition.yaml; do
  ajv validate -s thesa-definition-schema.json -d "$f" || exit 1
done
```

**In Go (at startup):**

```
The BFF performs this validation automatically during startup (step 4 of the
startup sequence). The JSON Schema is used programmatically alongside additional
cross-reference validations that cannot be expressed in JSON Schema alone:

- operation_id references exist in the OpenAPI index
- form_id references exist in the domain's forms
- command_id references exist in the domain's commands
- workflow_id references exist in the domain's workflows
- lookup_id references exist in the domain's lookups
- capability namespaces match the domain ID
- transition targets reference valid step IDs within the same workflow
```

---

## Appendix B: Capability String Format

### Format

```
<namespace>:<resource>:<action>

Where:
  namespace = domain ID (e.g., "orders", "inventory", "customers")
  resource  = logical resource (e.g., "list", "detail", "create", "cancel")
  action    = operation type (e.g., "view", "edit", "execute")
```

### Two-Part Capabilities

For domain-level capabilities, the format can be two parts:

```
<namespace>:<action>

Examples:
  orders:nav:view          # 3-part: can see orders in navigation
  orders:search:execute    # 3-part: orders appear in global search
```

### Wildcards

```
orders:*                  # All capabilities in the orders namespace
orders:list:*             # All actions on the orders list resource
*:*:view                  # View action on all resources in all namespaces (use cautiously)
```

Wildcard resolution is performed by the CapabilityResolver. A capability set
containing `orders:*` will match any check for `orders:list:view`,
`orders:detail:edit`, etc.

### Naming Conventions

| Convention | Example | Description |
|-----------|---------|-------------|
| Navigation visibility | `{domain}:nav:view` | Can see domain in sidebar |
| Page visibility | `{domain}:{page}:view` | Can see the page |
| Form visibility | `{domain}:{form}:view` | Can see the form |
| Edit capability | `{domain}:{resource}:edit` | Can see edit actions/fields |
| Command execution | `{domain}:{command}:execute` | Can execute the command |
| Workflow start | `{domain}:{workflow}:start` | Can start the workflow |
| Workflow step | `{domain}:{step}:execute` | Can advance the step |
| Search inclusion | `{domain}:search:execute` | Results appear in global search |

---

## Appendix C: HTTP Error Code Reference

### Client Errors

| Status | Code | When | Response Body |
|--------|------|------|---------------|
| 400 | `BAD_REQUEST` | Malformed JSON, missing required envelope fields | `{ "error": { "code": "BAD_REQUEST", "message": "..." } }` |
| 401 | `UNAUTHORIZED` | Missing, expired, or invalid JWT token | `{ "error": { "code": "UNAUTHORIZED", "message": "..." } }` |
| 403 | `FORBIDDEN` | Valid token but insufficient capabilities | `{ "error": { "code": "FORBIDDEN", "message": "..." } }` |
| 404 | `NOT_FOUND` | Page, command, workflow, resource, or wrong-tenant resource | `{ "error": { "code": "NOT_FOUND", "message": "..." } }` |
| 409 | `CONFLICT` | Idempotency key conflict, optimistic locking failure | `{ "error": { "code": "CONFLICT", "message": "..." } }` |
| 422 | `VALIDATION_ERROR` | Input validation failed (includes field-level details) | `{ "error": { "code": "VALIDATION_ERROR", "details": [...] } }` |
| 422 | `INVALID_TRANSITION` | Workflow event not valid for current step | `{ "error": { "code": "INVALID_TRANSITION", "message": "..." } }` |
| 429 | `RATE_LIMITED` | Rate limit exceeded | `{ "error": { "code": "RATE_LIMITED", "message": "...", "retry_after": 30 } }` |

### Server Errors

| Status | Code | When | Response Body |
|--------|------|------|---------------|
| 500 | `INTERNAL_ERROR` | Unexpected server error | `{ "error": { "code": "INTERNAL_ERROR", "message": "An unexpected error occurred" } }` |
| 502 | `BACKEND_UNAVAILABLE` | Backend service unreachable or circuit breaker open | `{ "error": { "code": "BACKEND_UNAVAILABLE", "message": "..." } }` |
| 504 | `BACKEND_TIMEOUT` | Backend service timed out | `{ "error": { "code": "BACKEND_TIMEOUT", "message": "..." } }` |

### Backend-Translated Errors

Commands may return backend-specific error codes that are translated via the
command's `error_map`. These use HTTP 422 with the translated error code:

```json
{
  "error": {
    "code": "ORDER_SHIPPED",
    "message": "Cannot cancel an order that has already shipped.",
    "trace_id": "trace-abc"
  }
}
```

---

## Appendix D: Configuration Quick Reference

| Feature | Config Key | Default | Notes |
|---------|-----------|---------|-------|
| Server port | `server.port` | 8080 | |
| Handler timeout | `server.handler_timeout` | 25s | Must be < client timeout |
| Shutdown timeout | `server.shutdown_timeout` | 30s | |
| Hot reload | `definitions.hot_reload` | false | true in development only |
| Capability cache TTL | `capability.cache.ttl` | 5m | |
| Backend call timeout | `services.*.timeout` | 10s | Per-service |
| Circuit breaker threshold | `services.*.circuit_breaker.failure_threshold` | 5 | Consecutive failures |
| Circuit breaker timeout | `services.*.circuit_breaker.timeout` | 30s | Time in open state |
| Retry max attempts | `services.*.retry.max_attempts` | 3 | Including first attempt |
| Idempotency TTL | `idempotency.store.default_ttl` | 24h | |
| Lookup cache TTL | `lookup.cache.ttl` | 300s | |
| Search timeout per provider | `search.timeout_per_provider` | 3s | |
| Log level | `observability.log_level` | info | debug, info, warn, error |
| Trace sampling rate | `observability.tracing.sampling_rate` | 0.1 | 0.0 to 1.0 |
| Workflow timeout check interval | `workflow.timeout_check_interval` | 60s | |

---

## Appendix E: Frequently Asked Questions

### Architecture

**Q: Why a BFF instead of direct API calls from the frontend?**

A: The BFF provides seven critical capabilities that direct API calls cannot:
1. Server-driven UI (descriptors tailored per user and capability).
2. Uniform authorization at the UI layer.
3. Backend abstraction (frontend never knows backend URLs or APIs).
4. Schema validation before backend invocation.
5. Contract stability (adapter layer absorbs backend changes).
6. Cross-cutting concerns (logging, tracing, metrics) in one place.
7. Multi-step workflows with server-managed state.

See [01 — Principles](01-principles-and-invariants.md) for the full rationale.

**Q: Can the frontend call backend services directly for performance-critical paths?**

A: No. Principle P2 (Single Choke Point) requires all frontend-to-backend
communication to go through the BFF. This is non-negotiable for security
(tenant isolation), observability (every call traced), and contract stability.
For latency-sensitive scenarios, use caching within the BFF or optimize the
backend service.

**Q: Can a definition reference operations from multiple backend services?**

A: Yes. A single domain definition can reference operations from different
services. For example, an orders domain might use `orders-svc` for CRUD
operations and `ledger-svc` for financial operations. The `service_id` in
each operation binding determines which service is called.

**Q: How do I add a new backend service?**

A: Three steps:
1. Add the OpenAPI spec to the `specs/` directory.
2. Add the service configuration in `config.yaml` (base_url, timeout, auth, pagination).
3. Reference the service's operations in definition files.
No code changes required (for OpenAPI-based services).

### Definitions

**Q: Who authors definition files?**

A: Domain teams. The team that owns the backend service (e.g., Orders team)
authors the definition file for their domain. They know their API, their
data model, and their UI requirements. The BFF/platform team provides the
schema, validation tools, and runtime. See [05 — Definitions](05-ui-exposure-definitions.md).

**Q: What happens if a definition references a non-existent operation ID?**

A: The BFF validates all operation_id references against the OpenAPI index at
startup (step 4). If any reference is invalid, the BFF refuses to start (exits
with non-zero status). This is a hard validation — no partial loading.

**Q: Can I have multiple definition files for the same domain?**

A: The current design uses one definition file per domain (`definitions/{domain}/definition.yaml`).
If a domain becomes very large, it can be split into multiple files within
the domain directory, which the loader assembles into a single DomainDefinition.

**Q: Can definitions be environment-specific?**

A: Definitions should be environment-agnostic. Environment-specific concerns
(backend URLs, timeouts, feature flags) belong in the configuration file.
If an entire page should be hidden in production, use a capability that is only
granted in non-production environments.

### Security

**Q: What if the frontend sends a fake tenant ID?**

A: The BFF ignores it. Tenant ID is extracted exclusively from the verified JWT
(Principle P3). Even if the frontend sends `X-Tenant-Id: evil-corp` as a
header, the middleware overrides it with the JWT's tenant ID. See
[17 — Security](17-security-model.md).

**Q: Are capabilities a security boundary?**

A: Capabilities are a UI-level authorization mechanism. They control what the
frontend can see and do. The actual security boundary is at the backend service,
which performs its own authorization. Capabilities are a "pre-filter" that
prevents the UI from showing actions the user cannot perform. See
[07 — Capabilities](07-capabilities-and-permissions.md), section on Capability-Backend Mapping.

**Q: How are secrets managed in definition files?**

A: They are not. Definition files NEVER contain secrets. Secrets (database
DSNs, API keys, client secrets) are managed through environment variables or
secret managers, injected at runtime. See [17 — Security](17-security-model.md),
Secret Management section.

### Operations

**Q: Can I deploy definition changes without restarting the BFF?**

A: Yes, if `definitions.hot_reload: true` is configured. The BFF watches for
file changes, revalidates, and atomically swaps the definition registry. However,
hot-reload is typically disabled in production for stability. Instead, deploy a
new container image with updated definitions. See [05 — Definitions](05-ui-exposure-definitions.md).

**Q: What happens during a rolling deployment?**

A: During rolling deployment, some pods run the old version and some run the new.
This is safe because:
- Each pod loads its own definitions (consistent within a pod).
- The frontend may see slightly different descriptors between requests if
  load-balanced to different pods, but this is transient and self-correcting.
- For critical changes, use the canary strategy described in [22 — Deployment](22-deployment-and-operations.md).

**Q: How do I debug a production issue?**

A: Use the trace_id from the error response:
1. Find the trace in your distributed tracing system (Jaeger, Tempo).
2. Follow the span tree: HTTP request → capability resolution → command execution → backend invocation.
3. Check structured logs filtered by the correlation_id.
4. Check Prometheus metrics for the relevant service.

---

## Document Index

| # | Title | Summary |
|---|-------|---------|
| [00](00-overview.md) | Overview | System purpose, data flows, design decisions, document map |
| [01](01-principles-and-invariants.md) | Principles and Invariants | 10 non-negotiable design principles (P1-P10) |
| [02](02-system-topology.md) | System Topology | Network layout, tenancy model, service categories |
| [03](03-transport-and-invocation.md) | Transport and Invocation | HTTP API, outbound invocation, middleware pipeline |
| [04](04-request-context-and-identity.md) | Request Context and Identity | JWT verification, context construction, authentication strategies |
| [05](05-ui-exposure-definitions.md) | UI Exposure Definitions | Definition system, loading, validation, hot-reload |
| [06](06-definition-schema-reference.md) | Definition Schema Reference | Complete YAML schema for all definition types |
| [07](07-capabilities-and-permissions.md) | Capabilities and Permissions | Authorization model, resolution, policy evaluation |
| [08](08-ui-descriptor-model.md) | UI Descriptor Model | All descriptor types, resolution process |
| [09](09-server-driven-ui-apis.md) | Server-Driven UI APIs | Complete endpoint specifications |
| [10](10-command-and-action-model.md) | Command and Action Model | Command execution pipeline, idempotency, error translation |
| [11](11-workflow-engine.md) | Workflow Engine | Lifecycle, state management, persistence |
| [12](12-global-search.md) | Global Search | Federated search, scoring, parallel execution |
| [13](13-api-mapping-and-invocation.md) | API Mapping and Invocation | OpenAPI index, dynamic invocation, pagination |
| [14](14-schema-and-contract-stability.md) | Schema and Contract Stability | Adapter layer, versioning, compatibility |
| [15](15-error-handling-and-validation.md) | Error Handling and Validation | Error envelope, validation layers |
| [16](16-observability-and-reliability.md) | Observability and Reliability | Logging, tracing, metrics, circuit breakers |
| [17](17-security-model.md) | Security Model | Threat model, security controls, audit logging |
| [18](18-core-abstractions-and-interfaces.md) | Core Abstractions and Interfaces | Complete interface catalog, dependency graph |
| [19](19-go-package-structure.md) | Go Package Structure | Directory layout, package responsibilities |
| [20](20-example-domain-orders.md) | Example Domain: Orders | Complete worked example with YAML definition |
| [21](21-example-end-to-end-flows.md) | Example End-to-End Flows | 12 detailed request flow walkthroughs |
| [22](22-deployment-and-operations.md) | Deployment and Operations | Build, deploy, configure, operate, scale |
| [23](23-glossary-and-appendices.md) | Glossary and Appendices | This document |
