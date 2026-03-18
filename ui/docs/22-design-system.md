# 22. UI Theming and Design System

## What Is a Design System?

A design system is a set of rules that define how the app looks. It specifies:
- What colors to use (and when to use each one)
- What fonts to use (and at what sizes)
- How much space to put between elements
- How buttons, cards, inputs, and other components should look
- How the app should look in light mode vs. dark mode

Without a design system, each developer makes their own choices — one uses blue buttons, another uses green, font sizes vary randomly, and the app looks inconsistent.

With a design system, everyone uses the same visual vocabulary, and the app looks professional and cohesive.

---

## Design Tokens

Design tokens are **named values** for visual properties. Instead of writing a raw color code everywhere, you use a named token. This means changing the brand color requires updating ONE value.

### Color Tokens

Colors are organized into semantic categories — the token name describes the PURPOSE, not the color:

| Token Name | Purpose | Light Mode Value | Dark Mode Value |
|---|---|---|---|
| `colorPrimary` | Main brand color, primary buttons, active states | `#1565C0` (blue) | `#90CAF9` (light blue) |
| `colorOnPrimary` | Text/icons on primary-colored backgrounds | `#FFFFFF` (white) | `#0D1B2A` (dark) |
| `colorSecondary` | Secondary buttons, less prominent accents | `#7B1FA2` (purple) | `#CE93D8` (light purple) |
| `colorSurface` | Background of cards, dialogs, inputs | `#FFFFFF` (white) | `#1E1E1E` (dark grey) |
| `colorOnSurface` | Text/icons on surface backgrounds | `#212121` (near-black) | `#E0E0E0` (light grey) |
| `colorBackground` | Page background | `#FAFAFA` (light grey) | `#121212` (near-black) |
| `colorError` | Error states, destructive actions | `#D32F2F` (red) | `#EF9A9A` (light red) |
| `colorOnError` | Text/icons on error backgrounds | `#FFFFFF` | `#1B0000` |
| `colorSuccess` | Success states, confirmations | `#388E3C` (green) | `#A5D6A7` (light green) |
| `colorWarning` | Warning states, cautions | `#F57C00` (orange) | `#FFB74D` (light orange) |
| `colorInfo` | Informational states, tips | `#1976D2` (blue) | `#64B5F6` (light blue) |

#### Status Colors

Status badges use specific colors based on the status value:

| Status | Color Token | Appearance |
|---|---|---|
| pending | `colorWarning` | ● Pending (orange/yellow) |
| active / processing | `colorInfo` | ● Active (blue) |
| shipped / in_transit | `colorInfo` | ● Shipped (blue) |
| completed / delivered | `colorSuccess` | ● Delivered (green) |
| cancelled / failed | `colorError` | ● Cancelled (red) |
| draft | `colorOnSurface` (muted) | ● Draft (grey) |

### Typography Scale

The typography scale defines a hierarchy of text sizes:

| Token Name | Size | Weight | Usage |
|---|---|---|---|
| `displayLarge` | 32px | Bold | Main page titles on dashboards |
| `headlineMedium` | 24px | Semi-bold | Page titles |
| `titleLarge` | 20px | Semi-bold | Section headings, card titles |
| `titleMedium` | 16px | Semi-bold | Sub-section headings |
| `bodyLarge` | 16px | Regular | Primary body text |
| `bodyMedium` | 14px | Regular | Secondary body text, table cells |
| `bodySmall` | 12px | Regular | Captions, timestamps, help text |
| `labelLarge` | 14px | Medium | Button text |
| `labelMedium` | 12px | Medium | Badge text, tag text |
| `labelSmall` | 11px | Medium | Overline text, tiny labels |

**Font family**: The default is the system font (SF Pro on Apple, Roboto on Android/web). Enterprise deployments can override this with their brand font.

### Spacing Scale

Spacing follows a consistent scale based on a 4px base unit:

| Token Name | Value | Usage |
|---|---|---|
| `space2` | 2px | Tight spacing (inside dense components) |
| `space4` | 4px | Minimal spacing (icon-to-text gap) |
| `space8` | 8px | Compact spacing (between related items) |
| `space12` | 12px | Default padding inside components |
| `space16` | 16px | Standard spacing between components |
| `space24` | 24px | Generous spacing between sections |
| `space32` | 32px | Large spacing between page sections |
| `space48` | 48px | Extra large spacing (page top margin) |

**Why a scale?** Without a scale, developers use arbitrary values (13px, 17px, 23px) and the layout looks uneven. With a scale, spacing is always proportional and the layout looks intentional.

---

## Component States

Every interactive component (buttons, inputs, checkboxes, etc.) has multiple visual states:

### Button States

| State | Appearance | When |
|---|---|---|
| **Default** | Solid color, normal text | Button is ready to be clicked |
| **Hover** | Slightly lighter color | Mouse hovers over (desktop only) |
| **Pressed** | Slightly darker color | User is clicking/tapping |
| **Focused** | Outline ring around button | Keyboard navigation has focused the button |
| **Disabled** | Greyed out, 50% opacity | Button cannot be interacted with |
| **Loading** | Spinner replaces text/icon | Action is in progress |

### Input Field States

| State | Appearance | When |
|---|---|---|
| **Default** | Grey border | Field is ready for input |
| **Hover** | Darker border | Mouse hovers (desktop) |
| **Focused** | Primary color border, label animates | Field has keyboard focus |
| **Filled** | Default border, value displayed | Field has a value but is not focused |
| **Error** | Red border, error message below | Validation failed |
| **Disabled** | Greyed background | Field cannot be edited |
| **Readonly** | No border change, value not editable | Display only |

---

## Dark Mode / Light Mode

The design system supports both light and dark themes. The active theme is determined by:

1. **User preference**: If the user has explicitly chosen light or dark in settings
2. **System preference**: If no user preference, follow the OS/browser setting
3. **Default**: Light mode

### How Theme Switching Works

All color tokens have two values (see the color tokens table above). When the theme switches:
1. `ThemeData` is rebuilt with the opposite color set
2. Flutter automatically repaints all widgets with the new colors
3. The transition is instant (no page reload)

### Guidelines for Dark Mode

- Never use pure black (`#000000`) for backgrounds — use dark grey (`#121212`) for better readability
- Never use pure white (`#FFFFFF`) for text on dark backgrounds — use off-white (`#E0E0E0`) to reduce eye strain
- Status colors must have sufficient contrast in both modes (WCAG AA minimum)
- Images and icons should use tokens, not hardcoded colors, so they adapt

---

## Enterprise Branding Overrides

Enterprise deployments often need to match their company's visual identity. The design system supports this through theme configuration:

### What Can Be Customized

| Customization | How |
|---|---|
| **Brand colors** | Override `colorPrimary`, `colorSecondary` tokens |
| **Logo** | Replace the logo asset in the app shell |
| **Font family** | Override the font family token |
| **Dark mode default** | Set whether dark mode is the default |
| **Login screen background** | Override the login page background color/image |
| **Favicon / app icon** | Replace the icon assets |

### What Should NOT Be Customized

| Element | Why Not |
|---|---|
| Status colors | These are semantic (red = error, green = success). Changing them causes confusion |
| Spacing scale | Changing spacing breaks layouts |
| Typography scale | Changing font sizes breaks responsive layouts |
| Component structure | The design system defines component anatomy; overriding it breaks consistency |

### How Branding Is Applied

Branding overrides are defined in a configuration file (or fetched from a branding endpoint). They are applied at app startup before the first screen renders:

```
1. Load branding configuration
2. Build ThemeData with branded color tokens
3. Provide ThemeData to MaterialApp
4. All widgets automatically use branded colors
```

---

## Design System Components

The design system includes pre-built, styled components used throughout the app:

### AppButton

A standardized button with consistent sizing, colors, and states:

```
Variants:
  Primary:    Solid primary color background, white text
  Secondary:  Outlined with primary color border
  Tertiary:   Text-only, no border or background
  Destructive: Solid red background, white text (for delete/cancel actions)

Sizes:
  Small:  Height 32px, body small text
  Medium: Height 40px, label large text (default)
  Large:  Height 48px, label large text

Options:
  With icon: Icon + text
  Icon only: Just an icon (for toolbars)
  Loading:   Spinner replaces content
```

### AppCard

A surface container with optional header, body, and footer:

```
┌──────────────────────────────────────┐
│ Card Title                   [Action]│  ← optional header
├──────────────────────────────────────┤
│                                      │
│  Card body content                   │  ← required body
│                                      │
├──────────────────────────────────────┤
│ Footer text                  [Link]  │  ← optional footer
└──────────────────────────────────────┘

Properties:
  - elevation: 0 (flat) to 4 (raised)
  - padding: follows spacing tokens
  - border radius: 8px (rounded corners)
```

### AppChip

A small labeled element for tags, filters, and selections:

```
[✕ Pending]  [Shipped]  [+ Add Filter]

Variants:
  Filter:    Removable (✕ button), used in filter bars
  Selection: Toggleable, used in multi-select fields
  Info:      Read-only, used for tags and labels
```

### AppDialog

A modal dialog with consistent structure:

```
┌────────────────────────────────────┐
│ Dialog Title                    [✕]│
├────────────────────────────────────┤
│                                    │
│  Dialog body content               │
│                                    │
├────────────────────────────────────┤
│              [Cancel]  [Confirm]   │
└────────────────────────────────────┘

Properties:
  - width: responsive (90% on phone, 400-600px on desktop)
  - close on backdrop tap: configurable
  - destructive variant: red confirm button
```

---

## Accessibility

The design system enforces accessibility standards:

| Standard | Implementation |
|---|---|
| **Color contrast** | All text meets WCAG AA contrast ratio (4.5:1 for body text, 3:1 for large text) |
| **Focus indicators** | Visible focus rings on all interactive elements for keyboard navigation |
| **Touch targets** | Minimum 48x48px touch target for all interactive elements |
| **Screen reader support** | Semantic widgets used throughout (e.g., `Semantics`, `Tooltip`, `ExcludeSemantics`) |
| **Reduced motion** | Animations respect the OS "reduce motion" setting |
| **Font scaling** | Text sizes respect the OS accessibility font size setting |
