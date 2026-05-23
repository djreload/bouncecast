# BounceCast Auth Unification Audit

Date: 2026-05-23

## What Existed Before This Pass

BounceCast inherited the original Owncast admin authentication model and added
public account and Studio login flows on top of it.

### Owncast Admin Auth

- Browser/admin pages under `/admin/*` were protected by `RequireAdminAuth`.
- Admin API routes under `/api/admin/*` accepted HTTP Basic auth with username
  `admin` and the configured Owncast admin password.
- BounceCast also added signed admin cookies:
  - `bouncecast_admin_session`
  - `bouncecast_admin_identity`
- The original Owncast admin password maps to owner-level BounceCast access for
  compatibility.

### BounceCast Public Accounts

- Public account registration and login lived under `/api/bouncecast/account/*`.
- Account auth used Owncast chat `user_access_tokens`.
- Frontend code stored the token in localStorage as `accessToken`.
- Account Hub, profile updates, notification preferences, Stars wallet, and
  schedule reminders used that token.

### BounceCast Studio

- Studio login lived under `/api/bouncecast/studio/login`.
- Studio used its own bearer token stored in `bouncecast_streamer_sessions`.
- Frontend code stored that token in localStorage as `bouncecastStudioToken`.
- DJ-role public accounts could already provision active Studio accounts.

### User-Facing Duplication

- `/login` had separate Admin and DJ/streamer tabs.
- `/account` had another public account login/register form.
- `/studio` had another DJ login/register form.
- The public user dropdown opened account registration/login in a modal.

### External Chat Auth

- Owncast IndieAuth and Fediverse chat auth endpoints and modals still exist.
- These are kept as chat identity/linking flows, not as primary site dashboard
  login entry points.

## Final Browser Login Model

`/login` is now the single visible login entry point for BounceCast.

Successful login can issue:

- normal account access token for chat/account/Stars features;
- admin cookies for owner/admin roles;
- Studio bearer token for DJ-capable accounts.

Default redirects:

- owner/admin -> `/admin/`
- DJ -> `/studio`
- standard user -> `/account`

Protected admin pages redirect unauthenticated browsers to
`/login?next=/admin/...` instead of showing the Basic Auth prompt. Admin API
routes still use the original Owncast-compatible Basic/cookie auth behavior.

## Compatibility Kept

- The original Owncast `admin` username and configured admin password remain
  valid.
- Existing `/api/admin/*` clients can continue using HTTP Basic auth.
- Existing `/api/bouncecast/account/login`, `/api/bouncecast/admin/login`, and
  `/api/bouncecast/studio/login` remain available for compatibility.
- IndieAuth and Fediverse chat identity flows remain available.

## Centralized Permission Direction

The product roles are:

- owner
- admin
- moderator
- dj
- user

The original Owncast admin password is mapped to the `owncast-admin` account
identity with owner-level access and the compatibility scopes needed to use the
site as a full operator.

## Role Hardening Added After Unification

- Signed BounceCast admin cookies are now rechecked against the stored account
  role on each admin API request. If an admin account is disabled or its admin
  scope is removed, existing cookies stop authorizing protected admin APIs.
- Older generated Owncast-compatible admin handlers now use role-aware wrappers:
  - owner/admin for day-to-day stream, chat, viewer, logs, schedule, streamer,
    and moderation operations;
  - owner-only for secrets, integrations, access tokens, webhooks, upgrade
    actions, admin password changes, stream keys, custom JavaScript, external
    actions, server binding/storage settings, notification secrets, and similar
    high-risk configuration.
- The admin sidebar reads `/api/admin/bouncecast/session` and hides owner-only
  sections from non-owner admin sessions.

## Follow-Up Risks

- Some older Owncast-compatible admin API handlers still rely on Basic auth as
  an owner-level compatibility path. Do not remove that without an API
  compatibility plan.
- Moderator-specific admin pages are still not exposed; moderator chat actions
  continue to use the existing Owncast moderation token flow.
- Studio still uses a bearer token internally because the existing Studio API is
  built around that model. The unified login now issues it automatically where
  permitted.
- IndieAuth/Fediverse chat auth should remain separate unless a later migration
  explicitly links those identities to BounceCast accounts.
