# 14. Authentication and Session Management

## What Is Authentication?

Authentication is the process of verifying who a user is. When someone opens Thesa UI, the app must confirm their identity before showing any data. This protects sensitive business information from unauthorized access.

Thesa UI uses **token-based authentication**, which is the standard for modern web and mobile applications.

---

## How Token-Based Authentication Works

### The Basics

Instead of sending a username and password with every request, the app uses **tokens** — short-lived digital keys that prove the user's identity.

1. **User logs in**: Sends username + password to the BFF
2. **BFF validates**: Checks credentials against the identity provider
3. **BFF responds**: Sends back two tokens:
   - **Access token**: Short-lived (e.g., 15-60 minutes). Used for every API request.
   - **Refresh token**: Long-lived (e.g., 7-30 days). Used to get a new access token when the old one expires.
4. **App stores tokens**: Saved in encrypted storage on the device
5. **App makes requests**: Attaches the access token to every API call
6. **Token expires**: App uses the refresh token to get a new access token — invisibly to the user

### Why Two Tokens?

- The **access token** is sent with every request. If it were stolen (by a network attacker), the attacker could only use it for a short time before it expires.
- The **refresh token** is sent rarely (only to get new access tokens). It has a longer life but is exposed less often.

This is a security tradeoff: frequent short-lived tokens limit the damage window if a token is compromised.

---

## The Complete Authentication Flow

### Login Flow

```
User opens the app
        │
        ▼
authProvider checks: Is there a stored token?
        │
        ├── NO stored token → Show login screen
        │       │
        │       ▼
        │   User enters email and password
        │       │
        │       ▼
        │   App sends: POST /auth/login { email, password }
        │       │
        │       ├── BFF responds: 200 OK { accessToken, refreshToken, expiresIn }
        │       │       │
        │       │       ▼
        │       │   Store tokens in flutter_secure_storage
        │       │       │
        │       │       ▼
        │       │   authProvider state = Authenticated
        │       │       │
        │       │       ▼
        │       │   Load capabilities, navigation, permissions
        │       │       │
        │       │       ▼
        │       │   Navigate to dashboard (or returnUrl if redirected from another page)
        │       │
        │       └── BFF responds: 401 Unauthorized
        │               │
        │               ▼
        │           Show error: "Invalid email or password"
        │           Stay on login screen
        │
        └── YES stored token → Check if access token is expired
                │
                ├── Not expired → authProvider state = Authenticated → Load app
                │
                └── Expired → Attempt refresh (see below)
```

### Token Refresh Flow

```
Access token is expired (or a request returned 401)
        │
        ▼
AuthInterceptor initiates refresh:
  POST /auth/refresh { refreshToken: "stored_refresh_token" }
        │
        ├── BFF responds: 200 OK { accessToken, refreshToken, expiresIn }
        │       │
        │       ▼
        │   Store new tokens in flutter_secure_storage
        │   Replay the original failed request with the new access token
        │   User sees no interruption
        │
        └── BFF responds: 401 Unauthorized (refresh token also expired or revoked)
                │
                ▼
            Clear all stored tokens
            Clear all cached data (permissions, sessions)
            authProvider state = Unauthenticated
            Navigate to login screen with message: "Your session has expired. Please log in again."
```

### Logout Flow

```
User clicks "Log Out"
        │
        ▼
App sends: POST /auth/logout { refreshToken: "stored_refresh_token" }
  (This tells the BFF to invalidate the refresh token server-side)
        │
        ▼
Clear stored tokens from flutter_secure_storage
        │
        ▼
Clear all caches:
  - Permission cache → deleted
  - UI decision cache → deleted
  - Workflow state → preserved (user might log back in and resume)
  - Navigation cache → preserved (structure is not user-specific)
  - Page cache → preserved (structure is not user-specific)
        │
        ▼
authProvider state = Unauthenticated
        │
        ▼
Navigate to login screen
```

---

## Token Storage Security

### Where Tokens Are Stored

Tokens are stored using `flutter_secure_storage`, which uses:

| Platform | Storage Mechanism |
|---|---|
| **iOS** | Keychain Services (hardware-encrypted, persists across app reinstalls unless deleted) |
| **Android** | Android Keystore + EncryptedSharedPreferences (encrypted at rest) |
| **Web** | `sessionStorage` or an encrypted cookie (cleared when the browser is closed, preventing long-term exposure) |
| **macOS** | Keychain Services |
| **Windows** | Windows Credential Manager |
| **Linux** | libsecret (GNOME Keyring / KDE Wallet) |

### Why Not Regular Storage?

Regular storage (like `SharedPreferences` or local files) stores data as **plain text**. Anyone with physical access to the device (or malware) could read the tokens. Secure storage encrypts the data using the operating system's key management, making it significantly harder to extract.

---

## The Auth Interceptor — How Every Request Gets Authenticated

### What It Does

The `AuthInterceptor` is a middleware in the dio HTTP client chain. It automatically:

1. **Adds the access token** to every outgoing request
2. **Handles 401 responses** by attempting token refresh
3. **Queues concurrent requests** during token refresh to prevent race conditions
4. **Redirects to login** if refresh fails

### Step-by-Step: Normal Request

```
Widget calls: GET /ui/pages/orders-list
        │
        ▼
AuthInterceptor.onRequest:
  1. Read access token from secure storage
  2. Add header: Authorization: Bearer eyJhbGciOi...
  3. Pass request to next interceptor
        │
        ▼
Request reaches the BFF server
        │
        ▼
BFF responds: 200 OK with page data
        │
        ▼
AuthInterceptor.onResponse:
  (No action needed for 200 responses)
  Pass response to the caller
```

### Step-by-Step: Token Expired During Request

```
Widget calls: GET /ui/pages/orders-list
        │
        ▼
AuthInterceptor adds expired token: Authorization: Bearer expired_token
        │
        ▼
BFF responds: 401 Unauthorized
        │
        ▼
AuthInterceptor.onError:
  1. Detect 401 response
  2. Acquire refresh lock (prevents multiple concurrent refreshes)
  3. Call: POST /auth/refresh { refreshToken }
  4. BFF responds: 200 OK { newAccessToken }
  5. Store new tokens
  6. Release refresh lock
  7. Retry original request with new token: GET /ui/pages/orders-list
        │
        ▼
BFF responds: 200 OK with page data
        │
        ▼
Widget receives data — no error seen by the user
```

### Handling Concurrent Requests During Refresh

If three requests all fail with 401 at the same time, the interceptor must NOT attempt three separate refreshes. Here is how it handles this:

```
Request A gets 401 → AuthInterceptor acquires lock → starts refresh
Request B gets 401 → AuthInterceptor sees lock is held → waits
Request C gets 401 → AuthInterceptor sees lock is held → waits

Refresh completes → new token stored → lock released

Request A is retried with new token → succeeds
Request B is retried with new token → succeeds
Request C is retried with new token → succeeds
```

Only one refresh request is ever in-flight at a time.

---

## Session Management

### What Is a Session?

A session represents the user's logged-in state. It includes:
- The user's identity (name, email, role)
- Their permissions (what they can see and do)
- Their preferences (language, timezone, theme)

### Session Provider

The `sessionProvider` holds the current session. It depends on `authProvider`:

```
When authProvider changes to Authenticated:
  1. Fetch user profile: GET /ui/session
  2. BFF responds: { userId, name, email, role, permissions, preferences }
  3. Cache the session data in Drift (permission_cache)
  4. sessionProvider state = Session(...)
```

### Permission Reload

Permissions can change while a user is logged in (e.g., an admin grants them a new role). The app must detect and apply these changes.

**Periodic reload**: Every 5 minutes (permission cache TTL), the background refresh checks:
```
GET /ui/session/permissions
If-None-Match: <cached_etag>
```

If permissions changed:
1. New permissions written to cache
2. `sessionProvider` updates
3. All `PermissionGate` widgets rebuild
4. Newly allowed items appear; revoked items disappear

**Forced reload on token refresh**: Every time a new access token is obtained, the permission cache is invalidated and refetched. This catches permission changes that happened while the old token was valid.

### Session Invalidation

The session can be invalidated from the server side (admin revokes access, password changed, account suspended). When this happens:

1. The next API request returns 401
2. Token refresh also returns 401 (the refresh token is revoked)
3. AuthInterceptor clears tokens and session data
4. User is redirected to login with message: "Your session was ended. Please log in again."

---

## Login Screen Design

The login screen is the ONE hardcoded screen in Thesa UI. It cannot be dynamic because the user is not yet authenticated — there is no BFF session to fetch descriptors from.

### Login Screen Layout

```
┌─────────────────────────────────────────────────────────┐
│                                                          │
│                    [App Logo]                             │
│                    App Name                               │
│                                                          │
│              ┌──────────────────────┐                    │
│              │                      │                    │
│   Email      │ alice@company.com    │                    │
│              │                      │                    │
│              └──────────────────────┘                    │
│                                                          │
│              ┌──────────────────────┐                    │
│              │                      │                    │
│   Password   │ ••••••••••           │                    │
│              │                      │                    │
│              └──────────────────────┘                    │
│                                                          │
│              [        Sign In        ]                   │
│                                                          │
│              Forgot password?                            │
│                                                          │
│  ⚠ Invalid email or password         ← error (if any)  │
│                                                          │
└─────────────────────────────────────────────────────────┘
```

### Login Screen Behavior

| Action | Behavior |
|---|---|
| User clicks "Sign In" | Validate fields locally (email format, non-empty password). If valid, submit to BFF. Show loading indicator on button. |
| BFF returns 200 | Store tokens, load session, navigate to dashboard or returnUrl. |
| BFF returns 401 | Show inline error: "Invalid email or password." Clear password field. |
| BFF returns 429 | Show inline error: "Too many attempts. Please wait and try again." |
| BFF unreachable | Show inline error: "Cannot reach the server. Check your network connection." |
| "Forgot password?" | Navigate to password reset flow (BFF-dependent). |

### Branding

The login screen uses the app's design system tokens (colors, fonts, spacing). Enterprise deployments can customize:
- The logo
- The background color or image
- The primary button color
- A custom welcome message

These customizations are defined in the theme configuration, NOT in the BFF (because the BFF is not accessible before login).

---

## Security Best Practices Implemented

| Practice | Implementation |
|---|---|
| **Tokens encrypted at rest** | `flutter_secure_storage` uses OS-level encryption |
| **Short access token lifetime** | 15-60 minutes (configured by BFF) |
| **Automatic refresh** | User never sees token expiration under normal use |
| **Server-side logout** | Refresh token is revoked on the server, not just deleted locally |
| **No credentials in memory** | Password is never stored; only tokens |
| **Redirect loop prevention** | Auth guard tracks the original URL and prevents infinite login redirects |
| **Concurrent refresh protection** | Lock mechanism ensures only one refresh request at a time |
| **Cache clearance on logout** | Permission and session caches are wiped immediately |
