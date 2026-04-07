# ADR 003: Supabase for Database and Auth Hosting

## Status

Accepted

## Context

We need a managed Postgres database with authentication capabilities. The solution should:

- Provide hosted Postgres without managing infrastructure
- Include built-in authentication (email/password at minimum)
- Support Row Level Security for defense-in-depth authorization
- Offer connection pooling compatible with pgx
- Have a generous free tier for development

## Decision

Use Supabase as the managed platform providing:

- **Postgres database** — accessed via pgx through the session pooler (port 5432)
- **Supabase Auth** — handles user registration, login, and JWT issuance on the Angular side
- **Row Level Security (RLS)** — third authorization layer (behind Angular guards and Go middleware)
- **Goose migrations** — manage schema changes in SQL files checked into the repo (not Supabase UI)

The Go API validates Supabase-issued JWTs but does not call Supabase Auth APIs directly. Authentication is a frontend concern; authorization is enforced at every layer.

## Consequences

**Positive:**
- Zero database infrastructure to manage
- Built-in auth eliminates custom password hashing, token issuance, and session management
- RLS provides database-level data isolation as a security backstop
- Session pooler is compatible with pgx's persistent connection model
- Free tier is sufficient for development and early usage

**Negative:**
- Vendor lock-in for auth — migrating away requires rebuilding auth flows
- Supabase's connection pooler has limitations compared to PgBouncer in transaction mode
- Schema changes must be coordinated between Goose migrations and any Supabase dashboard edits

**Risks:**
- If Supabase has downtime, both auth and database are affected. Mitigation: auth tokens are validated locally via JWT signature, so short outages don't block authenticated users
