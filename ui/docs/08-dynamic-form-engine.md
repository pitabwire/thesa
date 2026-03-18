# 8. Dynamic Form Engine

## What Is the Form Engine?

The Form Engine is a subsystem of the UI Engine that builds forms automatically from schema definitions. Instead of a developer writing code for every form field, the BFF sends a schema that describes what fields the form should have, what types they are, what validation rules apply, and when fields should appear or disappear.

Think of the schema as a recipe for a form. The Form Engine is the cook that follows the recipe.

---

## Why Dynamic Forms?

In a traditional app, every form is handwritten:

```
// Traditional: Developer writes this for every form
TextField(label: "Customer Name", required: true)
NumberField(label: "Quantity", min: 1, max: 999)
DropdownField(label: "Status", options: ["pending", "active"])
```

In Thesa UI, forms are defined by the server:

```json
{
  "fields": [
    { "name": "customer_name", "type": "string", "label": "Customer Name", "required": true },
    { "name": "quantity", "type": "number", "label": "Quantity", "min": 1, "max": 999 },
    { "name": "status", "type": "enum", "label": "Status", "options": ["pending", "active"] }
  ]
}
```

The Form Engine reads this JSON and produces the same form. Benefits:

- **No frontend code changes** when a form changes — just update the schema on the server
- **Consistent validation** — rules come from one source (the BFF)
- **Reusable** — the same schema can generate a creation form, an edit form, and validation for both

---

## How It Works — Step by Step

### Step 1: Schema Arrives

The Form Engine receives a schema from the BFF (via `schemaProvider`). A schema looks like this:

```json
{
  "schemaId": "order-create",
  "fields": [
    {
      "name": "customer_id",
      "type": "reference",
      "label": "Customer",
      "resource": "customers",
      "displayField": "name",
      "required": true
    },
    {
      "name": "notes",
      "type": "string",
      "label": "Notes",
      "multiline": true,
      "maxLength": 500
    }
  ]
}
```

### Step 2: Schema Resolution

Before building the form, the **Schema Resolver** processes the schema:

- **`$ref` resolution**: If a field references another schema (e.g., `"$ref": "schemas/address"`), the resolver fetches that schema and inlines it
- **Nested flattening**: If a schema uses `allOf` (combine multiple schemas), the resolver merges them into one flat list of fields
- **Circular reference detection**: If schema A references B, and B references A, the resolver catches this and prevents an infinite loop

### Step 3: Field Registry Mapping

Each field in the schema has a `type`. The **Field Registry** maps types to Flutter form widgets:

| Schema Type | Flutter Widget | What the User Sees |
|---|---|---|
| `string` | Text input field | A single-line text box |
| `string` (multiline) | Multiline text input | A tall text area |
| `number` | Number input | A text box that only accepts whole numbers |
| `decimal` | Decimal input | A text box that accepts numbers with decimal places |
| `money` | Money input | A formatted currency field (e.g., "$1,234.56") |
| `date` | Date picker | A calendar popup |
| `datetime` | Date-time picker | A calendar + time picker |
| `enum` | Dropdown / radio buttons | A list of predefined choices |
| `boolean` | Checkbox / switch | A toggle |
| `reference` | Autocomplete lookup | A search field that looks up items from a resource |
| `object` | Nested form section | A group of sub-fields rendered inside a bordered section |
| `array` | Repeatable field group | A list of items with "Add" and "Remove" buttons |

### Step 4: Validation Rule Application

Each field can have validation rules defined in the schema. These are applied to the form controls:

| Rule | What It Does | Example |
|---|---|---|
| `required: true` | Field must not be empty | "Customer is required" |
| `minLength: 3` | Text must be at least N characters | "Name must be at least 3 characters" |
| `maxLength: 500` | Text must be at most N characters | "Notes must be 500 characters or less" |
| `min: 1` | Number must be at least N | "Quantity must be at least 1" |
| `max: 9999` | Number must be at most N | "Quantity must be 9999 or less" |
| `pattern: "^[A-Z]{3}$"` | Text must match a regular expression | "Code must be 3 uppercase letters" |
| `email: true` | Text must be a valid email address | "Please enter a valid email" |
| `custom` | A named server-side validation rule | Custom message from BFF |

Validation happens:
1. **On field blur** (when the user moves to the next field) — shows inline errors
2. **On form submit** (when the user clicks Save/Submit) — validates all fields
3. **Server-side** (when the BFF responds with validation errors) — shows errors from the server

### Step 5: Visibility Rule Application

Some fields should only appear under certain conditions. The schema defines these rules:

```json
{
  "name": "expedite_reason",
  "type": "string",
  "label": "Reason for Expediting",
  "visibleWhen": {
    "field": "shipping_method",
    "equals": "overnight"
  }
}
```

This means: "Only show the 'Reason for Expediting' field when the user has selected 'overnight' as the shipping method."

The **Visibility Engine** evaluates these rules reactively. When the user changes the shipping method:
- If they select "overnight" → the expedite reason field smoothly appears (with animation)
- If they select anything else → the field smoothly disappears
- If a hidden field had a value, it is cleared (to prevent submitting invisible data)

### Step 6: Readonly Rules

Some fields should be visible but not editable:

```json
{
  "name": "order_total",
  "type": "money",
  "label": "Order Total",
  "readonly": true,
  "computedFrom": "items.sum(unit_price * quantity)"
}
```

Readonly fields:
- Appear as normal fields but with a greyed-out/non-editable appearance
- May display computed values (calculated by the server or derived from other fields)
- Cannot be modified by the user

---

## Field Types — Detailed Explanations

### String Field

The simplest field type. Renders as a text input.

**Schema:**
```json
{
  "name": "customer_name",
  "type": "string",
  "label": "Customer Name",
  "placeholder": "Enter customer name",
  "required": true,
  "minLength": 2,
  "maxLength": 100
}
```

**Renders as:**
```
Customer Name *
┌──────────────────────────────────┐
│ Enter customer name              │
└──────────────────────────────────┘
```

If `multiline: true`, renders as a text area:
```
Notes
┌──────────────────────────────────┐
│                                  │
│                                  │
│                                  │
└──────────────────────────────────┘
42 / 500 characters
```

### Number Field

Accepts only whole numbers. Shows numeric keyboard on mobile.

**Schema:**
```json
{
  "name": "quantity",
  "type": "number",
  "label": "Quantity",
  "min": 1,
  "max": 9999,
  "required": true
}
```

**Renders as:**
```
Quantity *
┌──────────────────────────────────┐
│ 1                          [- +] │
└──────────────────────────────────┘
```

Includes optional increment/decrement buttons. Shows validation error if value is outside min/max range.

### Decimal Field

Like a number field, but accepts decimal places.

**Schema:**
```json
{
  "name": "weight_kg",
  "type": "decimal",
  "label": "Weight (kg)",
  "precision": 2,
  "min": 0.01
}
```

**Renders as:**
```
Weight (kg)
┌──────────────────────────────────┐
│ 0.00                             │
└──────────────────────────────────┘
```

### Money Field

A specialized decimal field for currency values. Formats with currency symbol, thousands separators, and the correct number of decimal places.

**Schema:**
```json
{
  "name": "unit_price",
  "type": "money",
  "label": "Unit Price",
  "currency": "USD",
  "required": true
}
```

**Renders as:**
```
Unit Price *
┌──────────────────────────────────┐
│ $ 0.00                           │
└──────────────────────────────────┘
```

As the user types, the field auto-formats: typing "1234" becomes "$12.34", typing "100000" becomes "$1,000.00".

### Date/Time Fields

**Date only:**
```
Order Date *
┌──────────────────────────┐
│ Feb 6, 2026         [📅] │
└──────────────────────────┘
```

Tapping the calendar icon opens a date picker dialog. The display format respects the user's locale.

**Date + time:**
```
Delivery Time *
┌──────────────────────────────────┐
│ Feb 6, 2026 at 2:30 PM    [📅]  │
└──────────────────────────────────┘
```

### Enum Field

Displays a set of predefined choices. Renders differently based on the number of options:

**Few options (2-4) → Radio buttons or chips:**
```
Shipping Method *
  ○ Standard
  ● Express
  ○ Overnight
```

**Many options (5+) → Dropdown:**
```
Country *
┌──────────────────────────────────┐
│ United States                  ▾ │
└──────────────────────────────────┘
```

**Multi-select → Checkboxes or multi-select chips:**
```
Categories
  ☑ Electronics
  ☐ Clothing
  ☑ Books
  ☐ Home & Garden
```

### Reference Field

A lookup field that searches a remote resource. The user types to search, and the field shows matching results from the BFF.

**Schema:**
```json
{
  "name": "customer_id",
  "type": "reference",
  "label": "Customer",
  "resource": "customers",
  "displayField": "name",
  "searchable": true,
  "required": true
}
```

**Renders as:**
```
Customer *
┌──────────────────────────────────┐
│ Ali                              │
└──────────────────────────────────┘
  ┌──────────────────────────────┐
  │ Alice Smith (alice@mail.com) │
  │ Ali Hassan (ali@mail.com)    │
  └──────────────────────────────┘
```

When the user types, the Form Engine debounces the input (waits 300ms after the last keystroke) and queries `GET /ui/resources/customers?q=Ali` to fetch matching results. The user selects from the dropdown, and the internal value stored is the `customer_id`.

### Nested Object Field

When a field contains sub-fields, the Form Engine renders a bordered section with the sub-fields inside:

**Schema:**
```json
{
  "name": "billing_address",
  "type": "object",
  "label": "Billing Address",
  "fields": [
    { "name": "street", "type": "string", "label": "Street", "required": true },
    { "name": "city", "type": "string", "label": "City", "required": true },
    { "name": "state", "type": "string", "label": "State" },
    { "name": "zip", "type": "string", "label": "ZIP Code", "pattern": "^\\d{5}$" }
  ]
}
```

**Renders as:**
```
Billing Address
┌─────────────────────────────────────────────┐
│ Street *                                     │
│ ┌─────────────────────────────────────────┐ │
│ │                                         │ │
│ └─────────────────────────────────────────┘ │
│                                              │
│ City *                    State               │
│ ┌──────────────────┐     ┌──────────────┐   │
│ │                  │     │              │   │
│ └──────────────────┘     └──────────────┘   │
│                                              │
│ ZIP Code                                     │
│ ┌──────────────────┐                        │
│ │                  │                        │
│ └──────────────────┘                        │
└─────────────────────────────────────────────┘
```

### Array Field

A repeatable group of fields. The user can add and remove entries.

**Schema:**
```json
{
  "name": "line_items",
  "type": "array",
  "label": "Line Items",
  "minItems": 1,
  "maxItems": 50,
  "itemSchema": {
    "fields": [
      { "name": "product_id", "type": "reference", "label": "Product", "resource": "products", "required": true },
      { "name": "quantity", "type": "number", "label": "Qty", "min": 1, "required": true },
      { "name": "unit_price", "type": "money", "label": "Price", "readonly": true }
    ]
  }
}
```

**Renders as:**
```
Line Items (2 items)

  Item 1                                              [🗑 Remove]
  ┌─────────────────────────────────────────────────────────────┐
  │ Product *              Qty *           Price                 │
  │ [Widget Pro       ]    [2     ]        $49.99               │
  └─────────────────────────────────────────────────────────────┘

  Item 2                                              [🗑 Remove]
  ┌─────────────────────────────────────────────────────────────┐
  │ Product *              Qty *           Price                 │
  │ [Gadget Lite      ]    [1     ]        $42.52               │
  └─────────────────────────────────────────────────────────────┘

  [+ Add Item]
```

The "Add Item" button adds a new empty row. The "Remove" button removes a row (with a confirmation if configured). `minItems: 1` means the last item cannot be removed.

---

## Conditional Fields — How They Work

Conditional visibility lets the BFF define dynamic forms where fields appear or disappear based on other fields' values:

### Supported Conditions

| Condition | Meaning | Example |
|---|---|---|
| `equals` | Show when field X has value Y | `"field": "type", "equals": "premium"` |
| `notEquals` | Show when field X does NOT have value Y | `"field": "status", "notEquals": "closed"` |
| `in` | Show when field X is one of several values | `"field": "country", "in": ["US", "CA"]` |
| `isNotEmpty` | Show when field X has any value | `"field": "email", "isNotEmpty": true` |
| `isEmpty` | Show when field X is empty | `"field": "referral", "isEmpty": true` |
| `greaterThan` | Show when field X > N | `"field": "amount", "greaterThan": 1000` |

### Multiple Conditions

Conditions can be combined:

```json
{
  "name": "large_order_approval",
  "type": "reference",
  "label": "Approval Manager",
  "visibleWhen": {
    "all": [
      { "field": "total", "greaterThan": 10000 },
      { "field": "type", "equals": "new_customer" }
    ]
  }
}
```

This field only appears when BOTH conditions are true: total > $10,000 AND customer type is "new_customer".

---

## Validation Error Display

### Inline Errors

Each field shows its error directly below the input:

```
Email *
┌──────────────────────────────────┐
│ not-an-email                     │
└──────────────────────────────────┘
⚠ Please enter a valid email address
```

The error appears when:
- The user moves away from the field (blur event)
- The user attempts to submit the form

The error disappears when:
- The user corrects the value and the validation passes

### Server-Side Errors

When the form is submitted and the BFF returns validation errors, they are mapped to the corresponding fields:

```json
{
  "error": "validation_failed",
  "fields": {
    "customer_id": "Customer account is suspended",
    "items[0].quantity": "Insufficient stock for Widget Pro"
  }
}
```

The Form Engine maps these errors to the correct fields, including nested fields within arrays (e.g., `items[0].quantity` maps to the quantity field in the first line item).

### Form-Level Errors

Some errors are not specific to a field:

```json
{
  "error": "validation_failed",
  "message": "Cannot create order: credit limit exceeded"
}
```

These appear as a banner at the top of the form:

```
┌──────────────────────────────────────────────────┐
│ ⚠ Cannot create order: credit limit exceeded     │
└──────────────────────────────────────────────────┘
```

---

## Responsive Form Layout

Forms adapt to screen size:

| Screen Size | Layout |
|---|---|
| Desktop (>1200px) | Fields arranged in 2-3 columns where appropriate |
| Tablet (600-1200px) | Fields arranged in 2 columns |
| Phone (<600px) | All fields stacked in a single column |

The Form Engine uses the schema's `layout` hints to decide which fields can share a row:

```json
{
  "layout": {
    "rows": [
      ["first_name", "last_name"],
      ["email"],
      ["city", "state", "zip"]
    ]
  }
}
```

On desktop: first_name and last_name appear side by side. On phone: they stack vertically.
