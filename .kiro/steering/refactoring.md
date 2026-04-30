# リファクタリング方針

updated_at: 2026-04-30

## 目的

このリポジトリのリファクタリングは、Slack Socket Mode アプリとしての振る舞いを変えずに、KEDA launch コマンド周辺の責務を読みやすく保つことを目的とする。新機能追加や Slack/KEDA の実行時問題の調査は、この方針の主目的には含めない。

## 基本原則

- 挙動を変えない小さな変更を積み重ねる。
- ファイル分割は「何のユーザー操作フローを実現するか」を軸にする。
- 同じ Slack modal や message を扱う処理は、生成と読み取りを同じファイルに寄せる。
- 外部ライブラリの型や API を隠すだけの薄いラッパーは増やさない。
- `slack-go/slack` と `socketmode` の既存利用を前提にし、独自フレームワーク化しない。
- 設定読み込み、ロギング、プロセス起動は `main.go` に直接見える程度に保つ。
- テストは外部ライブラリの内部構造ではなく、このリポジトリが所有する振る舞いを確認する。
- コメントは責務や意図を補うために使う。短く自明な関数や構造体には無理に付けず、Slack ack、metadata、外部送信境界など誤読しやすい箇所には理由が分かるコメントを付ける。

## 分割軸

最上位の理解軸はユーザー操作フロー、コード配置の補助軸は Slack artifact のまとまりとする。

- ユーザー操作フロー: slash command、launch modal submit、change duration action、change duration modal submit、response posting など、ユーザーから見える操作の流れ。
- Slack artifact: launch modal、change duration modal、accepted response message、error response message など、同じ画面やメッセージとして生成・入力・metadata を共有する単位。

両者が衝突する場合は、Slack artifact の保守性を優先する。例えば launch request modal は slash command から開かれ、modal submit で読み取られるが、modal の block 定義、block/action ID、metadata、入力値の読み取りは同じ変更理由を持つため、別々のフロー用ファイルに分散させない。

ただし、artifact 単位の配置によってユーザー操作フローが読み取りづらくならないようにする。フロー全体を 1 ファイルに集約するのではなく、handler 登録順、関数名、テスト名、必要最小限のコメントで「slash command -> launch modal submit -> accepted response -> change duration」の流れが追える状態を保つ。

## 現在の責務境界

- `main.go`: 設定読み込み、Slack クライアント生成、Socket Mode handler の起動、共通イベントログ。
- `internal/kedalaunch/register.go`: `/launch` コマンドと関連 callback の登録、KEDA launcher client の組み立て。
- `internal/kedalaunch/slash_command.go`: `/launch` の受付、launch modal を開くところまでの入口処理。
- `internal/kedalaunch/launch_modal.go`: launch modal の生成、初回 modal metadata、modal submit の入力読み取りと検証、`LaunchRequest` 生成。
- `internal/kedalaunch/change_duration.go`: accepted message からの変更操作と変更 submit の処理フロー。
- `internal/kedalaunch/change_duration_modal.go`: change duration modal の生成、変更用 metadata、modal submit の入力読み取りと検証。
- `internal/kedalaunch/launch_request.go`: KEDA launcher への送信。
- `internal/kedalaunch/accepted_response.go`: 起動リクエスト成功時の Slack message artifact の生成。
- `internal/kedalaunch/slack_response.go`: `kedalaunch` 内で使う Slack ack と webhook response の小さな補助。
- Slack UI 部品と入力値取得の補助は、共通 package ではなく `kedalaunch` 内の unexported helper として扱う。

## 目標ファイル構造

この構造はリファクタリング時の目安であり、実装中により自然な責務境界が見つかった場合は、基本原則に反しない範囲で調整してよい。

- `main.go`: アプリ起動、設定読み込み、Slack client と Socket Mode handler の生成、共通イベントログの登録。
- `internal/kedalaunch/register.go`: `/launch` と関連 callback の登録、KEDA launcher client など依存の組み立て。
- `internal/kedalaunch/command.go`: `kedaLaunchCommand` とテスト用に差し替える依存インターフェース。
- `internal/kedalaunch/slash_command.go`: `/launch` の受付、ack、launch modal を開く入口処理。
- `internal/kedalaunch/launch_modal.go`: launch modal の block 定義、block/action ID、初回 modal metadata、入力値読み取り、submit 検証、`LaunchRequest` 生成。
- `internal/kedalaunch/launch_submission.go`: launch modal submit の受付、ack、KEDA launcher への送信呼び出し、送信結果に応じた response posting。
- `internal/kedalaunch/change_duration.go`: 変更ボタンと change duration modal submit の処理フロー。
- `internal/kedalaunch/change_duration_modal.go`: change duration modal の block 定義、変更用 metadata、入力値読み取り、submit 検証。
- `internal/kedalaunch/launch_request.go`: KEDA launcher への送信。
- `internal/kedalaunch/accepted_response.go`: accepted response message の生成。
- `internal/kedalaunch/slack_response.go`: Slack event ack と webhook response の小さな補助。利用箇所が `kedalaunch` だけの間は package 外へ出さない。

この構造では、ユーザー操作フローそのものを専用の巨大な `flow.go` に集約しない。フローの見通しは `register.go` の登録順と、各ファイルの責務名で表現する。

`register.go` の登録順は、ユーザーが体験する順序に寄せる。

1. `/launch` slash command
2. launch modal submit
3. change duration button
4. change duration modal submit

## 優先して整理する箇所

1. Slack UI 補助の境界を維持する。
   - 利用箇所が `kedalaunch` だけの間は、ローカルな unexported helper に置く。
   - 複数 feature で同じ UI 補助が必要になってから共通 package 化する。

2. ack 処理の一貫性を確認する。
   - slash command、block action、view submission で `ackIfPresent` の使い方をそろえる。
   - ack のタイミングは Slack の timeout を避けることを最優先にする。

3. metadata の責務を明確にする。
   - `private_metadata` は Slack modal callback 間の状態受け渡しとして扱う。
   - 初回 launch modal 用 metadata と変更操作用 metadata は、用途が違うため無理に統合しない。
   - metadata の encode/decode は、それを使う modal や message の近くに置く。
   - modal の block/action ID と入力値読み取りは、同じファイルで管理する。

4. response 生成と送信の境界を保つ。
   - 起動リクエスト成功時の Slack message artifact は `accepted_response.go` に置く。
   - KEDA launcher client への送信は `launch_request.go` に置き、送信結果を Slack にどう返すかは呼び出し側の submit flow に置く。
   - Slack ack と webhook response の小さな補助は `slack_response.go` に置く。
   - KEDA launcher client と Slack webhook は、テストしやすい依存注入を維持する。

5. `main.go` は起動配線に集中させる。
   - command ごとの詳細処理を `main.go` に戻さない。
   - ただし、一行程度の初期化を過剰に helper 化しない。

## やらないこと

- `/launch` の Slack App 側設定不一致や runtime timeout の調査を、このリファクタリングに混ぜない。
- `slack-go/slack` の block 構造や `socketmode` の handler map をテストで細かく検証しない。
- 将来の多数コマンド対応を想定しすぎて、現時点で不要な router package や framework 層を追加しない。
- KEDA launcher client の API 契約をこのリポジトリ内で再定義しない。
- README や環境設定を、コード整理の副作用で広く書き換えない。

## 検証方針

- 変更前後で `go test ./...` が通ることを確認する。
- Go build cache の権限問題が出る場合は、`GOCACHE=$(pwd)/.gocache` を使う。
- 既存テストを増やす場合は、以下のような repo-owned behavior を対象にする。
  - modal submit から正しい `LaunchRequest` が作られる。
  - invalid input が Slack view submission error になる。
  - launch submit が KEDA launcher に `LaunchRequest` を送る。
  - accepted response に変更ボタン用 metadata が含まれる。
  - 変更 submit が元の request id と scaled object を維持して duration だけを更新する。

## 実施順序

1. 現状テストを実行し、リファクタ前の基準を確認する。
2. Slack UI 補助と ack 処理のような小さい境界から整理する。
3. launch modal の生成、metadata、入力読み取り、検証を同じファイルへ寄せる。
4. 各ステップごとに `go test ./...` を実行する。
5. 挙動変更が必要だと分かった場合は、リファクタリングとは別タスクとして扱う。
