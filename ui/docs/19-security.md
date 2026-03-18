# 19. Security Principles

## The Core Security Rule

**The frontend is never trusted.** All security decisions are made by the BFF server. The frontend only renders what the server tells it to render. This single rule eliminates an entire class of security vulnerabilities.

---

## Why This Matters

In traditional apps, developers often implement permission checks on both the frontend and backend:

```
// Traditional (DANGEROUS pattern):
Frontend: if (user.role == "admin") { showDeleteButton() }
Backend:  if (user.role == "admin") { allowDelete() }
```

This creates problems:
1. **Logic mismatch**: The frontend check might use different logic than the backend
2. **Stale permissions**: The frontend might have cached outdated permissions
3. **Bypass risk**: An attacker can modify the frontend code to skip the check
4. **Maintenance burden**: Every permission change must be updated in two places

Thesa UI eliminates this pattern entirely.

---

## Security Architecture

### Principle 1: No Permission Logic Is Duplicated Locally

The frontend NEVER evaluates rules like "can this user do X?" Instead, every BFF response comes pre-filtered:

- Navigation items have `"allowed": true/false"` — the frontend only sees items the user can access
- Page components have `"allowed": true/false"` — disallowed components are never sent (or sent with `false`)
- Actions have `"allowed": true/false"` — the frontend only renders allowed actions
- Workflow steps have `"allowed": true/false"` — the frontend only shows reachable steps

The frontend reads these boolean flags. It does not compute them.

### Principle 2: Hidden Elements Are Absent, Not Disabled

When a user does not have permission for an action, the action button **does not exist** in the widget tree. It is not rendered and then hidden with CSS. It is not greyed out with `enabled: false`. It is simply not created.

**Why this matters**:
- A disabled button tells an attacker "this feature exists, and it might be accessible with the right permissions"
- An absent button reveals nothing
- A developer cannot accidentally make a disabled button clickable

### Principle 3: All Server Responses Are Authoritative

The frontend treats every BFF response as the absolute truth:

- If the BFF says a page has 5 components → the frontend renders 5 components
- If the BFF says an action is allowed → the frontend shows the button
- If the BFF says a schema has these fields → the frontend renders these fields
- The frontend NEVER adds, removes, or modifies what the BFF describes

This means:
- The BFF is the single point of security enforcement
- The frontend is a pure renderer — it cannot introduce security holes through rendering logic
- Security audits only need to verify the BFF, not the frontend

### Principle 4: No Direct Domain Service Calls

The frontend communicates with ONE server: the BFF. It NEVER calls backend domain services directly.

```
CORRECT:
  Flutter App → BFF → Domain Services

WRONG:
  Flutter App → Domain Service A
  Flutter App → Domain Service B
  Flutter App → BFF
```

**Why**:
- The BFF enforces authentication, authorization, and rate limiting for all requests
- Domain services might not have their own authentication (they rely on the BFF layer)
- If the frontend called domain services directly, it would bypass the BFF's security layer

### Principle 5: Client-Side Validation Is a UX Feature, Not a Security Feature

The form engine validates user input locally (required fields, min/max, patterns) to provide instant feedback. But this validation:
- Is purely for user experience (faster error messages)
- Does NOT replace server-side validation
- Is NEVER trusted by the BFF

The BFF re-validates all input. If the BFF rejects data that the frontend accepted, the BFF's decision wins and the error is shown to the user.

---

## Token Security

### Token Storage

| Platform | Mechanism | Security Level |
|---|---|---|
| iOS | Keychain Services | Hardware-encrypted, protected by device passcode |
| Android | Android Keystore + EncryptedSharedPreferences | Hardware-backed encryption on supported devices |
| Web | HttpOnly cookies or sessionStorage | Protected from JavaScript access (HttpOnly) or cleared on browser close |
| macOS | Keychain Services | Same as iOS |
| Windows | Windows Credential Manager | OS-level encrypted storage |
| Linux | libsecret (GNOME Keyring / KDE Wallet) | Desktop environment encrypted storage |

Tokens are NEVER stored in:
- `SharedPreferences` (unencrypted)
- Local files (accessible to other apps)
- Environment variables
- URL parameters
- Browser localStorage (vulnerable to XSS on web)

### Token Lifecycle

```
1. Login → BFF issues access token (15-60 min) + refresh token (7-30 days)
2. Every request → access token sent in Authorization header
3. Access token expires → refresh token used to get new access token
4. Refresh token expires → user must log in again
5. Logout → refresh token revoked on BFF, tokens deleted locally
6. Security event (password change, admin revocation) → all tokens invalidated server-side
```

### Token Refresh Security

- Only ONE refresh request is in-flight at a time (lock mechanism prevents race conditions)
- If refresh fails → all tokens cleared, user redirected to login
- Refresh tokens are rotated: each refresh returns a new refresh token and invalidates the old one
- This limits the window of vulnerability if a refresh token is compromised

---

## Data Security

### Cache Encryption

The Drift (SQLite) database stores cached BFF responses. These may contain sensitive data (user lists, financial summaries, etc.).

**On mobile/desktop**: The SQLite database file is stored in the app's sandboxed directory. On iOS and Android, the OS prevents other apps from accessing this directory. For additional protection, SQLite encryption (via `sqlcipher`) can be enabled.

**On web**: Cache is stored in IndexedDB (for sql.js). This is origin-sandboxed — other websites cannot access it. However, it is accessible to anyone with physical access to the browser. For sensitive deployments, web cache can be disabled (always fetch from network).

### Sensitive Data in Cache

Not all BFF data should be cached equally:

| Data Type | Cache Strategy |
|---|---|
| Navigation tree | Safe to cache (not sensitive) |
| Page descriptors | Safe to cache (layout definitions, not user data) |
| Schemas | Safe to cache (field definitions, not user data) |
| Permissions | Cache with short TTL (5 min). Clear on logout |
| Resource data (table rows) | Cache only if the BFF marks it as cacheable |
| Workflow state | Cache with encryption (may contain form data) |

### Cache Clearing

| Event | What Is Cleared |
|---|---|
| Logout | Permissions, session data, UI decisions |
| Session expiry | Same as logout |
| Different user login | ALL caches (to prevent data leakage between users) |
| App uninstall | OS handles file deletion |

---

## Web-Specific Security

When deployed as a web app, additional security measures apply:

### Content Security Policy (CSP)

The web deployment should serve a strict Content Security Policy header that:
- Only allows scripts from the app's own domain
- Prevents inline scripts (XSS mitigation)
- Only allows connections to the BFF's domain
- Prevents framing by other sites (clickjacking protection)

### HTTPS Only

All communication must use HTTPS. The dio client is configured to reject HTTP connections. The BFF should enforce HTTPS with HSTS headers.

### Cross-Origin Protection

- BFF responses include `Access-Control-Allow-Origin` headers limiting access to the app's domain
- Cookies (if used for tokens) are `SameSite=Strict` and `Secure`

---

## Security Testing

### What to Test

| Test | What It Verifies |
|---|---|
| Components with `allowed: false` are not in the widget tree | Permission enforcement |
| Navigation items with `allowed: false` are absent | Navigation security |
| Token is sent with every request | Auth header presence |
| 401 response triggers refresh, not data exposure | Token refresh flow |
| 403 response shows access denied, not partial data | Permission error handling |
| Logout clears all sensitive caches | Data clearance |
| Different user login clears previous user's data | User isolation |
| Form submission sends data to BFF only (not other endpoints) | Single BFF rule |
| Error messages do not leak sensitive information | Information disclosure |

### Penetration Testing Notes

When security testing Thesa UI, focus on:

1. **Can a user access pages they should not see?** — Try navigating to URLs not in the navigation tree
2. **Can a user trigger actions they should not?** — Modify requests in the browser dev tools
3. **Does logging out truly clear all data?** — Inspect storage after logout
4. **Are tokens properly encrypted at rest?** — Check storage mechanism per platform
5. **Can one user's data leak to another?** — Log in as user A, log out, log in as user B

Since Thesa UI is a renderer that trusts the BFF, most security vulnerabilities would be in the BFF, not the frontend. The frontend's security role is:
- Storing tokens safely
- Clearing data on logout
- Not exposing hidden UI elements
- Sending all requests through the BFF only
