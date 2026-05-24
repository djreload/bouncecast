# BounceCast Rewards Wheel

BounceCast Rewards Wheel is an internal/manual prize system. It does not use WooCommerce, external shops, checkout, payments, paid spins, streamer payouts, creator balances, or withdrawal dashboards.

## Viewer Flow

1. Viewers log in with a BounceCast account.
2. Viewers earn Spin Credits from admin grants today, with chat, task, achievement, and top-supporter hooks ready for deeper automation.
3. Viewers open `/rewards`.
4. Viewers spend 1 Spin Credit per spin by default.
5. The server selects the prize with weighted odds and records the spin before the frontend animation reveals the result.
6. Sorry prizes do not create fulfilment records.
7. Physical and digital prize wins create:
   - winner record
   - claim record
   - fulfilment order
   - admin message
   - viewer notification
   - live overlay event

## Admin Flow

Open `Admin -> Studio -> Rewards Wheel`.

Admin tabs include:

- Settings: enable/disable the wheel, spin cost, chat reward placeholders, top supporter placeholder values, and overlay template/sound.
- Prizes: create and edit physical, digital, discount placeholder, and sorry prizes.
- Spin Credits: manually grant or deduct credits with a ledger entry.
- Fulfilment: manage internal orders, update status, mark dispatched, and export CSV.
- Messages: review unread reward win alerts.
- Tasks & Achievements: configure placeholder reward tasks and achievement records.

## Prize Types

- `physical`: Creates a claim/order and can require delivery details.
- `digital`: Creates a claim/order for manual fulfilment.
- `discount_future_placeholder`: Records a spin result only. It is reserved for a later discount/shop integration and has no checkout behavior.
- `sorry`: Records the spin only. It never creates a winner, claim, order, admin alert, or overlay.

## Spin Credits

Balances are stored in `reward_spin_balances`.

Every balance change creates a ledger row in `reward_spin_ledger`. The service entry point is:

```go
AwardSpinCredits(userID, amount, source, referenceID, note)
```

Supported award sources:

- `chat_activity`
- `task_completed`
- `achievement_unlocked`
- `top_supporter_reward`
- `admin_grant`
- `admin_adjustment`

Spin spending is recorded internally as `spin_spend`.

## Security Notes

- Frontend never chooses a prize.
- Frontend never grants credits.
- Wheel spins require a logged-in user access token.
- The spin endpoint is rate limited.
- Credit deduction, prize selection, stock reduction, claim creation, order creation, and winner recording happen inside the server-side locked transaction path.
- Balances cannot go negative.
- Delivery details are visible only to the owning user and admins.
- Marketing consent is optional unless the configured prize explicitly requires it, and it is stored with timestamp/consent text.

## Database Tables

Migration `00013_bouncecast_rewards_wheel.sql` adds:

- `reward_config`
- `reward_prizes`
- `reward_spin_balances`
- `reward_spin_ledger`
- `reward_spins`
- `reward_winners`
- `reward_claims`
- `reward_orders`
- `reward_admin_messages`
- `reward_user_notifications`
- `reward_tasks`
- `reward_achievements`

## API Endpoints

Viewer:

- `GET /api/rewards/wheel`
- `GET /api/rewards/balance`
- `POST /api/rewards/spin`
- `GET /api/rewards/history`
- `GET /api/rewards/claims`
- `POST /api/rewards/claims/submit`
- `GET /api/rewards/notifications`
- `POST /api/rewards/notifications/read`

Admin:

- `GET /api/admin/bouncecast/rewards`
- `POST /api/admin/bouncecast/rewards/settings`
- `POST /api/admin/bouncecast/rewards/prizes`
- `POST /api/admin/bouncecast/rewards/credits/adjust`
- `POST /api/admin/bouncecast/rewards/orders/update`
- `POST /api/admin/bouncecast/rewards/orders/dispatch`
- `GET /api/admin/bouncecast/rewards/orders/export`
- `POST /api/admin/bouncecast/rewards/messages/read`
- `POST /api/admin/bouncecast/rewards/tasks`
- `POST /api/admin/bouncecast/rewards/achievements`

## VPS Deployment Notes

For a live Debian 13/Plesk Docker install:

1. Pull the latest BounceCast image or rebuild the Docker image from this branch.
2. Stop the old container.
3. Start the new container with the same mounted data volume.
4. Watch logs for migration `00013_bouncecast_rewards_wheel.sql`.
5. Log in as owner/admin and open `Admin -> Studio -> Rewards Wheel`.
6. Keep the wheel disabled until prizes and fulfilment terms are configured.
7. Add at least one sorry prize and any real prizes with stock quantities.
8. Grant Spin Credits to a test account and test `/rewards`.

## Known Limitations

- Chat activity, tasks, achievements, and top-supporter rewards have storage/config placeholders and service entry points, but broader automation rules should be expanded carefully in a later pass.
- Reward win and dispatch emails use the existing BounceCast SMTP settings. If SMTP is disabled or an address is missing, the system records the failure and keeps the in-panel notification/admin message.
- Discount prizes are placeholders only and do not connect to any shop, checkout, or external fulfilment system.
