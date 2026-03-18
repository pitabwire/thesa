# 13. Plugin / Extension System

## What Is the Plugin System?

The Plugin System allows developers to register custom Flutter widgets that override the default rendering for specific pages, components, or schemas. It is the escape hatch that lets domain-specific teams add specialized UX without modifying the core platform.

Think of it like a picture frame shop:
- The shop (Thesa UI) provides **standard frames** that fit any picture
- But for a special painting, you can bring a **custom frame** that fits that painting perfectly
- All other paintings still use the standard frames

---

## Why Plugins?

The dynamic UI Engine is powerful — it can render tables, forms, dashboards, and workflows from BFF descriptors. But sometimes the generic rendering is not enough:

- An **order tracking page** might need a map widget showing the delivery route
- A **financial dashboard** might need a custom interactive chart library
- A **scheduling page** might need a drag-and-drop calendar
- A **media library** might need a custom image gallery with crop and preview

These specialized UIs are impossible to describe in a generic BFF descriptor. The plugin system solves this by letting a developer write custom Flutter code for specific cases while keeping everything else generic.

---

## Three Types of Plugins

### 1. Page Plugin

**What it does**: Replaces the entire page rendering for a specific `pageId`.

**When to use**: When the BFF's generic page descriptor cannot express the UX you need for a particular page. The entire page — layout, components, interactions — is custom.

**How it works**:
```
Normal flow:
  User navigates to /orders/ORD-1234
  → routerProvider sends pageId "order-detail" to PageRenderer
  → PageRenderer reads the page descriptor and renders generic components

With page plugin:
  User navigates to /orders/ORD-1234
  → routerProvider sends pageId "order-detail" to PageRenderer
  → PageRenderer checks PluginRegistry: "Is there a page plugin for 'order-detail'?"
  → YES → PageRenderer hands control to the plugin's widget
  → The plugin renders a fully custom order detail page
```

**What the plugin receives**:
- The raw `PageDescriptor` (so it can still use data from the BFF)
- A `WidgetRef` (so it can access Riverpod providers — schemas, actions, permissions)

**What the plugin can do**:
- Render completely custom widgets (maps, 3D views, calendars)
- Delegate parts of the page back to the generic engine (e.g., use `DynamicForm` for the order fields, but add a custom map widget)
- Access permissions from the descriptor to show/hide elements
- Trigger BFF actions using `actionProvider`

**Example**: A custom order detail page that shows a delivery tracking map:
```
┌─────────────────────────────────────────────────────────┐
│ Order #ORD-1234                           [Edit] [Ship] │
├────────────────────────┬────────────────────────────────┤
│ Order Details          │ Delivery Tracking              │
│ (rendered by           │ (custom map widget             │
│  DynamicForm —         │  that only exists in           │
│  generic engine)       │  this plugin)                  │
│                        │                                │
│ Customer: Alice Smith  │  [====== 🚚 ======>           │
│ Total: $142.50         │   Chicago    →    New York     │
│ Status: Shipped        │                                │
│                        │  ETA: Feb 8, 3:00 PM           │
└────────────────────────┴────────────────────────────────┘
```

The left side uses `DynamicForm` from the generic engine. The right side is a custom map widget written specifically for this plugin.

### 2. Component Plugin

**What it does**: Replaces the rendering for a specific component type across the entire app.

**When to use**: When you need a specialized widget for a particular component type that the BFF describes. Unlike a page plugin (which replaces one page), a component plugin replaces a component type everywhere it appears.

**How it works**:
```
Normal flow:
  PageRenderer encounters component type "status_badge"
  → ComponentRegistry returns the built-in StatusBadge widget

With component plugin:
  Developer registers: "For component type 'delivery_map', use my CustomDeliveryMap widget"
  → PageRenderer encounters component type "delivery_map"
  → Checks PluginRegistry first → finds CustomDeliveryMap
  → Renders CustomDeliveryMap instead of an "Unknown Component" placeholder
```

**Example**: Registering a custom map component:
```
pluginRegistry.registerComponent(
  componentType: 'delivery_map',
  builder: (descriptor, ref) => DeliveryMapWidget(
    latitude: descriptor.properties['latitude'],
    longitude: descriptor.properties['longitude'],
    route: descriptor.properties['route'],
  ),
);
```

Now any page that includes a `"type": "delivery_map"` component will render the custom map widget.

### 3. Schema Renderer Plugin

**What it does**: Replaces the form rendering for a specific schema.

**When to use**: When the generic `DynamicForm` does not provide the right UX for a particular data type. For example, a "color picker" schema needs a specialized UI that the generic form engine cannot produce.

**How it works**:
```
Normal flow:
  DynamicForm receives schema "color-selection"
  → Renders generic text/enum fields from the schema

With schema renderer plugin:
  Developer registers: "For schema 'color-selection', use my ColorPickerForm widget"
  → DynamicForm checks PluginRegistry: "Is there a schema renderer for 'color-selection'?"
  → YES → Uses ColorPickerForm instead of generic fields
```

---

## Plugin Registration

### When Registration Happens

Plugins are registered during app initialization, before the first screen renders. The typical flow:

```
main.dart:
  1. Initialize Drift database
  2. Initialize secure storage
  3. Register plugins         ← here
  4. Create ProviderScope
  5. Run the app
```

### How Registration Works

The `PluginRegistry` is a simple registry that stores mappings:

```
PluginRegistry:
  pagePlugins:
    "order-detail"    → CustomOrderDetailPage builder
    "scheduling"      → CustomSchedulingPage builder

  componentPlugins:
    "delivery_map"    → DeliveryMapWidget builder
    "color_picker"    → ColorPickerWidget builder
    "gantt_chart"     → GanttChartWidget builder

  schemaRendererPlugins:
    "color-selection" → ColorPickerForm builder
    "address-intl"    → InternationalAddressForm builder
```

### Registration API

```
// Register a page plugin
pluginRegistry.registerPage(
  pageId: 'order-detail',
  builder: (context, descriptor, ref) => CustomOrderDetailPage(
    descriptor: descriptor,
    ref: ref,
  ),
);

// Register a component plugin
pluginRegistry.registerComponent(
  componentType: 'delivery_map',
  builder: (descriptor, ref) => DeliveryMapWidget(descriptor: descriptor),
);

// Register a schema renderer plugin
pluginRegistry.registerSchemaRenderer(
  schemaId: 'color-selection',
  builder: (schema, ref, onSaved) => ColorPickerForm(
    schema: schema,
    onSaved: onSaved,
  ),
);
```

---

## Resolution Order

When the UI Engine needs to render something, it checks for plugins FIRST, then falls back to the generic engine:

### Page Rendering Resolution

```
Step 1: Is there a page plugin for this pageId?
  │
  ├── YES → Use the plugin's page widget. Done.
  │
  └── NO → Step 2
              │
              ▼
Step 2: Use the generic PageRenderer.
  For each component in the page:

    Step 2a: Is there a component plugin for this type?
      │
      ├── YES → Use the plugin's component widget.
      │
      └── NO → Step 2b
                  │
                  ▼
    Step 2b: Is this a known built-in component type?
      │
      ├── YES → Use the built-in widget from ComponentRegistry.
      │
      └── NO → Render "Unknown Component" placeholder.
```

### Form Rendering Resolution

```
Step 1: Is there a schema renderer plugin for this schemaId?
  │
  ├── YES → Use the plugin's form widget. Done.
  │
  └── NO → Use the generic DynamicForm.
```

### Key Rule: Plugins Always Win

If a plugin is registered for a specific pageId, componentType, or schemaId, it **always** takes precedence over the built-in generic renderer. This ensures that domain teams can override any part of the UI without worrying about conflicts with the core platform.

---

## Plugin Design Guidelines

### DO:

- **Use the descriptor data**: Even in a custom plugin, read the `PageDescriptor` or `ComponentDescriptor` for permissions, action definitions, and metadata. Do not ignore the BFF's data.
- **Respect permissions**: Check `allowed` flags from the descriptor. If the BFF says an action is not allowed, the plugin must not show it.
- **Delegate to the engine when possible**: If 80% of your page is standard (tables, forms, cards) and only 20% is custom, use the generic engine for the 80% and only write custom code for the 20%.
- **Follow the design system**: Use the app's color tokens, typography, and spacing so the custom page looks consistent with the rest of the app.
- **Handle errors**: Wrap custom widgets in error boundaries. If your map widget fails, show a graceful error — do not crash the page.

### DON'T:

- **Don't call BFF endpoints directly**: Use the providers. They handle caching, auth, retries, and error handling.
- **Don't duplicate permission logic**: Read `allowed` flags from descriptors. Do not re-implement permission checks.
- **Don't hardcode BFF URLs**: Use the networking layer's BFF client.
- **Don't break the responsive layout**: Test your plugin at all breakpoints.

---

## Example: Custom Order Detail Page Plugin

### Scenario

The generic `PageRenderer` can display order details using cards and forms. But the business wants a custom order detail page with:

1. A delivery tracking map (not possible with generic components)
2. A real-time status timeline
3. A split layout: order details on the left, tracking on the right

### Implementation Approach

```
CustomOrderDetailPage (plugin):
│
├── Left panel:
│   ├── DynamicCard (generic) — displays order info from descriptor
│   ├── DynamicForm (generic) — displays editable fields from schema
│   └── ActionButtons (generic) — displays allowed actions from descriptor
│
└── Right panel:
    ├── DeliveryTrackingMap (custom) — shows map with delivery route
    ├── StatusTimeline (custom) — shows real-time status events
    └── ETAWidget (custom) — shows estimated delivery time
```

The plugin:
- **Delegates** the left panel to the generic engine (DynamicCard, DynamicForm, ActionButtons)
- **Adds** custom widgets on the right that cannot be expressed generically
- **Reads** the page descriptor for permissions and action definitions
- **Uses** `actionProvider` when the user clicks "Ship" or "Cancel"

### Registration

```
// In the app's plugin registration file:
pluginRegistry.registerPage(
  pageId: 'order-detail',
  builder: (context, descriptor, ref) => CustomOrderDetailPage(
    descriptor: descriptor,
    ref: ref,
  ),
);
```

### Result

When any user navigates to a URL like `/orders/ORD-1234`, the router loads `pageId: 'order-detail'`, the PageRenderer finds the page plugin, and renders the custom page. All permission checks still apply. All actions still go through the BFF. The custom page is just a different way of presenting the same data.

---

## Plugin Packaging

### In-App Plugins

For small teams or single-product deployments, plugins can live directly in the `lib/plugins/` folder of the Thesa UI project.

### Package Plugins

For larger organizations where different teams own different domains, plugins should be packaged as separate Dart packages:

```
packages/
├── order_tracking_plugin/
│   ├── lib/
│   │   ├── order_tracking_plugin.dart  ← registers the plugin
│   │   ├── widgets/
│   │   │   ├── delivery_map.dart
│   │   │   └── status_timeline.dart
│   │   └── ...
│   └── pubspec.yaml
│
├── scheduling_plugin/
│   ├── lib/
│   │   └── ...
│   └── pubspec.yaml
```

Each package is added as a dependency in the main app's `pubspec.yaml`. At startup, each plugin's registration function is called.

This separation means:
- Domain teams can develop and test their plugins independently
- Plugins can be version-controlled separately
- Adding a new domain's customizations is just adding a package dependency
