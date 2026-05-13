# BounceCast Roadmap

BounceCast starts as a careful fork of Owncast. The near-term priority is preserving livestreaming, RTMP ingest, HLS playback, chat, admin, Docker, and build behavior while visible branding moves to BounceCast.

## 1. Safe Rebrand Tasks

- Finish visible UI text updates in the public frontend and admin.
- Replace logo, favicon, app icon, and Open Graph assets when final BounceCast artwork exists.
- Keep Owncast attribution in license, README, and compatibility notes.
- Leave Go module paths, import paths, database keys, API routes, binary names, and persisted config names unchanged until each has a migration plan.
- Regenerate static frontend assets from source instead of hand-editing generated bundles.

## 2. Build/Test Hardening

- Make `go test ./...`, `go build ./...`, frontend tests, and frontend build reliable on WSL Debian 13.
- Document any required Go, Node, ffmpeg, gcc, and Docker versions.
- Add a lightweight smoke test for startup, admin availability, public page availability, RTMP port binding, and HLS route availability.
- Keep CI workflows fork-aware before changing publish or release jobs.

## 3. UI/UX Upgrades

- Add a coherent BounceCast visual identity after final assets exist.
- Improve first-run onboarding in the admin.
- Make stream setup, stream key discovery, and OBS instructions easier to scan.
- Improve mobile viewing and chat ergonomics.

## 4. Admin Panel Improvements

- Add clearer status cards for stream health, storage, chat, and federation.
- Add safer forms for high-risk settings with restart warnings.
- Improve backup/restore visibility.
- Add clearer admin audit information for moderation and config changes.

## 5. Chat Improvements

- Improve moderation workflows and message search.
- Add better spam controls and rate-limit visibility.
- Improve bot/access-token UX.
- Review anonymous chat and authenticated chat flows.

## 6. Creator Tools

- Add stream profiles or presets for common setups.
- Add better scheduled stream metadata.
- Add simple overlays, panels, and creator links.
- Explore VOD/clip workflows only after storage and retention behavior is designed.

## 7. OBS/RTMP Improvements

- Add clearer OBS setup instructions in admin.
- Surface RTMP URL and stream key status safely.
- Improve error messages for failed ingest and transcoding.
- Consider preset recommendations for bitrate, resolution, latency, and CPU usage.

## 8. Moderation/Roles

- Improve moderator assignment, visibility, and permissions.
- Add role-based access carefully. Owncast is fundamentally single-owner by design, so deeper multi-admin permissions need explicit architecture work.
- Add better ban, timeout, and audit tooling.

## 9. Multi-User/Multi-Channel Future Work

- Warning: Owncast is single-user and single-channel by design.
- Multi-user or multi-channel support is not a branding task. It would require deeper changes to data models, routing, stream state, chat state, auth, admin permissions, storage paths, ActivityPub identity, and migration strategy.
- Start with a design document before touching code.
- Prototype separate channels only in an isolated branch with migration and rollback plans.

## 10. Plugin/Module System

- Define stable extension points before accepting plugins.
- Start with low-risk integrations: webhooks, overlays, external actions, and bot APIs.
- Avoid loading arbitrary server-side plugin code until sandboxing, permissions, and upgrade safety are designed.
- Document plugin compatibility separately from Owncast compatibility.

## 11. Mobile/API Future Work

- Review API stability before encouraging third-party mobile apps.
- Add documented read-only endpoints only where contracts can be maintained.
- Consider mobile-first admin views for stream start/stop status, chat moderation, and health checks.

## 12. Deployment And Docker Improvements

- Keep existing Docker startup behavior intact.
- Add BounceCast documentation for Docker Compose after runtime behavior is verified.
- Review image names, labels, and release workflows separately from visible rebrand.
- Do not rename the binary or internal user/group until release packaging compatibility is understood.

## 13. Security Hardening

- Preserve existing admin login behavior.
- Review default password guidance and first-run warnings.
- Harden CORS, cookies, auth tokens, and webhook permissions.
- Add dependency scanning and regular update review.
- Treat database migrations, persisted config, ActivityPub identity, and stream keys as high-risk areas.
