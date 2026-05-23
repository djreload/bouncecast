# BounceCast Changelog

This changelog tracks the BounceCast fork work from the first visible rebrand onward. Keep new user-facing features, backend behavior changes, migrations, deployment notes, and known compatibility risks here as the project evolves.

## Unreleased / Next

### Planned

- Continue expanding owner/admin UI hiding and audit coverage across older Owncast-compatible admin pages.
- Continue reviewing older Owncast-compatible API auth surfaces before changing any Basic-auth behavior.
- Continue live VPS deployment notes as production configuration changes.

## 2026-05-23

### Added

- Added `docs/auth-unification-audit.md` documenting the existing Owncast admin, BounceCast account, Studio, and chat identity auth flows before unifying the browser login experience.
- Added `/api/bouncecast/auth/login` as the primary browser login endpoint that can issue the normal account access token, admin session cookies, and Studio bearer token in one successful login.
- Added `/api/bouncecast/auth/logout` to clear BounceCast admin cookies and revoke supplied account/Studio tokens from the unified UI.

### Changed

- Changed `/admin/*` browser requests to redirect unauthenticated users to `/login?next=...` while keeping `/api/admin/*` Owncast-compatible Basic auth behavior intact.
- Changed `/login` into the single visible BounceCast access page for admins, DJs, moderators, and viewers, with account creation and DJ access requests as secondary actions on that same page.
- Changed `/account`, `/studio`, the public user menu, and admin logout controls to hand off to the unified login/logout flow instead of keeping separate login forms.
- Changed Account Hub role destination links to open the actual admin or Studio area once the unified login has issued the correct session.

### Security

- Added rate limiting to the unified login path and added rate limiting to the older admin and Studio login/register endpoints.
- Preserved the original Owncast `admin` credentials as owner-level BounceCast access without exposing passwords or changing existing API clients.

## 2026-05-19

### Added

- Added database migration `00012_bouncecast_facebook_messenger_alerts.sql` for Meta Messenger opt-in subscribers, go-live alert campaigns, per-recipient campaign results, and per-DJ SEO/share customization fields.
- Added Admin -> Integrations -> Facebook Messenger Alerts with enablement, Meta App/Page configuration, masked secrets, webhook helper details, token validation, message preview, test sends, subscriber stats, and campaign logs.
- Added Meta Messenger webhook verification and POST event handling at `/integrations/facebook/messenger/webhook`, including App Secret signature validation when enabled.
- Added Messenger opt-in keyword handling for `LIVE`, `START`, and `SUBSCRIBE`, opt-out handling for `STOP`, `UNSUBSCRIBE`, and `CANCEL`, and `HELP` replies.
- Added go-live Messenger alert campaign sending with configured delay, cooldown/idempotency, 24-hour policy-window checks, per-recipient logs, and optional Facebook Page post fallback.
- Added a public "Get Messenger live alerts" CTA for the stream page and DJ profile page that sends viewers to the connected Page with the `LIVE` opt-in keyword.
- Added retry and CSV export tools for failed BounceCast notification deliveries in `/admin/schedule/`.
- Added per-DJ SEO title, share image, and featured schedule controls in `/admin/streamers/`, with bot-facing metadata support on `/djs/{handle}`.
- Added `docs/integrations/facebook-messenger-live-alerts.md` with Meta setup, webhook, policy, opt-in, troubleshooting, and production notes.
- Added database migration `00011_bouncecast_reminder_delivery_management.sql` for account-bound browser push subscriptions, reminder delivery status, notification delivery reminder links, and product audit events.
- Added admin schedule reminder management at `/admin/schedule/`, including viewer, channel, delivery status, and disable controls.
- Added Account Hub reminder status cards so viewers can see saved reminder channels and delivery state.
- Added per-account browser push endpoint mapping for schedule reminders, while keeping the existing global Owncast browser notification table in sync for compatibility.
- Added Messenger delivery settings for schedule reminders, storing the Page Access Token server-side only.
- Added Messenger reminder delivery transport through the Facebook Graph API when enabled.
- Added owner-only audit event viewing and audit records for sensitive BounceCast streamer, stream key, schedule, email, Messenger, and reminder actions.
- Added shareable public DJ profile URLs like `/djs/{handle}` with bot-facing metadata for profile name, bio, image, and genre tags.
- Added database migration `00010_bouncecast_profiles_permissions_reminders.sql` for rich DJ profile metadata and per-schedule viewer reminders.
- Added public DJ profile fields for bio, genres, social links, and hero artwork.
- Added Studio and admin profile editing for DJ public profile metadata.
- Added public schedule filtering by status, search query, handle, date range, and past inclusion.
- Added Account Hub "Remind me" actions for logged-in viewers to save per-set reminder preferences.
- Added go-live email delivery queuing for saved schedule reminders.
- Added signed BounceCast admin identity cookies so owner/admin account roles can be enforced on specific admin actions.

### Changed

- Changed Messenger go-live alerts to use a dedicated BounceCast service module rather than the earlier lightweight reminder-only transport.
- Changed public DJ profiles so configured SEO title/share image/featured schedule can override automatic metadata and cards.
- Upgraded the shared BounceCast visual system across the public stream page, DJ pages, Studio-style admin pages, forms, tables, dropdowns, modals, and Messenger alert CTA.
- Changed more BounceCast admin writes, including streamer account changes, stream key changes, schedule creation, and reminder disabling, to require owner/admin product roles.
- Changed `/admin/schedule/` to include Messenger settings and account reminder management alongside email, push, subscribers, live events, and delivery logs.
- Changed Account Hub reminder saving so browser push reminders attempt to capture the browser push subscription endpoint at the point of opt-in.
- Changed public DJ profile links to prefer `/djs/{handle}` while keeping the existing `?handle=` path working.
- Changed sensitive admin writes for account permission changes, Stars payment settings/packages/manual wallet adjustments, and SMTP settings to require owner-level access while preserving the original Owncast admin password as owner-level compatibility access.
- Updated `/djs` with richer DJ cards, profile artwork, social links, public lineup filters, and a filtered lineup panel.
- Updated `/admin/streamers/` and `/studio` so DJ public profiles can be managed without direct database edits.
- Updated `/admin/schedule/` so new sets can be marked public/private and planned/live at creation time.
- Updated the generated static web bundle so Docker serves the new Account Hub, DJ profile, schedule, Studio, and admin UI.
- Updated Debian 13/Plesk live server notes with the profile/reminder migration, owner-only settings behavior, and deployment checks.

## 2026-05-18

### Added

- Added public BounceCast discovery APIs for active DJ profiles, individual DJ profiles, and public schedule lineup.
- Added a protected unified Account Hub API that combines profile, notification preferences, roles, dashboard destinations, Stars wallet summary, leaderboard, and linked DJ schedule context.
- Added `/account` as a public Account Hub page for login/register, profile pictures, notification opt-ins, Stars wallet status, role destinations, and linked DJ schedule.
- Added `/djs` as a public DJ roster/profile page.
- Added an upcoming lineup panel to the public stream homepage.
- Added Account Hub and DJ lineup entries to the public user menu.
- Added an admin command-center summary for accounts, DJ access, public schedule, and Stars activity.
- Added database migration `00009_bouncecast_account_notifications.sql` for public account notification opt-in preferences.
- Added public account notification preference persistence for email, browser push, and Messenger destinations.
- Added public account UI controls for email, browser push, and Facebook Messenger go-live opt-ins.
- Added a public Stars leaderboard, ranked by total Stars sent, so viewers can compete for 1st, 2nd, and 3rd place.
- Added the Stars leaderboard to the admin Stars dashboard.
- Added an admin-only Stars overlay test action for previewing effects without spending Stars or changing wallets.
- Added owner/admin BounceCast account role login support for the admin dashboard while preserving the original Owncast admin credentials.
- Added DJ BounceCast account role login support for Studio, including automatic active Studio account provisioning on successful DJ-role login.
- Added in-memory rate limiting for public account registration, account login, profile updates, profile image uploads, Stars checkout, Stars payment capture, and Stars sending.
- Added local profile image uploads for public accounts, storing validated PNG/JPG/GIF images under `data/public/profiles`.

### Changed

- Changed `/admin/accounts/` from a multi-select role editor into a clearer permission matrix for owner/admin/moderator/DJ roles.
- Upgraded Stars overlay effects so sparkle, fireworks, hearts, hype, and DJ drop selections trigger distinct on-screen animations instead of only changing the toast styling.
- Updated linked Studio accounts so if a matching public BounceCast account exists, Studio access now requires that account to keep the `dj` role.
- Updated the public account modal so users can upload a profile picture or keep using an external profile picture URL.
- Updated Debian 13/Plesk live server notes with profile image storage and rate-limit deployment guidance.

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
