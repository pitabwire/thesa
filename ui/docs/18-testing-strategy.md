# 18. Testing Strategy

## Why Testing Matters

Thesa UI is a **runtime for dynamic UIs**. Unlike a traditional app where every screen is handwritten and visually inspected, Thesa UI builds screens from data. This means:

- You cannot manually test every possible page (the BFF defines unlimited pages)
- A bug in the rendering engine affects ALL pages, not just one
- A regression in the form engine breaks ALL forms
- A caching bug could show stale data everywhere

Automated testing is the only way to ensure reliability at this scale.

---

## Testing Pyramid

The testing strategy follows the testing pyramid — more unit tests (fast, cheap), fewer integration tests (slower, more comprehensive), even fewer golden tests (visual verification):

```
         /\
        /  \     Golden Tests (visual regression)
       / 10 \
      /______\
     /        \  Widget Tests (component behavior)
    /    30    \
   /____________\
  /              \  Unit Tests (logic correctness)
 /       60       \
/____________________\
```

| Level | Count (approx.) | Speed | What It Verifies |
|---|---|---|---|
| **Unit tests** | ~60% of tests | Milliseconds each | Logic is correct: parsing, validation, permissions, caching |
| **Widget tests** | ~30% of tests | Seconds each | Components render correctly and respond to interactions |
| **Golden tests** | ~10% of tests | Seconds each | Visual appearance has not changed unexpectedly |

---

## Unit Tests

Unit tests verify individual functions and classes in isolation — no Flutter widgets, no UI rendering, no database, no network.

### What to Unit Test

#### 1. Schema Parsing and Resolution

Test that BFF JSON schemas are correctly parsed into Dart objects:

```
Test: "Parses a simple string field schema"
  Input:  { "name": "email", "type": "string", "required": true, "maxLength": 100 }
  Expect: FieldDefinition(name: "email", type: string, required: true, maxLength: 100)

Test: "Resolves $ref pointers"
  Input:  { "name": "address", "type": "object", "$ref": "schemas/address" }
  Setup:  Mock schemaProvider to return address schema
  Expect: FieldDefinition with nested address fields inlined

Test: "Detects circular references"
  Input:  Schema A references Schema B, Schema B references Schema A
  Expect: Throws CircularReferenceError

Test: "Merges allOf schemas"
  Input:  { "allOf": [{ fields: [a, b] }, { fields: [c, d] }] }
  Expect: Merged schema with fields [a, b, c, d]
```

#### 2. Form Validation Logic

Test that validation rules from schemas produce correct error messages:

```
Test: "Required field with empty value produces error"
  Input:  Field(required: true), value: ""
  Expect: ValidationError("This field is required")

Test: "String field with value below minLength produces error"
  Input:  Field(minLength: 3), value: "ab"
  Expect: ValidationError("Must be at least 3 characters")

Test: "Number field with value above max produces error"
  Input:  Field(type: number, max: 100), value: 101
  Expect: ValidationError("Must be 100 or less")

Test: "Pattern validation with non-matching value produces error"
  Input:  Field(pattern: "^[A-Z]{3}$"), value: "abc"
  Expect: ValidationError("Must match pattern: ^[A-Z]{3}$")

Test: "Valid value produces no error"
  Input:  Field(required: true, minLength: 3, maxLength: 100), value: "hello"
  Expect: null (no error)
```

#### 3. Permission Filtering

Test that components with `allowed: false` are correctly excluded:

```
Test: "Filters out components where allowed is false"
  Input:  [
    { type: "table", allowed: true },
    { type: "button", allowed: false },
    { type: "form", allowed: true }
  ]
  Expect: [table, form] (button is excluded)

Test: "Filters out navigation items where allowed is false"
  Input:  [
    { label: "Dashboard", allowed: true },
    { label: "Admin", allowed: false, children: [...] }
  ]
  Expect: [Dashboard] (Admin and all its children are excluded)

Test: "Filters out row actions where allowed is false"
  Input:  [
    { actionId: "view", allowed: true },
    { actionId: "delete", allowed: false }
  ]
  Expect: [view] (delete is excluded)
```

#### 4. Cache Coordinator Logic

Test the decision matrix for cache vs. network:

```
Test: "Returns fresh cached data without network call"
  Setup:  Cache has data fetched 5 minutes ago, TTL is 15 minutes
  Expect: Returns cached data, no network request made

Test: "Returns stale cached data AND triggers background refresh"
  Setup:  Cache has data fetched 20 minutes ago, TTL is 15 minutes, device is online
  Expect: Returns cached data immediately, AND background refresh is triggered

Test: "Returns stale cached data without refresh when offline"
  Setup:  Cache has stale data, device is offline
  Expect: Returns cached data, no network request attempted

Test: "Shows loading state when cache is empty and fetches from network"
  Setup:  Cache is empty, device is online
  Expect: Returns loading state, network request is made

Test: "ETag match results in cache entry update without payload change"
  Setup:  Network responds with 304 Not Modified
  Expect: Cache entry's fetched_at is updated, payload is unchanged, stale flag cleared
```

#### 5. Workflow State Machine

Test that step transitions follow the defined rules:

```
Test: "Advances to the next step on successful submission"
  Setup:  Current step = "select-order", transitions = { "select-order": ["enter-reason"] }
  Action: Submit step data
  Expect: Current step = "enter-reason"

Test: "Evaluates conditions for branching"
  Setup:  Current step = "enter-details"
          Transitions = { "enter-details": ["compliance-review", "standard-review"] }
          Conditions = { "compliance-review": amount > 50000 }
          Step data: amount = 75000
  Expect: Current step = "compliance-review"

Test: "Falls through to default when no conditions match"
  Setup:  Same as above but amount = 1000
  Expect: Current step = "standard-review"

Test: "Cannot transition to a step not in the transitions map"
  Setup:  Current step = "select-order", transitions = { "select-order": ["enter-reason"] }
  Action: Attempt to advance to "confirm"
  Expect: Throws InvalidTransitionError

Test: "Back navigation returns to previous step"
  Setup:  History = ["select-order", "enter-reason"], current = "review"
  Action: Go back
  Expect: Current step = "enter-reason", history = ["select-order"]
```

#### 6. Navigation Route Generation

```
Test: "Generates routes for allowed items"
  Input:  [
    { path: "/dashboard", pageId: "dash", allowed: true },
    { path: "/admin", pageId: "admin", allowed: false }
  ]
  Expect: One GoRoute for /dashboard, none for /admin

Test: "Generates nested routes for children"
  Input:  {
    path: "/orders", allowed: true,
    children: [
      { path: "/orders/list", pageId: "orders-list", allowed: true },
      { path: "/orders/returns", pageId: "returns", allowed: true }
    ]
  }
  Expect: Two GoRoutes: /orders/list and /orders/returns

Test: "Generates parameterized routes"
  Input:  { path: "/orders/:id", pageId: "order-detail", allowed: true, hidden: true }
  Expect: GoRoute with path parameter extraction for :id
```

---

## Widget Tests

Widget tests verify that Flutter widgets render correctly and respond to user interactions. They run in a test environment that simulates the Flutter framework but does not require a real device.

### What to Widget Test

#### 1. Dynamic Page Rendering

```
Test: "PageRenderer builds correct widget tree from descriptor"
  Setup:  Provide a PageDescriptor with 3 components: search_bar, filter_panel, data_table
  Expect: Find 1 SearchBar widget, 1 FilterPanel widget, 1 DynamicTable widget

Test: "PageRenderer handles empty component list"
  Setup:  Provide a PageDescriptor with 0 components
  Expect: Renders empty state widget

Test: "PageRenderer shows error card for unknown component type"
  Setup:  Provide a PageDescriptor with component type "unknown_widget"
  Expect: Find 1 UnknownComponentPlaceholder widget

Test: "PageRenderer uses plugin when registered"
  Setup:  Register a page plugin for pageId "custom-page"
  Expect: Plugin widget is rendered, not the generic PageRenderer
```

#### 2. Dynamic Form Rendering

```
Test: "DynamicForm renders text field from schema"
  Setup:  Schema with one field: { name: "email", type: "string" }
  Expect: Find 1 TextField widget with label "email"

Test: "DynamicForm shows validation error on invalid submit"
  Setup:  Schema with required field, form submitted with empty value
  Expect: Find error text "This field is required"

Test: "DynamicForm shows/hides conditional field"
  Setup:  Schema with field visible when shipping == "overnight"
  Action: Select "overnight" from shipping dropdown
  Expect: Conditional field appears

  Action: Select "standard" from shipping dropdown
  Expect: Conditional field disappears

Test: "DynamicForm renders nested object fields"
  Setup:  Schema with object field containing 3 sub-fields
  Expect: Find bordered section with 3 input fields inside

Test: "DynamicForm renders array field with add/remove"
  Setup:  Schema with array field, minItems: 1
  Expect: Find 1 item row, "Add" button, no "Remove" button (minItems enforced)
  Action: Tap "Add"
  Expect: Find 2 item rows, 2 "Remove" buttons
```

#### 3. Dynamic Table Rendering

```
Test: "DynamicTable renders columns from descriptor"
  Setup:  Descriptor with 3 columns: Order #, Customer, Status
  Expect: Find 3 column headers

Test: "DynamicTable sorts when column header tapped"
  Setup:  Sortable column "Customer"
  Action: Tap "Customer" header
  Expect: Sort indicator appears (▲), sort request sent with "customer_name,asc"

Test: "DynamicTable shows bulk action bar when rows selected"
  Action: Tap 2 row checkboxes
  Expect: Bulk action bar appears with "2 items selected"

Test: "DynamicTable hides low-priority columns on narrow screen"
  Setup:  Screen width 500px, columns with priorities 1 and 3
  Expect: Priority 1 column visible, priority 3 column hidden
```

#### 4. Workflow Rendering

```
Test: "WorkflowRenderer shows current step"
  Setup:  Workflow at step 2 of 4
  Expect: Step indicator shows step 2 highlighted, step 1 completed, steps 3-4 pending

Test: "WorkflowRenderer renders form step"
  Setup:  Step with renderAs: "form", schemaRef: "test-schema"
  Expect: DynamicForm rendered with the referenced schema

Test: "WorkflowRenderer handles back navigation"
  Setup:  At step 3, history has step 1 and 2
  Action: Tap "Back"
  Expect: Step 2 is displayed, step 3 data preserved

Test: "WorkflowRenderer shows waiting state for external steps"
  Setup:  Step with waitFor: "external_approval"
  Expect: Shows waiting indicator and polling status text
```

#### 5. Error State Rendering

```
Test: "ErrorBoundary catches render errors and shows fallback"
  Setup:  Child widget that throws during build
  Expect: Error card visible with "Retry" button, rest of page unaffected

Test: "Stale cache banner appears when data is stale"
  Setup:  Provider returns data marked as stale
  Expect: Banner text "Showing cached data" visible

Test: "Permission denied page shown for 403 response"
  Setup:  Page provider returns 403 error
  Expect: Access denied page with "Go to Dashboard" button
```

---

## Golden Tests

Golden tests capture a snapshot of how a widget looks (a PNG image) and compare future renders against this snapshot. If the appearance changes unexpectedly, the test fails.

### What to Golden Test

```
Test: "Shell layout at phone breakpoint"
  Setup:  Screen width 375px
  Capture: Full app shell with sidebar hidden and hamburger menu

Test: "Shell layout at tablet breakpoint"
  Setup:  Screen width 768px
  Capture: Full app shell with rail sidebar

Test: "Shell layout at desktop breakpoint"
  Setup:  Screen width 1440px
  Capture: Full app shell with expanded sidebar

Test: "Data table with 5 columns"
  Capture: Table with headers, 3 rows, pagination controls

Test: "Form with all field types"
  Capture: Form showing string, number, money, date, enum, reference, boolean fields

Test: "Workflow stepper at step 3 of 5"
  Capture: Stepper indicator showing steps 1-2 completed, 3 current, 4-5 pending

Test: "Dashboard card grid"
  Capture: 4 metric cards in a grid layout

Test: "Error boundary with error card"
  Capture: Page with one component replaced by error card
```

### How Golden Tests Work

1. First run: The test generates a PNG image and saves it as the "golden" reference
2. Subsequent runs: The test generates a new PNG and compares it pixel-by-pixel to the golden
3. If they match: Test passes
4. If they differ: Test fails and shows the difference (highlighting changed pixels)

### Updating Goldens

When you intentionally change the UI (new design, updated component style), you need to update the golden files:

```bash
flutter test --update-goldens
```

This regenerates all golden reference images. The new images should be committed to version control so the team can review the visual changes in code review.

---

## Test Organization

```
test/
├── unit/
│   ├── schema_resolver_test.dart
│   ├── form_validator_test.dart
│   ├── permission_filter_test.dart
│   ├── cache_coordinator_test.dart
│   ├── workflow_state_machine_test.dart
│   ├── navigation_builder_test.dart
│   └── ...
├── widget/
│   ├── page_renderer_test.dart
│   ├── dynamic_form_test.dart
│   ├── dynamic_table_test.dart
│   ├── workflow_renderer_test.dart
│   ├── sidebar_test.dart
│   ├── error_boundary_test.dart
│   └── ...
├── golden/
│   ├── shell_layout_test.dart
│   ├── form_layout_test.dart
│   ├── table_layout_test.dart
│   ├── workflow_stepper_test.dart
│   └── goldens/                    ← PNG reference files
│       ├── shell_phone.png
│       ├── shell_tablet.png
│       ├── shell_desktop.png
│       └── ...
└── fixtures/
    ├── page_descriptors/           ← Sample BFF JSON responses
    │   ├── orders_list.json
    │   ├── dashboard.json
    │   └── ...
    ├── schemas/
    │   ├── order_create.json
    │   └── ...
    └── navigation/
        └── main_nav.json
```

### Test Fixtures

Test fixtures are sample BFF JSON responses stored as files. Tests load these fixtures to simulate BFF data without making real network calls. This ensures:
- Tests are fast (no network dependency)
- Tests are deterministic (same data every time)
- Tests cover specific scenarios (edge cases, error cases)

---

## Running Tests

```bash
# Run all tests
flutter test

# Run only unit tests
flutter test test/unit/

# Run only widget tests
flutter test test/widget/

# Run golden tests
flutter test test/golden/

# Update golden reference images
flutter test --update-goldens

# Run with coverage
flutter test --coverage

# Generate coverage report
genhtml coverage/lcov.info -o coverage/html
```

### CI/CD Integration

Tests run automatically in the CI pipeline:
1. Every pull request: All unit and widget tests run. Golden tests run and flag visual changes.
2. Every merge to main: Full test suite including golden tests.
3. Coverage threshold: Fail the build if code coverage drops below the threshold (recommended: 80%+ for core engine).
