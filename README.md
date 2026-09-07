# anapa2006

An automated tool for telegram channel managent interfaced via a telegram bot.

## Running

This bot depends on [RSSHub](https://docs.rsshub.app/) for fetching source channels. Self-hosted instance is included as a service in `bot/compose.yaml`, so no separate setup is needed if you use compose.

1. Copy `.env.example` to `.env` and fill in `TELEGRAM_BOT_TOKEN`.
2. `docker compose -f bot/compose.yaml -f bot/compose.prod.yaml up -d`
3. Check both services are healthy: `docker compose -f bot/compose.yaml -f bot/compose.prod.yaml ps`

> [!IMPORTANT]
> If you're pulling container from [ghcr.io](https://ghcr.io/qvarkk/anapa2006) then use a public RSSHub instance or host one yourself.
> Then specify link to it in the `.env` config with the `RSSHUB_BASE_URL` variable.
