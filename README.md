# Lab Slack App

A Slack Socket Mode app that sends KEDA launcher scaler launch requests from the Slack `/launch` command.

This repository owns only the integration point between Slack UI and the KEDA launcher client. The KEDA launcher receiver API contract and Kubernetes-side control logic are delegated to the client package from `github.com/Kotaro7750/keda-launcher-scaler`.

## Features

- Opens a KEDA launch request modal from the `/launch` slash command
- Accepts namespace, ScaledObject name, and duration through a Slack modal
- Sends a `LaunchRequest` to the KEDA launcher receiver
- Returns accepted responses as Slack ephemeral messages
- Lets users change only the duration for the same request id and ScaledObject from the accepted response button
- Acknowledges Socket Mode interactive events before external requests, then sends the external request with a bounded timeout

## Requirements

- Go 1.26.2
- `mise`
- Slack App Bot Token and App-Level Token
- KEDA launcher receiver URL

## Configuration

Copy `.env.example` to `.env` and fill in the values. The repo-local `.mise.toml` loads `.env`.

```sh
cp .env.example .env
```

| Variable | Required | Default | Description |
| --- | --- | --- | --- |
| `SLACK_BOT_TOKEN` | yes | - | Slack Bot Token starting with `xoxb-` |
| `SLACK_APP_TOKEN` | yes | - | App-Level Token starting with `xapp-` and the `connections:write` scope |
| `KEDA_LAUNCHER_RECEIVER_URL` | yes | - | KEDA launcher receiver URL |
| `SLACK_LAUNCH_COMMAND` | no | `/launch` | Slash command name to register |

## Slack App Configuration

Create the app with `Create New App > From an app manifest`, then paste this manifest.

```json
{
  "display_information": {
    "name": "Lab Slack App",
    "description": "Launch KEDA ScaledObjects from Slack."
  },
  "features": {
    "bot_user": {
      "display_name": "lab-slackapp",
      "always_online": false
    },
    "slash_commands": [
      {
        "command": "/launch",
        "description": "Open the KEDA launch request form.",
        "should_escape": false
      }
    ]
  },
  "oauth_config": {
    "scopes": {
      "bot": [
        "commands"
      ]
    }
  },
  "settings": {
    "interactivity": {
      "is_enabled": true
    },
    "org_deploy_enabled": false,
    "socket_mode_enabled": true,
    "token_rotation_enabled": false
  }
}
```

After creating the app from the manifest:

1. Install the app to the workspace and copy the Bot User OAuth Token to `SLACK_BOT_TOKEN`.
2. Create an App-Level Token with the `connections:write` scope and copy it to `SLACK_APP_TOKEN`.
3. If you change `/launch` in the manifest, set the same value in `SLACK_LAUNCH_COMMAND`.

Slash command and interactivity request URLs are omitted because Socket Mode delivers those payloads over the app's WebSocket connection.

## Run

```sh
mise install
mise exec -- go run .
```

Run `/launch` in Slack to show the KEDA launch request form only to the user who invoked it. When the request is accepted, the app returns an ephemeral message with a change button that can resend only the duration with the same `requestId`.

## Test

```sh
GOCACHE=$(pwd)/.gocache mise exec -- go test ./...
```

Use the repo-local `GOCACHE` because the default Go build cache may not be writable in some environments.

## Docker

```sh
docker build -t lab-slackapp .
```

When a `v*.*.*` or `*.*.*` tag is pushed, GitHub Actions publishes linux/amd64 and linux/arm64 Docker images to Docker Hub.

## License

MIT
