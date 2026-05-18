# BounceCast Changelog

This changelog tracks the BounceCast fork work from the first visible rebrand onward. Keep new user-facing features, backend behavior changes, migrations, deployment notes, and known compatibility risks here as the project evolves.

## Unreleased / Next

- Add rate limiting and abuse protection around public account registration, login, profile updates, Stars checkout, and Stars sending before broader public launch.
- Add profile image upload/storage support; current public profile images are URL/path based.
- Continue live VPS deployment notes as production configuration changes.

## 2026-05-18

### Added

- Added database migration `00009_bouncecast_account_notifications.sql` for public account notification opt-in preferences.
- Added public account notification preference persistence for email, browser push, and Messenger destinations.
- Added public account UI controls for email, browser push, and Facebook Messenger go-live opt-ins.
- Added a public Stars leaderboard, ranked by total Stars sent, so viewers can compete for 1st, 2nd, and 3rd place.
- Added the Stars leaderboard to the admin Stars dashboard.
- Added an admin-only Stars overlay test action for previewing effects without spending Stars or changing wallets.
- Added owner/admin BounceCast account role login support for the admin dashboard while preserving the original Owncast admin credentials.
- Added DJ BounceCast account role login support for Studio, including automatic active Studio account provisioning on successful DJ-role login.

### Changed

- Upgraded Stars overlay effects so sparkle, fireworks, hearts, hype, and DJ drop selections trigger distinct on-screen animations instead of only changing the toast styling.
- Updated linked Studio accounts so if a matching public BounceCast account exists, Studio access now requires that account to keep the `dj` role.

### Fixed

- Fixed Stars overlay queue timing so every sent Stars event can dismiss cleanly and the next overlay plays instead of getting stuck behind the first animation.

## 2026-05-17

### Added

- Added public BounceCast account registration and login APIs.
- Added profile editing for display name and profile image URL.
- Added a public `Register / Login` entry point in the site user menu.
- Added a unified `/login` page for DJ/streamer and admin sign-in:
  - Admin credentials create a browser session and redirect to `/admin/`.
  - DJ/streamer credentials create a Studio session and redirect to `/studio`.
- Added chat avatar rendering from registered user profile images.
- Added admin account management at `/admin/accounts/`.
- Added configurable user role flags for visitor, owner, admin, moderator, and DJ.
- Added database migration `00007_bouncecast_user_accounts.sql` for account fields on existing users.
- Added database migration `00008_bouncecast_single_chat_reaction.sql` to enforce one reaction per user per chat message.
- Added backend tests for account registration, login, profile updates, and admin role updates.

### Changed

- Updated chat user persistence and chat history reads to carry profile image metadata.
- Updated Tenor GIF chat picks so selected GIFs render inline instead of appearing as plain links.
- Updated Stars chat messages and live overlay effects so sent Stars show a clearer message, queued overlay animation, and matching effect styling.
- Updated the generated static web bundle so Docker serves the account and admin role UI.

### Fixed

- Fixed chat reactions so one user can only select one reaction on a message; picking a different reaction switches it, and picking the same reaction clears it.
- Fixed Stars overlay delivery in test and early-start paths so sending Stars cannot panic if the chat server has not started yet.

### Notes

- Visitor is the default derived role and is not stored as a database scope.
- Moderator uses the existing Owncast moderator scope, preserving existing chat moderation behavior.
- Owner/admin/DJ scopes are foundation work and need deeper permission wiring in later passes.

### Deployment

- Deployed the account/roles build to the live Docker/Plesk server for `k-nrg.co.uk`.
- Verified the live database migrated successfully to version 7.
- Verified the public status endpoint, `/admin/accounts/`, and the admin BounceCast users API returned HTTP 200.

## 2026-05-15

### Added

- Added BounceCast chat customization settings:
  - Custom chat background.
  - Chat transparency controls.
  - Tenor GIF API key configuration.
  - Clickable emoji reactions on chat messages.
- Added the BounceCast logo mark and favicon treatment using the heart/disc branding direction.
- Added the BounceCast Stars support system foundation:
  - Admin Stars settings.
  - Configurable star packages.
  - Ledger-based wallet storage.
  - PayPal order/capture/webhook storage paths.
  - Public buy/send Stars UI paths.
  - Stars overlay event plumbing.
  - Admin Stars logs.
- Added Debian 13 plus Plesk live server installation documentation.

### Fixed

- Fixed Stars wallet adjustment locking behavior to protect ledger/balance consistency.

### Notes

- Stars are site-support gifts only. They have no cash value, streamer payout, creator balance, revenue split, or withdrawal dashboard.

## 2026-05-14

### Added

- Added SMTP email notifications for go-live events.
- Added browser push notifications for stream events.
- Added Brevo SMTP relay options.
- Added inactive DJ registration flow so admins can activate DJs.
- Added Studio dashboard auth foundation.
- Added DJ Studio dashboard pages.
- Added Studio schedule management.
- Added Windows CGO test setup notes/tooling support.

### Changed

- Applied a BounceCast rave-theme pass across public/admin UI.
- Unified admin pages toward the Studio visual style for cleaner contrast and consistency.
- Updated project dependencies.
- Hardened the streamer go-live workflow and modernization foundations.

## 2026-05-13

### Added

- Completed the initial safe visible BounceCast rebrand while preserving Owncast internals.
- Added repository inspection, branding asset documentation, WSL Debian 13 developer setup documentation, and the BounceCast roadmap.
- Added the multi-streamer dashboard foundation.
- Added Studio admin APIs.
- Added per-streamer RTMP stream keys.
- Added stream go-live event tracking.
- Added queued go-live notifications.
- Added webhook notifications for BounceCast stream events.

### Notes

- Go module paths, import paths, persisted config keys, API route names, migration compatibility, RTMP/HLS internals, and Docker startup behavior were intentionally preserved unless a later task proved a change safe.
- BounceCast remains an Owncast fork and retains required upstream attribution.
