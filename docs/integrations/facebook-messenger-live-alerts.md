# Facebook Messenger Live Alerts

BounceCast can send Facebook Messenger alerts when the BounceCast stream goes
live. The stream remains hosted on BounceCast. Facebook is only used as an
opt-in Messenger delivery channel.

This integration is intentionally Meta-policy-safe:

- BounceCast does not message all Facebook Page followers.
- A user must message your connected Facebook Page or use a supported Messenger
  opt-in flow before they can receive alerts.
- Standard Messenger replies are limited by Meta's 24-hour messaging window
  unless you have an approved notification flow or another allowed policy basis.
- Users can opt out at any time.

## Admin Setup

Open:

```text
Admin -> Integrations -> Facebook Messenger Alerts
```

Configure:

- Enable Facebook Messenger alerts
- Facebook App ID
- Facebook App Secret
- Facebook Page ID
- Facebook Page Access Token
- Webhook Verify Token
- Webhook App Secret validation
- Graph API version, default `v25.0`
- Default live alert message template
- Optional live URL override
- Button label, default `Watch Live`
- Send delay after go-live
- Cooldown between live alert sends
- Test recipient PSID
- Optional Facebook Page post fallback

Secrets are never returned to the browser after they are saved. If
`BOUNCECAST_SECRET_KEY` or `FACEBOOK_MESSENGER_SECRET_KEY` is set, new saved
secrets are encrypted at rest. Without one of those keys, BounceCast stores the
values in the existing local settings table, so protect the database and server.

## Environment Defaults

These environment variables are used as defaults when an admin value has not
been saved:

```bash
FACEBOOK_MESSENGER_ENABLED=true
FACEBOOK_APP_ID=your-app-id
FACEBOOK_APP_SECRET=your-app-secret
FACEBOOK_PAGE_ID=your-page-id
FACEBOOK_PAGE_ACCESS_TOKEN=your-page-access-token
FACEBOOK_WEBHOOK_VERIFY_TOKEN=choose-a-random-token
FACEBOOK_GRAPH_API_VERSION=v25.0
BOUNCECAST_SECRET_KEY=use-a-long-random-secret-for-encrypting-saved-secrets
```

Admin settings override these defaults where appropriate. Blank secret fields in
the admin form keep the existing stored or environment value.

## Meta Developer App

1. Create or open a Meta Developer App.
2. Add the Messenger product.
3. Connect the Facebook Page you want BounceCast to send from.
4. Generate a Page Access Token.
5. Add the webhook callback URL shown in BounceCast:

```text
https://your-domain.example/integrations/facebook/messenger/webhook
```

6. Use the same webhook verify token you entered in BounceCast.
7. Subscribe the webhook to Page Messenger message events.
8. Request the permissions required by your app mode and use case, usually
   `pages_messaging`. Page post fallback additionally needs Page publishing
   permissions.

Meta app review may be required before this works for the public.

## User Opt-In

The public stream page and DJ profile page show a "Get Messenger live alerts"
CTA when Messenger alerts are enabled and a Page ID is configured.

Users are sent to the Facebook Page Messenger thread and should message:

```text
LIVE
```

BounceCast also handles:

- `START` or `SUBSCRIBE` to opt in
- `STOP`, `UNSUBSCRIBE`, or `CANCEL` to opt out
- `HELP` for instructions

The webhook stores the user's PSID and opt-in status. BounceCast does not import
or message Page followers who have not interacted or opted in.

## Go-Live Behavior

When a BounceCast RTMP stream transitions offline to online:

1. BounceCast creates a go-live event.
2. Existing email, browser push, webhook, and schedule reminder deliveries are
   queued as before.
3. Messenger Alerts waits the configured delay.
4. A go-live campaign is created once for that go-live event.
5. The cooldown prevents duplicate sends during reconnect flapping.
6. Only active opted-in subscribers are considered.
7. Subscribers outside the 24-hour Messenger window are skipped unless their
   metadata marks an allowed notification opt-in.
8. Per-recipient send results are logged.

Offline alerts are not sent.

## Templates

Default message:

```text
We're live now!
{stream_title}

Watch here:
{stream_url}
```

Supported variables:

- `{site_name}`
- `{stream_title}`
- `{stream_url}`
- `{channel_name}`
- `{started_at}`
- `{facebook_page_name}`

BounceCast first tries a Messenger button template using the configured button
label and stream URL. If that fails, it falls back to a plain text message.

## Admin Tools

The integration page includes:

- Validate Meta config
- Send test message to PSID
- Send test go-live alert to PSID
- Preview message template
- Copy webhook callback URL
- Subscriber counts and subscriber table
- Campaign/send logs

The Schedule admin page also includes retry/export tools for failed notification
deliveries.

## Troubleshooting

- Webhook verification fails: check the callback URL, HTTPS certificate, and
  verify token.
- Webhook POST fails: enable App Secret validation only after the App Secret is
  saved correctly.
- Token validation fails: regenerate the Page Access Token and check app/Page
  permissions.
- Messages are skipped: the user may be outside the standard 24-hour Messenger
  messaging window or opted out.
- Messages fail with permission errors: check `pages_messaging`, app mode, Page
  connection, and app review status.
- Messages fail for one user: they may have blocked the Page or deleted the
  Messenger thread.

## Production Notes

Use a stable public HTTPS domain for the webhook. If BounceCast is behind Plesk,
Cloudflare, or a tunnel, make sure the callback URL routes to BounceCast and not
another project.

Keep `BOUNCECAST_SECRET_KEY` stable once set. Changing it means previously
encrypted admin-saved Messenger secrets cannot be decrypted.

## Official Meta References

- Messenger Platform: https://developers.facebook.com/docs/messenger-platform
- Messenger Send API: https://developers.facebook.com/docs/messenger-platform/send-messages
- Messenger webhooks: https://developers.facebook.com/docs/messenger-platform/webhooks
- Webhook signatures: https://developers.facebook.com/docs/graph-api/webhooks/getting-started
- Messenger Platform policy overview: https://developers.facebook.com/docs/messenger-platform/policy/policy-overview
