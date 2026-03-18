# 12. Responsive Layout Strategy

## What Is Responsive Design?

Responsive design means the application's layout adapts to the screen size. The same app looks and works well whether the user is on a 5-inch phone, a 10-inch tablet, a 14-inch laptop, or a 32-inch desktop monitor.

This is NOT about "shrinking" the desktop layout to fit a phone. It is about deliberately choosing what to show, how to arrange it, and how interactions work for each screen size.

---

## Why This Matters for Thesa UI

Enterprise users access admin tools from varied devices:
- A warehouse manager checks orders on a tablet
- An executive reviews a dashboard on a phone while travelling
- An operations analyst works with large tables on a wide desktop monitor
- A developer debugs a workflow on a laptop

The same BFF data must present well on all of these.

---

## Breakpoint Definitions

A breakpoint is a screen width at which the layout changes. Thesa UI defines five breakpoints:

| Name | Width Range | Typical Devices | Layout Strategy |
|---|---|---|---|
| **Phone** | < 600px | iPhone, Android phones | Single column, bottom navigation or hamburger |
| **Tablet** | 600px – 960px | iPad Mini, small tablets, large phones in landscape | Single column, sidebar rail (icons only) |
| **Laptop** | 960px – 1200px | Laptops, small desktops | Two columns where useful, narrow sidebar |
| **Desktop** | 1200px – 1600px | Standard desktop monitors | Multi-column, full sidebar |
| **Wide** | > 1600px | Ultra-wide monitors, dual screens | Three columns, sidebar + content + detail panel |

### How Breakpoints Are Used

The app does not check the exact pixel width every time. Instead, it checks which breakpoint range the current width falls into and applies that breakpoint's rules. When the window is resized across a breakpoint boundary, the layout transitions smoothly.

---

## Shell Layout at Each Breakpoint

The "shell" is the outer frame of the app: sidebar, top bar, and content area. Here is how it adapts:

### Phone (< 600px)

```
┌─────────────────────────────┐
│ ☰ Dashboard        [🔔] [👤]│  ← top bar with hamburger menu
├─────────────────────────────┤
│                             │
│                             │
│    (full-width content)     │
│                             │
│                             │
│                             │
└─────────────────────────────┘
```

- **Sidebar**: Hidden. Accessible via hamburger menu (☰) which opens a drawer overlay.
- **Content**: Full width, single column.
- **Navigation**: Primary routes may optionally appear as bottom tabs if the BFF defines `bottomNav: true`.
- **Top bar**: Shows page title, notification icon, user avatar.

### Tablet (600px – 960px)

```
┌──┬──────────────────────────┐
│📊│ Dashboard        [🔔] [👤]│
│🛒│                          │
│📦│   (content area)         │
│  │                          │
│  │                          │
│⚙│                          │
└──┴──────────────────────────┘
```

- **Sidebar**: Rail mode — icons only, no labels. Width: ~72px.
- **Content**: Takes most of the screen width.
- **Hovering**: Hovering over a rail icon shows a tooltip with the label.
- **Expanding**: Tapping a parent icon shows a flyout menu with children.

### Laptop (960px – 1200px)

```
┌────────┬────────────────────┐
│ 📊 Dash│ Dashboard  [🔔] [👤]│
│ 🛒 Ord │                    │
│   ├ All│   (content area)   │
│   └ Ret│                    │
│        │                    │
│ ⚙ Set │                    │
└────────┴────────────────────┘
```

- **Sidebar**: Narrow mode — icons + short labels. Width: ~200px.
- **Content**: Remaining width.
- **Sidebar groups**: Expanded items show children with indentation.

### Desktop (1200px – 1600px)

```
┌────────────┬────────────────────────────────┐
│ 📊 Dashboard│ Dashboard              [🔔] [👤]│
│            │                                │
│ 🛒 Orders  │                                │
│   ├ All    │        (content area)          │
│   └ Returns│                                │
│            │                                │
│ 📦 Products│                                │
│            │                                │
│ ⚙ Settings│                                │
└────────────┴────────────────────────────────┘
```

- **Sidebar**: Full mode — icons + full labels. Width: ~280px.
- **Content**: Generous width for tables and forms.

### Wide (> 1600px)

```
┌────────────┬────────────────────────┬───────────────┐
│ 📊 Dashboard│ Orders                 │ Order #1234   │
│            │                        │               │
│ 🛒 Orders  │ ☐ │ ORD-1234│Alice│$142│ Customer:     │
│   ├ All    │ ☐ │ ORD-1235│Bob  │ $89│ Alice Smith   │
│   └ Returns│                        │               │
│            │                        │ Total: $142.50│
│ 📦 Products│                        │               │
│            │                        │ Status: Pend  │
│ ⚙ Settings│                        │               │
└────────────┴────────────────────────┴───────────────┘
```

- **Sidebar**: Full mode (same as desktop).
- **Content**: Split into master (list) and detail panels.
- **Detail panel**: Shows details of the selected item without navigating away from the list.

---

## Content Layout Adaptations

Beyond the shell, the content within pages also adapts:

### Forms

| Breakpoint | Form Layout |
|---|---|
| Phone | Single column — all fields stacked vertically |
| Tablet | Single column — wider fields |
| Laptop | Two columns — short fields placed side by side (e.g., First Name + Last Name) |
| Desktop | Two or three columns as defined by the schema's layout hints |
| Wide | Same as desktop (forms do not need more than 3 columns) |

Example — Address form on phone vs. desktop:

**Phone:**
```
Street *
┌──────────────────────────┐
│                          │
└──────────────────────────┘

City *
┌──────────────────────────┐
│                          │
└──────────────────────────┘

State
┌──────────────────────────┐
│                          │
└──────────────────────────┘

ZIP Code
┌──────────────────────────┐
│                          │
└──────────────────────────┘
```

**Desktop:**
```
Street *
┌────────────────────────────────────────────────────────┐
│                                                        │
└────────────────────────────────────────────────────────┘

City *                          State            ZIP Code
┌──────────────────────┐ ┌──────────────┐ ┌─────────────┐
│                      │ │              │ │             │
└──────────────────────┘ └──────────────┘ └─────────────┘
```

### Tables

Tables use column prioritization (see Section 9 for details):

| Breakpoint | Visible Columns |
|---|---|
| Phone | Priority 1 only (2-3 essential columns) |
| Tablet | Priority 1 + 2 (4-5 columns) |
| Desktop+ | All columns |

On phone, tapping a table row expands it to show hidden column data:

```
│ ORD-1234 │ Alice │ ● Pend │  ← tap
  ┌────────────────────────┐
  │ Total: $142.50         │   ← expanded details
  │ Date: Feb 6, 2026      │
  │ [View] [Edit]          │   ← row actions
  └────────────────────────┘
```

### Dashboard Cards

Dashboard metric cards and chart widgets arrange in a responsive grid:

| Breakpoint | Grid Columns | Card Size |
|---|---|---|
| Phone | 1 column | Full width, stacked |
| Tablet | 2 columns | Half width each |
| Laptop | 3 columns | Third width each |
| Desktop | 4 columns | Quarter width each |
| Wide | 4 columns | Quarter width (with more padding) |

```
Desktop (4 columns):
┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐
│ Orders   │ │ Revenue  │ │ Customers│ │ Returns  │
│  1,247   │ │ $89,432  │ │   892    │ │    23    │
└──────────┘ └──────────┘ └──────────┘ └──────────┘

Phone (1 column):
┌──────────────────────────────┐
│ Orders: 1,247   (+12% ▲)    │
├──────────────────────────────┤
│ Revenue: $89,432  (+8% ▲)   │
├──────────────────────────────┤
│ Customers: 892    (+5% ▲)   │
├──────────────────────────────┤
│ Returns: 23       (-2% ▼)   │
└──────────────────────────────┘
```

### Action Buttons

| Breakpoint | Button Display |
|---|---|
| Desktop | Full text buttons in the page header: `[+ New Order]  [Export]  [Settings]` |
| Tablet | Icon buttons with tooltips: `[+]  [⬇]  [⚙]` |
| Phone | Single primary action as FAB (floating action button), others in overflow menu (`⋮`) |

```
Phone:
┌─────────────────────────────┐
│ Orders                  [⋮] │  ← overflow menu
│                             │
│                         (●) │  ← floating action button (+ New Order)
└─────────────────────────────┘

Overflow menu:
  ┌─────────────────┐
  │ Export CSV       │
  │ Settings         │
  │ Bulk Edit        │
  └─────────────────┘
```

### Dialogs and Modals

| Breakpoint | Dialog Style |
|---|---|
| Desktop | Centered modal dialog (width: 400-600px) |
| Tablet | Centered modal dialog (width: ~90% of screen) |
| Phone | Full-screen page (bottom sheet for small dialogs) |

On phone, a "New Order" dialog becomes a full-screen page with a back button, because there is not enough room for a floating modal.

---

## How the BFF Influences Responsive Behavior

The BFF does not control responsive breakpoints directly. Instead, it provides **hints** that the responsive system interprets:

### Column Priority

```json
{ "field": "customer_name", "priority": 1 }  → visible on all screens
{ "field": "total", "priority": 2 }          → hidden on phone
{ "field": "created_at", "priority": 3 }     → hidden on phone and tablet
```

### Layout Hints

```json
{
  "layout": {
    "type": "grid",
    "columns": { "phone": 1, "tablet": 2, "desktop": 4 }
  }
}
```

### Action Position

```json
{
  "actionId": "create-order",
  "position": "header",
  "mobilePosition": "fab"
}
```

The BFF suggests where actions should appear, and the responsive system adapts them to the current breakpoint.

---

## Implementation Approach

### Material 3 Adaptive Scaffold

Thesa UI uses `flutter_adaptive_scaffold` from the Material 3 package. This provides:

- `AdaptiveScaffold`: Automatically switches between bottom nav (phone), rail (tablet), and drawer (desktop)
- `SlotLayout`: Defines what appears in each layout slot (body, sidebar, detail panel) at each breakpoint
- Built-in animations for transitions between layouts

### MediaQuery and LayoutBuilder

For fine-grained responsive decisions within components:

- `MediaQuery.sizeOf(context)`: Gets the screen size. Used at the shell level.
- `LayoutBuilder`: Gets the available space for a specific widget. Used within components to adapt independently of the screen size. This is important because a component might be in a narrow panel on a wide screen.

### Responsive Testing

Responsive layouts are tested at each breakpoint using golden tests (see Section 18). This ensures that layout changes do not introduce regressions. If a new feature breaks the phone layout, the golden test catches it before deployment.
