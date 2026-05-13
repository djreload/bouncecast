# BounceCast Multi-Streamer Architecture

BounceCast currently inherits Owncast's single-channel architecture. The admin surface, stream state, RTMP ingest, HLS output, chat, notifications, and persisted configuration all assume one broadcaster, one channel, and one admin password. Multi-DJ support should be built in phases so existing livestreaming, chat, admin login, RTMP ingest, HLS playback, Docker startup, and upgrades from Owncast keep working.

## Current Single-User Assumptions

- Admin authentication is HTTP Basic Auth with username `admin` and one configured password in `webserver/router/middleware/auth.go`.
- Stream keys are stored as one global `stream_keys` config value in `persistence/configrepository`.
- RTMP ingest accepts only one inbound connection at a time via `_hasInboundRTMPConnection` in `core/rtmp/rtmp.go`.
- Current broadcast state is a single global `_currentBroadcast` in `core/streamState.go`.
- HLS storage paths are global through `config.HLSStoragePath`.
- Chat, viewer stats, webhooks, browser notifications, and Fediverse go-live messaging all represent one channel.
- The existing `users` table is for chat/API identities, not dashboard streamer accounts.

## Product Model

Start with one public BounceCast channel that can be operated by multiple streamer accounts. This gives DJs a dashboard, schedule, and per-DJ stream key without requiring multiple simultaneous channels yet.

Recommended entities:

- `streamer_accounts`: dashboard users with display name, handle, email, password hash, role, status, avatar, and timestamps.
- `streamer_stream_keys`: per-streamer RTMP keys with labels, last-used metadata, enabled state, and rotation history.
- `stream_schedule`: scheduled DJ sets with title, description, streamer id, planned start/end, timezone, visibility, status, and notification policy.
- `go_live_events`: immutable live-session records tying an RTMP connection to a streamer, schedule item, start/end times, and delivery state.
- `notification_subscribers`: email/web push subscribers with consent, verification, and preferences.
- `notification_deliveries`: delivery attempts for email, web push, webhook, browser, and Fediverse announcements.

## Phase Plan

### Phase 1: Dashboard Foundation

- Keep current admin Basic Auth working.
- Add Streamers and Schedule admin screens as non-destructive frontend foundations.
- Document the schema and auth changes before adding migrations.
- Keep global stream state, RTMP ingest, HLS paths, and public APIs unchanged.

### Phase 2: Dashboard Auth

- Use the additive `bouncecast_*` tables introduced in `00002_bouncecast_multi_streamer_foundation.sql`.
- Add session or token auth for dashboard users.
- Preserve the existing admin password as an owner bootstrap login.
- Add roles such as owner, manager, streamer, moderator.
- Do not reuse chat `users` as dashboard identities.

### Phase 3: Per-Streamer Stream Keys

- Add per-streamer stream keys while continuing to support the existing global stream key config.
- Resolve inbound RTMP keys to a streamer account.
- Attach streamer metadata to the live event and admin status response.
- Avoid renaming the existing `stream_keys` config key without a migration.

### Phase 4: Scheduling

- Add schedule CRUD APIs and admin UI.
- Store timezone-aware schedule windows.
- Link an inbound stream to the nearest scheduled set for that streamer.
- Track planned, live, completed, missed, and cancelled statuses.

### Phase 5: Go-Live Notifications

- Add email provider configuration.
- Add web push VAPID keys and subscriber storage.
- Queue go-live notifications after RTMP validation and the existing debounce window.
- Keep current browser notification, webhook, and Fediverse behavior working.
- Add per-streamer and per-schedule notification templates.

### Phase 6: Multi-Channel Future

True simultaneous multi-channel streaming is deeper than multi-DJ operation. It requires channel-scoped RTMP connections, transcoder instances, HLS storage paths, chat rooms, viewer stats, webhooks, public pages, and moderation state. This should be treated as a separate architecture project after the single-channel multi-DJ workflow is stable.

## Risk Boundaries

- Do not rename Go module/import paths in this phase.
- Do not rename persisted config keys such as `stream_keys`.
- Do not change existing API routes until compatibility wrappers are designed.
- Do not split HLS storage or stream state until streamer-scoped go-live metadata is working.
- Do not remove Owncast attribution or license files.
- Do not replace Basic Auth until an owner bootstrap and recovery path exists.

## Immediate Code Targets

- `web/components/admin/MainLayout.tsx`: admin navigation and layout shell.
- `web/pages/admin/streamers.tsx`: streamer roster dashboard foundation.
- `web/pages/admin/schedule.tsx`: schedule and go-live notification dashboard foundation.
- `web/public/styles/admin/bouncecast-studio.css`: BounceCast admin polish layer.
- `webserver/router/middleware/auth.go`: future session/role auth boundary.
- `core/rtmp/rtmp.go`: future stream-key-to-streamer resolution point.
- `core/streamState.go`: future go-live event and notification ownership point.
- `persistence/migrations`: future database tables.
- `db/query.sql`: future sqlc queries for dashboard accounts, schedules, and notifications.
