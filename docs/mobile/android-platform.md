# BounceCast Android Platform

This document records the initial Android/mobile platform architecture for the BounceCast fork of Owncast.

## Phase 1 Inspection Findings

BounceCast already has the foundations the native Android client must reuse:

- Livestream playback is served by the existing web server at `/hls/*`; the public HLS entry point is `/hls/stream.m3u8`.
- Stream state is exposed through `/api/status` and the internal `core.GetStatus()` path.
- Public web configuration is exposed through `/api/config`, including site name, stream title, logo, offline message, chat settings, Tenor GIF key, browser push settings, and appearance variables.
- Real-time chat uses the existing websocket endpoint `/ws?accessToken=...`.
- Chat identity uses the existing `users` and `user_access_tokens` tables.
- Chat history is exposed through the existing generated internal endpoint `/api/chat?accessToken=...`.
- Chat messages, moderation visibility updates, bans, disabled users, chat reactions, Stars, and Rewards events all flow through the existing websocket/event system.
- Original Owncast-compatible admin auth still exists, with BounceCast owner/admin role wrappers added around newer admin routes.
- BounceCast account, Studio, DJ, schedule, Stars, Rewards, Messenger, and notification features are additive tables and handlers on the same SQLite database.
- Go-live events are detected around RTMP stream key/session handling in `core/rtmp/bouncecast_keys.go`.

## What Is Reused Directly

- HLS stream: `/hls/stream.m3u8`
- Stream status: `/api/status` and `core.GetStatus()`
- Chat websocket: `/ws`
- Chat history: `/api/chat`
- Chat registration: `/api/chat/register`
- Chat moderation and delete/visibility events
- BounceCast account tokens and roles
- Existing admin auth and owner/admin route guards
- Existing CORS hardening for BounceCast browser/mobile APIs

## Adapter Layer

The mobile API is a clean adapter over existing BounceCast behavior:

- `/api/mobile/v1/config`
- `/api/mobile/v1/theme`
- `/api/mobile/v1/navigation`
- `/api/mobile/v1/features`
- `/api/mobile/v1/assets`
- `/api/mobile/v1/legal`
- `/api/mobile/v1/stream-status`
- `/api/mobile/v1/chat/config`
- `/api/mobile/v1/ads`
- `/api/mobile/v1/notifications/config`
- `/api/mobile/v1/devices/register`
- `/api/mobile/v1/devices/unregister`
- `/api/mobile/v1/devices/preferences`

These endpoints do not create a second stream, chat server, message store, or community layer.

## Admin Foundation

Owner/admin configuration starts at:

- Admin -> Integrations -> Mobile App Platform
- API: `/api/admin/bouncecast/mobile`
- Save API: `/api/admin/bouncecast/mobile/settings`
- Asset upload API: `/api/admin/bouncecast/mobile/assets/upload`

The admin screen manages:

- App display name, public BounceCast URL, maintenance mode, force update, support URL, and version gates
- Logo/splash/icon/background image URLs and mobile theme colors
- Mobile feature flags, including chat reactions, Stars, Stars overlays, Rewards Wheel, and Rewards overlays
- Google/Unity ad IDs, provider priority, app-open behavior, fallback, timeout, retries, and consent flags
- FCM go-live push templates and topic/project metadata
- Navigation and legal page JSON
- Registered Android device token count

Secrets such as FCM service account credentials are intentionally not exposed in the public API. FCM delivery is a later stage that should load server-side credentials from environment or encrypted storage only.

## Database Tables

Migration `00015_bouncecast_mobile_platform.sql` adds:

- `mobile_app_settings`
- `mobile_branding_settings`
- `mobile_navigation_items`
- `mobile_feature_flags`
- `mobile_ad_settings`
- `mobile_notification_settings`
- `mobile_device_tokens`
- `mobile_legal_pages`
- `mobile_asset_uploads`
- `mobile_notification_logs`

Migration `00016_bouncecast_mobile_app_features.sql` extends the mobile settings with:

- `mobile_branding_settings.app_background_url`
- Mobile feature flags for Stars, chat reactions, Stars overlays, Rewards Wheel, and reward overlays

The device-token table stores a token hash, preview, and protected token value. Set `BOUNCECAST_SECRET_KEY` in production so token values are encrypted at rest.

## Proposed Android Architecture

The Android app should be a native Kotlin client:

- Kotlin
- Jetpack Compose
- Material 3
- AndroidX Media3 / ExoPlayer for HLS
- OkHttp/Retrofit or Ktor client
- WebSocket client for `/ws`
- Coroutines + Flow
- DataStore for cached remote config
- Firebase Cloud Messaging
- Google Mobile Ads SDK
- Unity Ads behind the same ad-provider abstraction

Suggested app packages/modules:

- `app`
- `core-network`
- `core-model`
- `core-storage`
- `core-ui`
- `feature-stream`
- `feature-chat`
- `feature-auth`
- `feature-profile`
- `feature-notifications`
- `feature-ads`
- `feature-settings`

The app should fetch `/api/mobile/v1/config` on launch, cache the last valid config, and fall back to safe defaults if the API is temporarily unavailable.

## Android Client Integration Rules

- Watch the same HLS stream as desktop users.
- Connect to the same `/ws` chat websocket as desktop users.
- Use the same account/user token identity source.
- Render the same chat/system/moderation events.
- Send chat messages through the existing Owncast/BounceCast chat protocol.
- Treat Stars, Rewards, GIFs, reactions, badges, and future events as shared event payloads.
- Never duplicate chat storage or create mobile-only message tables.

## Environment Defaults

Optional server-side defaults:

- `BOUNCECAST_MOBILE_BASE_URL`
- `BOUNCECAST_SECRET_KEY`
- `FIREBASE_PROJECT_ID`
- Future FCM credentials should be kept server-side only.

## Mobile Config Shape

The public config includes:

- `app`
- `branding`
- `bouncecast`
- `features`
- `ads`
- `notifications`
- `navigation`
- `legal`
- `stream`
- `chat`

The `bouncecast` block contains the real stream/chat URLs the app should use:

```json
{
  "stream_url": "https://example.com/hls/stream.m3u8",
  "chat_websocket_url": "wss://example.com/ws",
  "stream_status_url": "https://example.com/api/mobile/v1/stream-status",
  "chat_history_url": "https://example.com/api/chat",
  "register_chat_url": "https://example.com/api/chat/register"
}
```

## Safe Implementation Plan

Stage 1:

- Mobile backend config/admin foundation
- Versioned public mobile API
- Schema migration and defaults
- Device token registration foundation
- Backend tests and docs

Stage 2:

- Android project foundation
- Remote config models, fetching, validation, and DataStore cache
- App theming and navigation from backend config

Initial source lives in `android/` and includes a native Kotlin/Compose app with:

- Remote config loading and local DataStore cache
- Material 3 theme driven by backend colors
- Media3 HLS player pointed at BounceCast `/hls/stream.m3u8`
- Existing BounceCast websocket chat integration through `/ws?accessToken=...`
- Existing chat registration/history endpoints
- Chat reaction send/render support
- Stars and Rewards overlay rendering from shared websocket events
- Stars entry button that opens the existing BounceCast account page
- Native Rewards Wheel panel that uses the viewer access token to show active prize options, earning tasks, Spin Credits, and spin results
- Admin-configured app background image rendering
- Media3 HLS playback set to fit landscape video instead of cropping/zooming it
- FCM receiver/deep link foundation
- Google and Unity banner provider abstractions with fallback ordering
- App-open ad cooldown logic placeholder

Stage 3:

- HLS playback through Media3
- Stream status/offline screen
- Reconnect and player error states

Stage 4:

- Existing BounceCast websocket chat integration
- Chat history, send, moderation events, GIFs, reactions, Stars, and Rewards event rendering

Stage 5:

- FCM go-live push sender
- Topic/device targeting
- Notification logs and test send tooling

Stage 6:

- Google Ads and Unity Ads provider abstractions
- Banner/app-open fallback logic
- Consent handling

Stage 7:

- Android/backend tests
- Manual desktop-to-Android chat verification
- Production checklist and release build instructions

## Risks And Mitigations

- Chat protocol drift: keep Android using `/ws` and shared event models rather than a mobile-only chat API.
- Secret leakage: never include FCM server credentials or ad account secrets in `/api/mobile/v1/*`.
- Bad remote config: Android must validate and cache config before applying it.
- Go-live notification duplicates: FCM stage must use go-live event/session IDs and cooldown locks.
- CORS/API exposure: public config is safe by design; admin writes remain owner-protected.
- Migration safety: new tables are additive and do not rename Owncast module paths, API routes, RTMP logic, HLS logic, or existing persisted config keys.

## Local Android Verification

Gradle was installed locally at `C:\tools\gradle-8.12.1` because the system PATH did not include Gradle and Chocolatey could not install packages without elevation.

From the repository root:

```powershell
$env:ANDROID_HOME="$env:LOCALAPPDATA\Android\Sdk"
$env:ANDROID_SDK_ROOT=$env:ANDROID_HOME
C:\tools\gradle-8.12.1\bin\gradle.bat -p android :app:testDebugUnitTest :app:assembleDebug --no-daemon
```

## Production Checklist

- Set the public BounceCast URL in Admin -> Integrations -> Mobile App Platform.
- Set `BOUNCECAST_SECRET_KEY` before accepting Android push tokens.
- Add Firebase project details and keep FCM service-account credentials server-side only.
- Enable FCM go-live sending only after server-side sender credentials and notification logs are wired.
- Keep AdMob and Unity in test mode until store approval and consent flows are complete.
- Add real privacy-policy and terms URLs before enabling ads or push notifications.
- Test desktop chat to Android chat and Android chat to desktop chat before release.
- Test admin moderation/deleted-message behavior while Android is connected.
- Test HLS playback over the production Plesk/Cloudflare domain and direct RTMP ingest separately.
- Generate signed release APK/AAB from Android Studio or Gradle with a private keystore outside the repository.

## Current Limitations

- The Android app foundation compiles and renders the stream/chat shell, reactions, Stars/Rewards entry points, native Rewards Wheel options/spins, and live overlay events, but profile login UX, persisted account token storage, native Stars purchase/send forms, reward claim submission forms, full moderation controls, and production push sending still need deeper passes.
- FCM receive/deep-link scaffolding exists; server-side FCM fan-out is not implemented yet.
- App-open ad display is a non-blocking placeholder until provider-specific loading and consent are wired.
- The Google Services Gradle plugin is intentionally not applied until a real `google-services.json` exists outside source control.
- Android chat currently registers a default display name for the first shell pass; the next auth pass should use the unified BounceCast login/account identity.
