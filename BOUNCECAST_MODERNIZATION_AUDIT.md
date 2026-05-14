# BounceCast Modernization Audit

This audit captures the first pass at updating BounceCast toward current engineering practices while preserving Owncast compatibility.

## Current Stack

- Backend: Go 1.26 module, chi router, SQLite migrations, RTMP/HLS pipeline, generated OpenAPI server glue, Docker/Earthly build flow.
- Frontend: Next.js 14 pages router, React 18, Ant Design 4, TypeScript with `strict: false`, Jest, Storybook, next-pwa static export.
- Runtime checked locally: Go 1.26.3, Node 24.14.1, npm 11.11.0.
- CI already targets Go 1.26.3 and Node 24.12.0.

## Safe Updates Completed

- Hardened BounceCast SMTP delivery with email address parsing, header sanitization, and a bounded SMTP network dial timeout.
- Added validation for BounceCast streamer handles and optional account email addresses.
- Added channel-specific validation for BounceCast notification subscribers:
  - email destinations must parse as email addresses
  - webhook destinations must be HTTP or HTTPS URLs
  - push destinations must be browser push subscription JSON
- Added schedule validation so set end times must be after start times.
- Updated same-major Go dependencies for router, storage, markdown, system metrics, and `golang.org/x/*` packages.
- Updated frontend package-lock dependencies within the current Next 14 / React 18 architecture and ran `npm audit fix` without force.

## Dependency Findings

`npm outdated` shows several major upgrades are available, but most are migration-level changes:

- Next.js 14.2.35 -> 16.2.6
- React 18.3.1 -> 19.2.6
- Ant Design 4.24.16 -> 6.3.7
- ESLint 8.57.1 -> 10.3.0
- Storybook 9.1.x -> 10.3.x
- TypeScript 5.9.3 -> 6.0.3

These should be handled as separate branches with visual regression checks, not bundled into livestreaming/backend work.

`go list -m -u all` shows patch/minor updates are available for selected Go dependencies, including `golang.org/x/*`, `go-chi/chi/v5`, AWS SDK modules, and `goldmark`. These are safer than frontend framework major upgrades but still need full build/test coverage because they touch networking, storage, rendering, and generated dependencies.

After the safe frontend package update, `npm audit` still reports 19 vulnerabilities. The remaining fixes require breaking changes:

- Next.js 14 -> 16 for Next/PostCSS/eslint-config-next advisories.
- Storybook 9 -> 10 or a Storybook package reshuffle for Storybook CLI/webpack polyfill advisories.
- Replacing or deeply revisiting `next-pwa`, because npm's suggested fix downgrades it to `2.0.2` and is marked semver-major.

Do not run `npm audit fix --force` on the main BounceCast branch without a dedicated migration pass.

## Recommended Next Modernization Order

1. Keep Go and Node versions pinned consistently across docs, CI, Docker, and local setup.
2. Add focused tests around BounceCast admin validation and notification delivery state changes.
3. Update Go dependencies in small groups:
   - security/runtime packages first, such as `golang.org/x/*`
   - router/middleware packages second
   - storage/cloud packages separately
4. Modernize frontend TypeScript gradually:
   - type new BounceCast pages/components first
   - remove local `any` usage
   - then enable stricter compiler options one at a time
5. Plan frontend framework migrations separately:
   - Ant Design 4 -> 6
   - React 18 -> 19
   - Next.js 14 -> 16
   - ESLint flat config migration
6. Add browser-based smoke checks for admin pages after each UI modernization slice.

## Do Not Batch

Avoid combining any of these in one commit:

- Go dependency upgrades plus frontend framework upgrades
- TypeScript strict-mode changes plus Ant Design upgrades
- RTMP/HLS changes plus Docker/runtime changes
- database key/migration changes plus visible admin UI cleanup

BounceCast still inherits Owncast's single-stream architecture. Multi-streamer dashboard work should keep isolating new BounceCast tables and APIs until a deeper multi-channel streaming architecture is explicitly designed.
