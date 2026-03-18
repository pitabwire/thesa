# Thesa UI

> A generic, dynamic Flutter frontend that renders enterprise admin interfaces entirely from backend (BFF) descriptors.

## Overview

Thesa UI is an enterprise capability-driven UI runtime that contains **zero hardcoded domain knowledge**. Every screen, form, table, action, and workflow is built at runtime from server-provided contracts.

### Key Features

- ✅ **Dynamic UI** - Pages rendered from BFF JSON descriptors
- ✅ **Offline-First** - Drift (SQLite) cache with stale-while-revalidate
- ✅ **Reactive State** - Riverpod 3.0 with code generation
- ✅ **Smart Caching** - ETag-based cache validation with TTL
- ✅ **Plugin System** - Domain teams can override generic rendering
- ✅ **Multi-Platform** - Web, desktop (Windows/macOS/Linux), mobile (iOS/Android)

## Architecture

```
┌─────────────────────────────────────────────────────────┐
│  LAYER 1: PRESENTATION                                   │
│  "What the user sees"                                    │
├─────────────────────────────────────────────────────────┤
│  LAYER 2: UI ENGINE                                      │
│  "How BFF descriptions become widgets"                   │
├─────────────────────────────────────────────────────────┤
│  LAYER 3: STATE & CAPABILITY                             │
│  "What data the app is currently holding"                │
├─────────────────────────────────────────────────────────┤
│  LAYER 4: CACHE                                          │
│  "What data is saved on the device"                      │
├─────────────────────────────────────────────────────────┤
│  LAYER 5: NETWORKING                                     │
│  "How the app talks to the server"                       │
├─────────────────────────────────────────────────────────┤
│  LAYER 6: PLUGINS                                        │
│  "Custom extensions for specific domains"                │
└─────────────────────────────────────────────────────────┘
```

## Tech Stack

| Layer | Technology | Purpose |
|-------|-----------|---------|
| **Framework** | Flutter 3.5+ | Cross-platform UI framework |
| **State** | Riverpod 3.0 | Reactive state management with code generation |
| **Cache** | Drift + SQLite | Offline-first persistent storage |
| **Routing** | go_router | URL-based navigation with deep linking |
| **Networking** | dio + retrofit | HTTP client with interceptor chain |
| **Models** | freezed + json_serializable | Immutable data classes with JSON parsing |
| **Forms** | reactive_forms | Dynamic form engine with validation |
| **Tables** | data_table_2 | High-performance data tables |

## Prerequisites

- **Flutter SDK**: 3.5.0 or higher
- **Dart SDK**: 3.5.0 or higher
- **IDE**: VS Code or Android Studio (recommended)

## Getting Started

### 1. Clone the Repository

```bash
git clone <repository-url>
cd thesa-ui
```

### 2. Install Dependencies

```bash
flutter pub get
```

### 3. Generate Code

Run code generation for Riverpod, Drift, Freezed, and JSON serialization:

```bash
dart run build_runner build --delete-conflicting-outputs
```

For continuous code generation during development:

```bash
dart run build_runner watch --delete-conflicting-outputs
```

### 4. Run the Application

```bash
# Web
flutter run -d chrome

# Desktop
flutter run -d macos    # or windows, linux

# Mobile
flutter run -d ios      # or android
```

## Project Structure

```
lib/
├── main.dart                 # Entry point
├── app/                      # App shell, routing, navigation
├── core/                     # Shared models, errors, constants
├── networking/               # BFF client, interceptors
├── cache/                    # Drift database, DAOs
├── state/                    # Riverpod providers
├── ui_engine/                # Dynamic rendering engine
├── plugins/                  # Domain-specific extensions
├── design_system/            # Tokens, theme, styled components
├── telemetry/                # Monitoring and logging
└── shared_widgets/           # Reusable UI components
```

## Documentation

Comprehensive documentation is available in the [`docs/`](./docs/) directory:

- **[Architecture Overview](./docs/README.md)** - Start here
- **[Executive Summary](./docs/01-executive-summary.md)** - What and why
- **[Technology Choices](./docs/02-technology-choices.md)** - Every library explained
- **[Layered Architecture](./docs/03-layered-architecture.md)** - The six layers
- **[Folder Structure](./docs/04-folder-structure.md)** - Where everything goes

See the [full documentation index](./docs/README.md) for all 23 sections.

## Development

### Code Generation

This project uses heavy code generation. Always run the build runner after:

- Adding/modifying `@riverpod` annotated providers
- Adding/modifying `@freezed` models
- Adding/modifying Drift database tables
- Adding/modifying `@JsonSerializable` classes

```bash
dart run build_runner build --delete-conflicting-outputs
```

### Linting

The project uses strict linting rules. Check for issues:

```bash
flutter analyze
```

Run custom lints (Riverpod):

```bash
dart run custom_lint
```

### Testing

```bash
# Run all tests
flutter test

# Run with coverage
flutter test --coverage

# Run specific test file
flutter test test/unit/core/models/capability_test.dart
```

## Building for Production

### Web

```bash
flutter build web --release
```

### Desktop

```bash
flutter build macos --release   # or windows, linux
```

### Mobile

```bash
flutter build apk --release     # Android
flutter build ios --release     # iOS
```

## Contributing

This project follows strict architectural principles:

1. **Never** let the Presentation layer directly access Cache or Networking
2. **Always** route data through the State layer
3. **Use** code generation for models, providers, and database code
4. **Wrap** all dynamic components with `ErrorBoundary`
5. **Cache-first** - render from cache, refresh in background

See [CONTRIBUTING.md](./CONTRIBUTING.md) for detailed guidelines (coming soon).

## License

[License information to be added]

## Architecture Documents

For in-depth understanding, read these documents in order:

1. [Executive Summary](./docs/01-executive-summary.md)
2. [Technology Choices](./docs/02-technology-choices.md)
3. [Layered Architecture](./docs/03-layered-architecture.md)
4. [Folder Structure](./docs/04-folder-structure.md)
5. [Offline-First Cache](./docs/05-offline-first-cache.md)
6. [State Architecture](./docs/06-state-architecture.md)
7. [Dynamic UI Engine](./docs/07-dynamic-ui-engine.md)

Full index: [docs/README.md](./docs/README.md)
