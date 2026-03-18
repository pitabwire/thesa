/// A modern, production-grade Flutter theme inspired by BoundaryML's design system.
///
/// Provides a complete theming solution with dark/light modes, custom color
/// tokens, typography, and pre-styled components.
///
/// ```dart
/// MaterialApp(
///   theme: StawiTheme.light(),
///   darkTheme: StawiTheme.dark(),
///   themeMode: ThemeMode.dark,
/// )
/// ```
library stawi_theme;

export 'src/colors.dart';
export 'src/theme.dart';
export 'src/theme_extensions.dart';
export 'src/typography.dart';
export 'src/button_styles.dart';
export 'src/input_styles.dart';
export 'src/card_styles.dart';
export 'src/component_styles.dart';
export 'src/widgets/stawi_badge.dart';
export 'src/widgets/stawi_card.dart';
export 'src/widgets/stawi_section_header.dart';
export 'src/widgets/stawi_terminal.dart';
export 'src/widgets/stawi_status_dot.dart';
export 'src/widgets/stawi_metric_card.dart';
