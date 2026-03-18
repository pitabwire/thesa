/// Authentication provider using OIDC Authorization Code + PKCE.
///
/// Authenticates via Ory Hydra using the openid_client package with
/// platform-specific flows (mobile deep links, desktop loopback, web redirect).
library;

import 'package:logging/logging.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

import '../core/dependencies_provider.dart';
import 'auth_state.dart';
import 'oidc_service.dart';

part 'auth_provider.g.dart';

final _logger = Logger('AuthProvider');

/// OIDC service provider (singleton).
@Riverpod(keepAlive: true)
OidcService oidcService(Ref ref) {
  final secureStorage = ref.watch(secureStorageProvider);
  return OidcService(secureStorage);
}

/// Auth provider — always alive, manages OIDC login/logout lifecycle.
@Riverpod(keepAlive: true)
class Auth extends _$Auth {
  @override
  Future<AuthState> build() async {
    final oidc = ref.read(oidcServiceProvider);

    // Check for existing session (including web redirect result)
    final isLoggedIn = await oidc.isAuthenticated();

    if (isLoggedIn) {
      _logger.info('Found existing session, restoring');

      final accessToken = await oidc.getAccessToken();
      final refreshToken = await oidc.getRefreshToken();
      final idToken = await oidc.getIdToken();

      if (accessToken != null && refreshToken != null) {
        final claims = await oidc.getUserInfo() ?? {};
        return AuthState.loggedIn(
          accessToken: accessToken,
          refreshToken: refreshToken,
          idToken: idToken,
          tenantId: claims['tenant_id'] as String?,
          partitionId: claims['partition_id'] as String?,
          profileId: claims['sub'] as String?,
          email: claims['email'] as String?,
          roles: _extractRoles(claims),
        );
      }
    }

    return const AuthState.loggedOut();
  }

  /// Initiate OIDC login via system browser.
  Future<void> login() async {
    state = const AsyncValue.data(AuthState.loggingIn());

    try {
      _logger.info('Initiating OIDC login');

      final oidc = ref.read(oidcServiceProvider);
      final token = await oidc.authenticate();

      if (token == null) {
        // Web redirect flow — page will reload and build() handles recovery
        return;
      }

      // IO platforms get the token immediately
      final claims = await oidc.getUserInfo() ?? {};

      state = AsyncValue.data(
        AuthState.loggedIn(
          accessToken: token.accessToken!,
          refreshToken: token.refreshToken!,
          idToken: await oidc.getIdToken(),
          tenantId: claims['tenant_id'] as String?,
          partitionId: claims['partition_id'] as String?,
          profileId: claims['sub'] as String?,
          email: claims['email'] as String?,
          roles: _extractRoles(claims),
        ),
      );

      _logger.info('OIDC login successful');
    } catch (e, stack) {
      _logger.severe('OIDC login failed', e, stack);
      state = AsyncValue.data(AuthState.error(message: 'Login failed: $e'));
    }
  }

  /// Refresh tokens via OIDC token endpoint.
  Future<bool> refreshTokens() async {
    try {
      final oidc = ref.read(oidcServiceProvider);
      final result = await oidc.refreshTokenWithResult();

      if (result.result != TokenRefreshResult.success) {
        if (result.result == TokenRefreshResult.permanentError) {
          _logger.warning('Permanent refresh failure, logging out');
          await logout();
        }
        return false;
      }

      // Update state with new tokens and claims
      final claims = await oidc.getUserInfo() ?? {};
      state = AsyncValue.data(
        AuthState.loggedIn(
          accessToken: result.token!.accessToken!,
          refreshToken: result.token!.refreshToken!,
          idToken: await oidc.getIdToken(),
          tenantId: claims['tenant_id'] as String?,
          partitionId: claims['partition_id'] as String?,
          profileId: claims['sub'] as String?,
          email: claims['email'] as String?,
          roles: _extractRoles(claims),
        ),
      );

      _logger.info('Token refresh successful');
      return true;
    } catch (e, stack) {
      _logger.severe('Token refresh failed', e, stack);
      return false;
    }
  }

  /// Logout — clear tokens and cache.
  Future<void> logout() async {
    _logger.info('Logging out');

    try {
      final oidc = ref.read(oidcServiceProvider);
      final cacheCoordinator = await ref.read(cacheCoordinatorProvider.future);

      await oidc.logout();
      await cacheCoordinator.clearAll();

      state = const AsyncValue.data(AuthState.loggedOut());
      _logger.info('Logout successful');
    } catch (e, stack) {
      _logger.severe('Logout failed', e, stack);
      // Still mark as logged out even if cleanup fails
      try {
        final oidc = ref.read(oidcServiceProvider);
        await oidc.logout();
      } catch (_) {}
      state = const AsyncValue.data(AuthState.loggedOut());
    }
  }

  List<String> _extractRoles(Map<String, dynamic> claims) {
    final roles = claims['roles'];
    if (roles is List) {
      return roles.map((r) => r.toString()).toList();
    }
    return [];
  }
}
