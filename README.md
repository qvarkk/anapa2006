# anapa2006

An automated tool for telegram channel managent interfaced via a telegram bot.

## Running

This bot depends on [RSSHub](https://docs.rsshub.app/) for fetching source channels. Self-hosted instance is included as a service in `bot/compose.yaml`, so no separate setup is needed if you use compose.

1. Copy `.env.example` to `.env` and fill in `TELEGRAM_BOT_TOKEN`, the two channel IDs and `TELEGRAM_OWNER_ID`.
2. `docker compose -f bot/compose.yaml -f bot/compose.prod.yaml up -d`
3. Check both services are healthy: `docker compose -f bot/compose.yaml -f bot/compose.prod.yaml ps`
4. Send `/bootstrap` to the bot from the account whose numeric ID is in `TELEGRAM_OWNER_ID`. This adds the owner to `allowed_users`, without which the bot ignores everyone.

## Access model

- **Owner** — the account in `TELEGRAM_OWNER_ID`. Full access, and the only one who can manage users.
- **Admins** — everyone else in `allowed_users`. They can use the bot and manage channels, but not users.
- Users are added by the owner from the access menu by numeric ID, or by forwarding a message from that person.
- Channels are managed from the channels menu: add by handle (without `@`), enable/disable, delete.
- Target channels (where posts are published) are managed from the target channels menu: add by handle, delete. `TELEGRAM_CHANNEL_CHAT_ID` seeds the first target channel when the table is empty; the media cache channel stays an env variable.

> [!IMPORTANT]
> If you're pulling container from [ghcr.io](https://ghcr.io/qvarkk/anapa2006) then use a public RSSHub instance or host one yourself.
> Then specify link to it in the `.env` config with the `RSSHUB_BASE_URL` variable.

> [!NOTE]
> The schema is applied with `CREATE TABLE IF NOT EXISTS`. When the bot refuses to start with an "outdated database schema" error, delete the database file (or the docker volume) and start again.
