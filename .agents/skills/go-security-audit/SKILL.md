---
name: go-security-audit
description: Web security auditing guidelines for Go HTTP servers, JWT cookies, and template rendering.
---

# Go Security Audit Skill

This skill defines security standards for Go web applications utilizing `golang-jwt` and server-rendered HTML templates.

## Security Controls

1. **HTTP Security Headers**:
   - `X-Frame-Options: DENY`
   - `X-Content-Type-Options: nosniff`
   - `X-XSS-Protection: 1; mode=block`
   - `Referrer-Policy: strict-origin-when-cross-origin`

2. **Cookie & JWT Management**:
   - Cookies must set `HttpOnly: true`, `SameSite: http.SameSiteLaxMode` (or `SameSiteStrictMode`), and `Path: "/"`.
   - Validate token expiration (`exp`), signing method, and HMAC secret integrity.

3. **XSS Prevention**:
   - Always auto-escape raw variables using `html/template` or `template.HTMLEscapeString`.
   - Never use `template.HTML()` on un-sanitized user inputs.

4. **Database Injection Protection**:
   - Always use parameterized queries (`SELECT ... WHERE id = ?`). Never concatenate string variables into SQL statements.
