# BounceCast Android App

This is the native Android client for BounceCast. It is intentionally wired to the same BounceCast stream, chat, moderation, account, Stars, Rewards, and event backend used by the desktop web app.

## Requirements

- Android Studio Ladybug or newer
- JDK 17
- Android SDK 35
- A running BounceCast server with `/api/mobile/v1/config`
- Optional `google-services.json` for Firebase Cloud Messaging

## Local Setup

1. Open the `android` folder in Android Studio.
2. Create `android/local.properties` with your SDK path if Android Studio has not done it automatically.
3. For local testing, update `BuildConfig.DEFAULT_CONFIG_URL` in `android/app/build.gradle.kts` or use a debug build variant override.
4. Sync Gradle.
5. Run the `app` configuration on an emulator or device.

Command-line build from this machine:

```powershell
$env:ANDROID_HOME="$env:LOCALAPPDATA\Android\Sdk"
$env:ANDROID_SDK_ROOT=$env:ANDROID_HOME
C:\tools\gradle-8.12.1\bin\gradle.bat -p android :app:testDebugUnitTest :app:assembleDebug --no-daemon
```

If you add a real Firebase `google-services.json`, also apply the Google Services Gradle plugin in `app/build.gradle.kts`. The first scaffold keeps that plugin unapplied so the app can compile without committed Firebase credentials.

## Backend Contract

The app fetches:

- `GET /api/mobile/v1/config`

The returned config tells the app where to find:

- HLS stream: `/hls/stream.m3u8`
- Chat websocket: `/ws?accessToken=...`
- Chat history: `/api/chat?accessToken=...`
- Chat registration: `/api/chat/register`
- Stream status: `/api/mobile/v1/stream-status`

## Notes

- Do not add a separate mobile chat database.
- Do not add a separate mobile stream backend.
- FCM sender credentials stay server-side in BounceCast, never in the APK.
- Live AdMob/Unity IDs should be configured from the BounceCast admin panel, with test mode enabled during development.
- Unity is used as the app-open fallback via an interstitial-style abstraction until a true Unity app-open product is available.
