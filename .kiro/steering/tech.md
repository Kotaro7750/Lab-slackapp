# Technology Stack

updated_at: 2026-04-30

## Architecture

単一 Go binary の Slack Socket Mode アプリとして構成する。`main.go` は起動配線、設定読み込み、Slack client / Socket Mode handler の生成、共通イベントログに集中させる。コマンド固有の処理は `internal/` 配下の feature package に置く。

Slack の interactive event は handler 内で ack する。KEDA launcher receiver への送信は `launchRequest` 内で timeout 付き context を使い、Slack webhook response posting の責務とは分ける。

## Core Technologies

- **Language**: Go 1.26.2
- **Slack SDK**: `github.com/slack-go/slack` と `socketmode`
- **Config**: `github.com/caarlos0/env/v11`
- **KEDA launcher client**: `github.com/Kotaro7750/keda-launcher-scaler/pkg/client`
- **Logging**: 標準 `log/slog` の JSON handler
- **Tooling**: `mise` による Go version pinning と `.env` loading

## Development Standards

### Comments

構造体や関数には、一目で責務が分かる短いものを除いてコメントを付ける。コメントは「何をするか」「なぜその境界にしているか」を短く説明し、処理内容を行ごとになぞるだけの説明は避ける。

Slack interactive event の ack タイミング、`private_metadata` による状態受け渡し、外部 KEDA launcher 送信前後の制御など、読み落とすと挙動を誤解しやすいロジックには、意図を説明する短いコメントを置く。Go コード内のコメントは英語で書く。

### Configuration

環境変数は `caarlos0/env` の struct tag で宣言する。`.env` の読み込みはアプリ内で自前実装せず、repo-local `mise` 設定に任せる。

Required variables:

- `SLACK_BOT_TOKEN`
- `SLACK_APP_TOKEN`
- `KEDA_LAUNCHER_RECEIVER_URL`

Optional/defaulted variables:

- `SLACK_LAUNCH_COMMAND` defaults to `/launch`

### Slack Integration

`slack-go/slack` と `socketmode` の型をそのまま使う。外部ライブラリの型や API を隠すだけの薄い wrapper package は増やさない。UI block や view state の小さな補助が必要な場合は、まず利用 feature の unexported helper として置く。

### Logging

`main()` で `slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))` を直接設定する。一行程度の初期化は過剰に helper 化しない。

### Testing

テストは repo-owned behavior を対象にする。Slack SDK や `socketmode` の内部 map、block 構造そのものを細かく再検証するテストは避ける。

Good test targets:

- modal submit から正しい `LaunchRequest` が作られる。
- invalid input が Slack view submission error になる。
- KEDA launcher client へ期待する request が送られる。
- accepted response に change duration 用 metadata が含まれる。
- change duration submit が request id と ScaledObject を維持し、duration だけ更新する。

## Common Commands

```sh
mise install
mise exec -- go run .
GOCACHE=$(pwd)/.gocache mise exec -- go test ./...
```

通常の Go build cache が権限で使えない場合は、repo-local `GOCACHE=$(pwd)/.gocache` を使う。

## Key Technical Decisions

- Slack App の入口は Socket Mode と slash command に寄せ、HTTP server をこのアプリ内に立てない。
- KEDA launcher との契約は外部 client package を信頼し、この repo では再定義しない。
- Slack response は ephemeral を基本にし、duration 変更時は元メッセージを置き換えられるよう `ReplaceOriginal` を使う。
- 起動配線は `main.go` に見える形で残し、command の詳細処理だけを `internal/` に切り出す。
