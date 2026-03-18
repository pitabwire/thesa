/// Authentication state model for OIDC-based auth.
library;

import 'package:freezed_annotation/freezed_annotation.dart';

part 'auth_state.freezed.dart';

/// Authentication state — no JSON serialization needed
/// since auth state is managed in-memory and tokens stored individually.
@Freezed(toJson: false, fromJson: false)
class AuthState with _$AuthState {
  /// User is logged out
  const factory AuthState.loggedOut() = _LoggedOut;

  /// OIDC login in progress (redirect started)
  const factory AuthState.loggingIn() = _LoggingIn;

  /// User is authenticated via OIDC
  const factory AuthState.loggedIn({
    required String accessToken,
    required String refreshToken,
    String? idToken,
    String? tenantId,
    String? partitionId,
    String? profileId,
    String? email,
    @Default(<String>[]) List<String> roles,
    DateTime? tokenExpiry,
  }) = _LoggedIn;

  /// Authentication error
  const factory AuthState.error({required String message}) = _AuthError;
}
