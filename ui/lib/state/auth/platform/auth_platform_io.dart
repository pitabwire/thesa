import 'dart:async';
import 'dart:io';

import 'package:app_links/app_links.dart';
import 'package:logging/logging.dart';
import 'package:openid_client/openid_client.dart';
import 'package:openid_client/openid_client_io.dart';
import 'package:url_launcher/url_launcher.dart';

import 'auth_platform.dart';

final _log = Logger('AuthPlatformIO');

/// Custom URL scheme for mobile OAuth redirect (RFC 8252 compliant).
const String _customScheme = 'org.stawi.thesa';
const String _customHost = 'sso';

/// Desktop authenticator using loopback HTTP server.
class _DesktopAuthenticator {
  _DesktopAuthenticator({required this.flow, required this.urlLauncher});

  final Flow flow;
  final Future<void> Function(String url) urlLauncher;
  HttpServer? _server;
  bool _cancelled = false;

  static const _successHtml = '''
<!DOCTYPE html>
<html>
<head>
  <meta charset="UTF-8">
  <title>Authentication Complete</title>
  <style>
    body {
      font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
      display: flex; justify-content: center; align-items: center;
      min-height: 100vh; margin: 0;
      background: linear-gradient(135deg, #1a73e8 0%, #6c47d9 100%);
      color: white;
    }
    .container { text-align: center; padding: 2rem;
      background: rgba(255,255,255,0.1); border-radius: 16px; }
    h1 { margin-bottom: 1rem; }
    p { opacity: 0.9; }
  </style>
</head>
<body>
  <div class="container">
    <h1>Authentication Successful</h1>
    <p>You can close this window and return to the app.</p>
  </div>
  <script>setTimeout(function() { window.close(); }, 1500);</script>
</body>
</html>
''';

  Future<Credential> authorize() async {
    final completer = Completer<Credential>();

    try {
      _server = await HttpServer.bind(
        InternetAddress.loopbackIPv4,
        flow.redirectUri.port,
      );

      _server!.listen((request) async {
        if (_cancelled) {
          request.response.statusCode = HttpStatus.gone;
          request.response.write('Authentication cancelled');
          await request.response.close();
          return;
        }

        try {
          final error = request.uri.queryParameters['error'];
          if (error != null) {
            final errorDesc =
                request.uri.queryParameters['error_description'] ?? error;
            request.response.statusCode = HttpStatus.badRequest;
            request.response.write('Authentication error: $errorDesc');
            await request.response.close();
            if (!completer.isCompleted) {
              completer.completeError(Exception('OAuth error: $errorDesc'));
            }
            return;
          }

          final code = request.uri.queryParameters['code'];
          final state = request.uri.queryParameters['state'];

          if (code == null) {
            request.response.statusCode = HttpStatus.badRequest;
            request.response.write('Missing authorization code');
            await request.response.close();
            return;
          }

          request.response.statusCode = HttpStatus.ok;
          request.response.headers.contentType = ContentType.html;
          request.response.write(_successHtml);
          await request.response.close();

          try {
            final credential = await flow.callback({
              'code': code,
              if (state != null) 'state': state,
            });
            if (!completer.isCompleted) {
              completer.complete(credential);
            }
          } catch (e) {
            if (!completer.isCompleted) {
              completer.completeError(e);
            }
          }
        } catch (e) {
          try {
            request.response.statusCode = HttpStatus.internalServerError;
            request.response.write('Internal error');
            await request.response.close();
          } catch (_) {}
          if (!completer.isCompleted) {
            completer.completeError(e);
          }
        }
      });

      final authUri = flow.authenticationUri;
      await urlLauncher(authUri.toString());

      return await completer.future;
    } catch (e) {
      await cancel();
      rethrow;
    }
  }

  Future<void> cancel() async {
    _cancelled = true;
    try {
      await _server?.close(force: true);
      _server = null;
    } catch (_) {}
  }
}

/// Mobile authenticator using custom URL scheme deep links (RFC 8252).
class _MobileAuthenticator {
  _MobileAuthenticator({required this.flow, required this.urlLauncher});

  final Flow flow;
  final Future<void> Function(String url) urlLauncher;
  final AppLinks _appLinks = AppLinks();
  StreamSubscription<Uri>? _linkSubscription;
  bool _cancelled = false;

  Future<Credential> authorize() async {
    final completer = Completer<Credential>();

    try {
      _linkSubscription = _appLinks.uriLinkStream.listen((Uri uri) async {
        if (_cancelled) return;

        if (uri.scheme != _customScheme || uri.host != _customHost) return;

        final error = uri.queryParameters['error'];
        if (error != null) {
          final errorDesc = uri.queryParameters['error_description'] ?? error;
          if (!completer.isCompleted) {
            completer.completeError(Exception('OAuth error: $errorDesc'));
          }
          return;
        }

        final code = uri.queryParameters['code'];
        final state = uri.queryParameters['state'];

        if (code == null) {
          if (!completer.isCompleted) {
            completer.completeError(Exception('Missing authorization code'));
          }
          return;
        }

        try {
          final credential = await flow.callback({
            'code': code,
            if (state != null) 'state': state,
          });
          if (!completer.isCompleted) {
            completer.complete(credential);
          }
        } catch (e) {
          if (!completer.isCompleted) {
            completer.completeError(e);
          }
        }
      });

      final authUri = flow.authenticationUri;
      await urlLauncher(authUri.toString());

      return await completer.future;
    } catch (e) {
      await cancel();
      rethrow;
    }
  }

  Future<void> cancel() async {
    _cancelled = true;
    await _linkSubscription?.cancel();
    _linkSubscription = null;
  }
}

AuthPlatform getAuthPlatform() => AuthPlatformIO();

/// IO (mobile + desktop) implementation of [AuthPlatform].
class AuthPlatformIO implements AuthPlatform {
  static const int _authPort = 5170;
  static const Duration _authTimeout = Duration(minutes: 3);

  Issuer? _issuer;
  Client? _client;
  _DesktopAuthenticator? _desktopAuthenticator;
  _MobileAuthenticator? _mobileAuthenticator;

  @override
  Client? get client => _client;

  bool get _isMobile => Platform.isAndroid || Platform.isIOS;

  Uri _getRedirectUri() {
    if (_isMobile) {
      return Uri.parse('$_customScheme://$_customHost/redirect');
    } else {
      return Uri.parse('http://localhost:$_authPort');
    }
  }

  @override
  Future<void> initialize(String issuerUrl, String clientId) async {
    if (_issuer == null || _client == null) {
      try {
        _issuer = await Issuer.discover(
          Uri.parse(issuerUrl),
        ).timeout(const Duration(seconds: 15));
        _client = Client(_issuer!, clientId);
        _log.fine(
          'OIDC initialized: issuer=$issuerUrl, '
          'platform=${_isMobile ? "mobile" : "desktop"}',
        );
      } catch (e) {
        _log.severe('Failed to discover OIDC issuer: $issuerUrl', e);
        rethrow;
      }
    }
  }

  @override
  Future<TokenResponse?> authenticate(List<String> scopes) async {
    if (_client == null) {
      throw StateError('AuthPlatformIO not initialized');
    }

    await cancelAuthentication();

    final redirectUri = _getRedirectUri();

    Future<void> urlLauncher(String url) async {
      final uri = Uri.parse(url);
      final launched = await launchUrl(
        uri,
        mode: LaunchMode.externalApplication,
      );
      if (!launched) {
        throw Exception('Could not launch authentication URL');
      }
    }

    final flow = Flow.authorizationCodeWithPKCE(_client!)
      ..scopes.addAll(scopes)
      ..redirectUri = redirectUri;

    try {
      Credential credential;

      if (_isMobile) {
        _mobileAuthenticator = _MobileAuthenticator(
          flow: flow,
          urlLauncher: urlLauncher,
        );
        credential = await _mobileAuthenticator!.authorize().timeout(
          _authTimeout,
          onTimeout: () {
            cancelAuthentication();
            throw TimeoutException(
              'Authentication timed out. Please try again.',
              _authTimeout,
            );
          },
        );
      } else {
        _desktopAuthenticator = _DesktopAuthenticator(
          flow: flow,
          urlLauncher: urlLauncher,
        );
        credential = await _desktopAuthenticator!.authorize().timeout(
          _authTimeout,
          onTimeout: () {
            cancelAuthentication();
            throw TimeoutException(
              'Authentication timed out. Please try again.',
              _authTimeout,
            );
          },
        );
      }

      if (_isMobile) {
        try {
          await closeInAppWebView();
        } catch (_) {}
        await Future<void>.delayed(const Duration(milliseconds: 500));
      }

      // Get token response with retry for network errors
      final tokenResponse = await _retryWithBackoff(
        () async {
          final response = await credential.getTokenResponse();
          if (response.accessToken == null || response.accessToken!.isEmpty) {
            throw Exception(
              'Token exchange failed: No access token in response',
            );
          }
          return response;
        },
        maxAttempts: 3,
        initialDelay: const Duration(seconds: 1),
      );

      await cancelAuthentication();
      return tokenResponse;
    } catch (e) {
      await cancelAuthentication();
      rethrow;
    }
  }

  @override
  Future<void> cancelAuthentication() async {
    if (_mobileAuthenticator != null) {
      try {
        await _mobileAuthenticator!.cancel();
      } catch (_) {}
      _mobileAuthenticator = null;
    }
    if (_desktopAuthenticator != null) {
      try {
        await _desktopAuthenticator!.cancel();
      } catch (_) {}
      _desktopAuthenticator = null;
    }
  }

  @override
  Future<TokenResponse?> getRedirectResult() async => null;

  Future<T> _retryWithBackoff<T>(
    Future<T> Function() operation, {
    required int maxAttempts,
    required Duration initialDelay,
  }) async {
    var delay = initialDelay;
    for (var attempt = 1; attempt <= maxAttempts; attempt++) {
      try {
        return await operation().timeout(const Duration(seconds: 30));
      } catch (e) {
        final isNetworkError =
            e.toString().contains('SocketException') ||
            e.toString().contains('Failed host lookup') ||
            e.toString().contains('Connection refused') ||
            e is TimeoutException;
        if (!isNetworkError || attempt == maxAttempts) rethrow;
        _log.warning(
          'Network error on attempt $attempt/$maxAttempts, '
          'retrying in ${delay.inSeconds}s',
        );
        await Future<void>.delayed(delay);
        delay *= 2;
      }
    }
    throw StateError('Retry loop completed without returning');
  }
}
