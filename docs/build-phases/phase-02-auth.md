# Phase 2: Auth Foundation — Build Summary

**Date:** 2026-04-06-07
**Duration:** Phase 2 of 12
**Branch:** main

---

## What Was Built

Phase 2 established the security foundation: Supabase Auth integration on the frontend, JWT validation middleware on the backend, and protected routes with RBAC infrastructure.

### Frontend (Angular)

| Component | Description |
|-----------|-------------|
| `AuthService` (`core/auth.service.ts`) | Supabase JS client wrapper; manages auth state as signals (`user()`, `isLoggedIn()`); handles login, signup, logout, token refresh |
| `Auth Guard` (`core/auth.guard.ts`) | Route guard (`authGuard()`, `adminGuard()`); protects user routes, prevents access without JWT |
| `Auth Interceptor` (`core/auth.interceptor.ts`) | HTTP interceptor; attaches Bearer token to all API requests |
| `Login Component` (`features/auth/login.component.ts`) | Email/password login form via Supabase client; redirects to dashboard on success |
| `Register Component` (`features/auth/register.component.ts`) | Email/password signup form via Supabase client; auto-logs in on success |
| `Token Refresh Logic` | Access token stored in memory-only signal; refresh token in HTTP-only cookie via Supabase session rotation |

**Key Decisions:**
- Access token in memory only (no localStorage) per security rules
- Refresh token managed by Supabase in HTTP-only, Secure, SameSite=Strict cookie
- JWT validation on every route transition via guard
- Admin routes use `canMatch` to prevent code splitting for non-admin users

### Backend (Go)

| Component | Description |
|-----------|-------------|
| `middleware/auth.go` | JWT validation middleware; extracts and verifies Supabase RS256 tokens; sets `user_id` in context |
| `middleware/rbac.go` | Role-based access control; checks user role from profiles table; returns 403 for unauthorized |
| `handler/profile.go` | `/api/v1/profile` — GET current user profile, PUT to update display_name |
| `model/Profile` | User profile domain type; ID (UUID), email, role (user/admin), display_name, timestamps |
| `repository/profile.go` | Database queries for profile CRUD; uses parameterized pgx queries |
| `config/Config` | `JWTSecret` loaded from environment; uses Viper for override capability |
| `health.go` handler | Unauthenticated health check for monitoring |

**Key Decisions:**
- Three-layer RBAC enforcement: Angular guards (UX) → Go middleware (API) → Supabase RLS (database) as defense-in-depth
- JWT secret never logged; sanitized from error messages
- User isolation at database level via RLS policies
- Minimal JWT claims: user_id, role, expiration only

### Database

**New Migration: `00001_create_profiles.sql`**
- `profiles` table: id, email, role, display_name, created_at, updated_at
- Extends Supabase `auth.users` without duplicating auth fields
- Role enum: 'user' | 'admin'
- RLS policy: users can only read/update their own row
- Index on user_id for fast lookups

---

## Architecture Decisions Applied

- **ADR 001** — Angular signals used for auth state (`user()`, `isLoggedIn()`)
- **ADR 002** — Go middleware layer enforces JWT validation
- **ADR 003** — Supabase Auth for passwordless future capability + managed tokens
- **ADR 011** — Sentinel errors (`ErrUnauthorized`) wrapped in middleware, mapped to 401

---

## Test Coverage Added

| Test file | Tests | Notes |
|-----------|-------|-------|
| `auth.service.spec.ts` | 8 | TDD; login, signup, logout, token refresh |
| `auth.guard.spec.ts` | 5 | Guard permits/denies based on login state |
| `auth.interceptor.spec.ts` | 3 | Token attachment to requests |
| `middleware/auth_test.go` | 7 | Valid token, expired token, malformed, missing |
| `middleware/rbac_test.go` | 4 | User vs admin role enforcement |
| `handler/profile_test.go` | 6 | Profile CRUD, user isolation |

**Total new tests: 33** across frontend and backend.

---

## Pre-requisites Verified

- Supabase project created with:
  - Auth enabled (Email/Password provider)
  - Postgres database accessible
  - Service role key available for migrations
  - JWT secret exposed for Go validation

---

## How to Verify

### Frontend

```bash
cd frontend && ng serve
# Navigate to http://localhost:4200/login
# Create an account with test@example.com / password
# Should redirect to dashboard (after API is running)
```

### Backend

```bash
cd backend && make dev

# Without token — should fail
curl http://localhost:8080/api/v1/profile

# With valid JWT (from Supabase client)
curl -H "Authorization: Bearer $SUPABASE_TOKEN" \
  http://localhost:8080/api/v1/profile

# With invalid token
curl -H "Authorization: Bearer invalid" \
  http://localhost:8080/api/v1/profile  # 401 Unauthorized
```

### Tests

```bash
cd frontend && ng test --watch=false   # 16/16 auth tests pass ✅
cd backend && make test         # auth middleware tests pass ✅
```

---

## Known Limitations

- No passwordless/OAuth yet (deferred to future phase)
- No email verification (Supabase config can enable this)
- Token rotation happens only on explicit refresh (not on every request)
- No session invalidation on role change (user sees old role until refresh)

---

## Next Phase

Phase 3: Database Schema & API — Portfolios, transactions, watchlists tables; full CRUD API endpoints.
