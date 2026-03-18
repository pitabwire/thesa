/// Application configuration loaded from environment.
library;

import 'package:flutter_dotenv/flutter_dotenv.dart';

/// App configuration sourced from .env files and dart-defines.
///
/// Redirect URIs are handled by the platform-specific auth layer
/// (mobile: custom URL scheme, desktop: loopback, web: current page URL).
class AppConfig {
  const AppConfig._();

  /// BFF base URL for all API requests.
  static String get bffBaseUrl =>
      const String.fromEnvironment('BFF_BASE_URL').isNotEmpty
      ? const String.fromEnvironment('BFF_BASE_URL')
      : dotenv.get('BFF_BASE_URL', fallback: 'http://localhost:8080');

  /// OIDC issuer URL (Ory Hydra public endpoint).
  static String get oidcIssuer =>
      const String.fromEnvironment('OIDC_ISSUER').isNotEmpty
      ? const String.fromEnvironment('OIDC_ISSUER')
      : dotenv.get('OIDC_ISSUER', fallback: 'https://oauth2.stawi.org');

  /// OIDC client ID for this application (public client).
  static String get oidcClientId =>
      const String.fromEnvironment(
        'OIDC_CLIENT_ID',
      ).isNotEmpty
      ? const String.fromEnvironment('OIDC_CLIENT_ID')
      : dotenv.get('OIDC_CLIENT_ID', fallback: 'thesa-ui');

  /// OIDC scopes to request.
  static List<String> get oidcScopes {
    final raw =
        const String.fromEnvironment('OIDC_SCOPES').isNotEmpty
        ? const String.fromEnvironment('OIDC_SCOPES')
        : dotenv.get(
            'OIDC_SCOPES',
            fallback: 'openid,profile,contact,offline_access',
          );
    return raw.split(',').map((s) => s.trim()).toList();
  }
}
