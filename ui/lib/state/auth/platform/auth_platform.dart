import 'package:openid_client/openid_client.dart';

/// Abstract class for platform-specific authentication logic.
///
/// Each platform (mobile, desktop, web) handles the OAuth2 redirect
/// differently:
/// - Mobile: custom URL scheme deep links
/// - Desktop: loopback HTTP server
/// - Web: page redirect with localStorage state
abstract class AuthPlatform {
  /// Initialize the OIDC issuer and client.
  Future<void> initialize(String issuerUrl, String clientId);

  /// Authenticate the user via Authorization Code + PKCE.
  ///
  /// Returns a [TokenResponse] on success.
  /// On Web, this triggers a redirect and returns null.
  Future<TokenResponse?> authenticate(List<String> scopes);

  /// Check for redirect result (Web only, no-op on other platforms).
  Future<TokenResponse?> getRedirectResult();

  /// Cancel any ongoing authentication flow.
  Future<void> cancelAuthentication() async {}

  /// Get the OIDC client instance (if initialized).
  Client? get client;
}
