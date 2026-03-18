/// OIDC authentication service using openid_client with platform-specific flows.
///
/// Handles user authentication using OpenID Connect with:
/// - Authorization Code + PKCE on all platforms
/// - Mobile: deep link callbacks via custom URL scheme
/// - Desktop: loopback HTTP server for callbacks
/// - Web: page redirect with localStorage state
library;

import 'dart:async';
import 'dart:convert';

import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import 'package:logging/logging.dart';
import 'package:openid_client/openid_client.dart';

import '../../core/config/app_config.dart';
import 'platform/auth_platform.dart';
import 'platform/auth_platform_stub.dart'
    if (dart.library.io) 'platform/auth_platform_io.dart'
    if (dart.library.html) 'platform/auth_platform_web.dart';

final _log = Logger('OidcService');

/// Result classification for token refresh operations.
enum TokenRefreshResult { success, permanentError, transientError }

/// Secure storage keys for OIDC tokens.
class AuthStorageKeys {
  AuthStorageKeys._();

  static const accessToken = 'access_token';
  static const refreshToken = 'refresh_token';
  static const idToken = 'id_token';
  static const tokenExpiresAt = 'token_expires_at';
}

/// OIDC authentication service.
///
/// Wraps [openid_client] with platform-specific auth flows, secure
/// token storage, and mutex-protected token refresh.
class OidcService {
  OidcService(this._storage)
    : _issuerUrl = AppConfig.oidcIssuer,
      _clientId = AppConfig.oidcClientId;

  final FlutterSecureStorage _storage;
  final String _issuerUrl;
  final String _clientId;
  final AuthPlatform _platform = getAuthPlatform();

  // Default token lifetime when server doesn't provide expiry.
  static const _defaultTokenLifetime = Duration(hours: 1);

  /// Initialize OIDC issuer discovery and client.
  Future<void> _ensureInitialized() async {
    await _platform.initialize(_issuerUrl, _clientId);
  }

  /// Authenticate user with OIDC provider.
  ///
  /// Returns [TokenResponse] on IO platforms, null on Web (redirect flow).
  Future<TokenResponse?> authenticate() async {
    try {
      _log.info('Starting OIDC authentication');
      await _ensureInitialized();

      final token = await _platform.authenticate(AppConfig.oidcScopes);

      if (token != null) {
        if (token.accessToken == null || token.accessToken!.isEmpty) {
          throw Exception('Authentication failed: No access token received');
        }
        await _saveTokens(token);
        _log.info('User authenticated successfully');
        return token;
      } else {
        // Web redirect flow — tokens saved in getRedirectResult
        _log.info('Authentication initiated (expecting redirect)');
        return null;
      }
    } catch (e, stack) {
      _log.severe('Authentication failed', e, stack);
      rethrow;
    }
  }

  /// Cancel any ongoing authentication flow.
  Future<void> cancelAuthentication() async {
    try {
      await _platform.cancelAuthentication();
    } catch (_) {}
  }

  /// Check for and process redirect result (Web only).
  ///
  /// Returns true if a valid session was recovered from redirect.
  Future<bool> handleRedirectResult() async {
    try {
      await _ensureInitialized();
      final token = await _platform.getRedirectResult();
      if (token != null) {
        _log.info('Recovered session from redirect');
        await _saveTokens(token);
        return true;
      }
      return false;
    } catch (e) {
      _log.warning('Error handling redirect result: $e');
      return false;
    }
  }

  // ---------- Token storage ----------

  Future<void> _saveTokens(TokenResponse token) async {
    await _storage.write(
      key: AuthStorageKeys.accessToken,
      value: token.accessToken,
    );
    await _storage.write(
      key: AuthStorageKeys.refreshToken,
      value: token.refreshToken,
    );
    try {
      await _storage.write(
        key: AuthStorageKeys.idToken,
        value: token.idToken.toCompactSerialization(),
      );
    } catch (_) {
      _log.fine('No ID token in response');
    }

    final expiresAt =
        token.expiresAt ?? DateTime.now().add(_defaultTokenLifetime);
    await _storage.write(
      key: AuthStorageKeys.tokenExpiresAt,
      value: expiresAt.millisecondsSinceEpoch.toString(),
    );
  }

  Future<String?> getAccessToken() async =>
      _storage.read(key: AuthStorageKeys.accessToken);

  Future<String?> getRefreshToken() async =>
      _storage.read(key: AuthStorageKeys.refreshToken);

  Future<String?> getIdToken() async =>
      _storage.read(key: AuthStorageKeys.idToken);

  // ---------- Token expiry ----------

  Future<bool> isTokenExpired({
    Duration buffer = const Duration(minutes: 2),
  }) async {
    final expiresAtStr = await _storage.read(
      key: AuthStorageKeys.tokenExpiresAt,
    );
    if (expiresAtStr == null) return true;

    try {
      final expiresAt = DateTime.fromMillisecondsSinceEpoch(
        int.parse(expiresAtStr),
      );
      return DateTime.now().isAfter(expiresAt.subtract(buffer));
    } catch (_) {
      return true;
    }
  }

  // ---------- Token refresh ----------

  Completer<({TokenRefreshResult result, TokenResponse? token, String? error})>?
  _refreshCompleter;

  /// Refresh the access token using the refresh token.
  ///
  /// Safe to call concurrently — if a refresh is in progress, callers
  /// wait for the existing operation.
  Future<({TokenRefreshResult result, TokenResponse? token, String? error})>
  refreshTokenWithResult() async {
    if (_refreshCompleter != null && !_refreshCompleter!.isCompleted) {
      return _refreshCompleter!.future;
    }

    _refreshCompleter =
        Completer<
          ({TokenRefreshResult result, TokenResponse? token, String? error})
        >();

    try {
      final refreshTokenValue = await getRefreshToken();
      if (refreshTokenValue == null) {
        const r = (
          result: TokenRefreshResult.permanentError,
          token: null as TokenResponse?,
          error: 'No refresh token',
        );
        _refreshCompleter!.complete(r);
        return r;
      }

      try {
        await _ensureInitialized();
      } catch (e) {
        final r = (
          result: TokenRefreshResult.transientError,
          token: null as TokenResponse?,
          error: 'OIDC initialization failed: $e',
        );
        _refreshCompleter!.complete(r);
        return r;
      }

      if (_platform.client == null) {
        const r = (
          result: TokenRefreshResult.transientError,
          token: null as TokenResponse?,
          error: 'Auth client not initialized',
        );
        _refreshCompleter!.complete(r);
        return r;
      }

      final credential = _platform.client!.createCredential(
        accessToken: await getAccessToken(),
        refreshToken: refreshTokenValue,
      );

      final newCredential = await credential
          .getTokenResponse(true)
          .timeout(const Duration(seconds: 30));

      if (newCredential.accessToken == null ||
          newCredential.accessToken!.isEmpty) {
        const r = (
          result: TokenRefreshResult.permanentError,
          token: null as TokenResponse?,
          error: 'Refresh returned empty access token',
        );
        _refreshCompleter!.complete(r);
        return r;
      }

      await _saveTokens(newCredential);

      final r = (
        result: TokenRefreshResult.success,
        token: newCredential,
        error: null as String?,
      );
      _refreshCompleter!.complete(r);
      return r;
    } on TimeoutException {
      const r = (
        result: TokenRefreshResult.transientError,
        token: null as TokenResponse?,
        error: 'Refresh timed out',
      );
      _refreshCompleter!.complete(r);
      return r;
    } catch (e) {
      final errorStr = e.toString().toLowerCase();
      final isPermanent = _isPermanentRefreshError(errorStr);
      final r = (
        result: isPermanent
            ? TokenRefreshResult.permanentError
            : TokenRefreshResult.transientError,
        token: null as TokenResponse?,
        error: e.toString(),
      );
      _refreshCompleter!.complete(r);
      return r;
    } finally {
      if (_refreshCompleter != null && !_refreshCompleter!.isCompleted) {
        _refreshCompleter!.completeError(
          StateError('Token refresh exited without completing'),
        );
      }
      _refreshCompleter = null;
    }
  }

  bool _isPermanentRefreshError(String errorStr) {
    const transientPatterns = [
      'timeout',
      'timed out',
      'connection refused',
      'connection reset',
      'connection closed',
      'no route to host',
      'network is unreachable',
      'host not found',
      'dns',
      'socket',
      'eof',
      'broken pipe',
      'ssl',
      'tls',
      'certificate',
      'handshake',
      '500',
      '502',
      '503',
      '504',
      '429',
      'too many requests',
      'rate limit',
      'temporarily unavailable',
      'service unavailable',
      'try again',
      'retry',
    ];
    for (final p in transientPatterns) {
      if (errorStr.contains(p)) return false;
    }

    const permanentOAuthErrors = [
      'invalid_grant',
      'invalid_client',
      'unauthorized_client',
      'access_denied',
    ];
    for (final e in permanentOAuthErrors) {
      if (errorStr.contains(e)) return true;
    }

    const permanentMessages = [
      'refresh token has been revoked',
      'refresh token was revoked',
      'the refresh token is no longer valid',
      'refresh token is no longer active',
    ];
    for (final m in permanentMessages) {
      if (errorStr.contains(m)) return true;
    }

    return false;
  }

  // ---------- User info ----------

  /// Check if user is authenticated (has access token OR refresh token).
  Future<bool> isAuthenticated() async {
    await handleRedirectResult();
    final accessToken = await getAccessToken();
    if (accessToken != null) return true;
    final refreshTokenValue = await getRefreshToken();
    return refreshTokenValue != null;
  }

  /// Decode user claims from the ID token (JWT payload).
  Future<Map<String, dynamic>?> getUserInfo() async {
    final idToken = await getIdToken();
    if (idToken == null) return null;

    try {
      final parts = idToken.split('.');
      if (parts.length != 3) return null;
      final normalized = base64.normalize(parts[1]);
      final decoded = utf8.decode(base64.decode(normalized));
      return json.decode(decoded) as Map<String, dynamic>;
    } catch (e) {
      _log.warning('Failed to decode ID token', e);
      return null;
    }
  }

  // ---------- Logout ----------

  Future<void> logout() async {
    await _storage.delete(key: AuthStorageKeys.accessToken);
    await _storage.delete(key: AuthStorageKeys.refreshToken);
    await _storage.delete(key: AuthStorageKeys.idToken);
    await _storage.delete(key: AuthStorageKeys.tokenExpiresAt);
  }
}
