# AudioStream Discord Bridge

AudioStream Discord Bridge connects a Discord voice channel to a web-friendly audio stream. The application joins a Discord voice channel, captures incoming Opus audio frames, and repackages them into an HTTP `audio/ogg` stream that can be played directly by an HTML5 `<audio>` element. No Mumble server is required.

## Features

- 🎧 Listen to a Discord voice channel from any modern browser using a standard audio tag.
- 🔁 Real-time Opus forwarding with minimal buffering and per-listener backpressure protection.
- 🌐 Simple HTTP API with `/stream` for audio and `/healthz` for monitoring.
- 🧰 Runs as a single Go binary or Docker container; configuration via flags or environment variables.

## Requirements

1. A Discord bot with permission to connect to the target voice channel.
2. The numeric Guild ID (`DISCORD_GID`) and Channel ID (`DISCORD_CID`) for the voice channel.
3. The bot token (`DISCORD_TOKEN`).

Refer to the [Discord developer portal](https://discord.com/developers/applications) for instructions on creating a bot and retrieving these identifiers. Enable the following privileged intent for the bot: **Server Members Intent** (required for voice state updates).

## Quick Start

```bash
# Optionally load values from a .env file in the working directory
cat <<'ENV' > .env
DISCORD_TOKEN=your_bot_token
DISCORD_GID=your_guild_id
DISCORD_CID=your_voice_channel_id
HTTP_BIND=:8080
ENV

# Run the bridge
make dev-race  # or go run ./cmd/audostream-discord-bridge
```

When the application is running, open a browser and load an HTML page containing:

```html
<audio controls autoplay src="http://localhost:8080/stream"></audio>
```

The element will start playing the live Discord audio as soon as someone speaks in the channel.

## Configuration

All configuration values can be provided via environment variables or command-line flags. The following table lists the available options:

| Environment Variable | Flag             | Default  | Description |
| -------------------- | ---------------- | -------- | ----------- |
| `DISCORD_TOKEN`      | `-discord-token` | _(none)_ | Discord bot token (required). |
| `DISCORD_GID`        | `-discord-gid`   | _(none)_ | Discord guild (server) ID (required). |
| `DISCORD_CID`        | `-discord-cid`   | _(none)_ | Discord voice channel ID (required). |
| `HTTP_BIND`          | `-http-bind`     | `:8080`  | Address for the HTTP server (e.g. `0.0.0.0:8080`). |
| `STREAM_BUFFER`      | `-stream-buffer` | `256`    | Number of Opus packets buffered per listener before dropping audio. |

To see the complete list of flags, run:

```bash
./audostream-discord-bridge --help
```

## HTTP Endpoints

| Endpoint   | Description |
| ---------- | ----------- |
| `/stream`  | Live `audio/ogg` stream containing Opus audio from Discord. Designed for direct use in `<audio>` or media players. |
| `/healthz` | Returns `200 OK` when the service is running. Useful for container health checks. |

## Docker Usage

```bash
docker run \
  -e DISCORD_TOKEN=your_bot_token \
  -e DISCORD_GID=your_guild_id \
  -e DISCORD_CID=your_voice_channel_id \
  -e HTTP_BIND=0.0.0.0:8080 \
  -p 8080:8080 \
  --name audostream-discord-bridge \
  stieneee/audostream-discord-bridge:latest
```

Then point your HTML5 audio player to `http://<host>:8080/stream`.

## Development

AudioStream Discord Bridge is written in Go. Useful commands during development:

```bash
go run ./cmd/audostream-discord-bridge
make format
make test-chart
```

Unit tests cover utility packages (e.g., timing helpers). The streaming binary can be built with `goreleaser` using the provided configuration.

## License

This project is released under the [MIT License](LICENSE).
