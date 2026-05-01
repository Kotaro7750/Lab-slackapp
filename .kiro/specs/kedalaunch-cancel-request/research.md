# Research & Design Decisions

## Summary
- **Feature**: `kedalaunch-cancel-request`
- **Discovery Scope**: Extension
- **Key Findings**:
  - `github.com/Kotaro7750/keda-launcher-scaler v0.1.4` は `pkg/client.Client` に `DeleteRequest` と `DeletedRequest` を追加し、HTTP `DELETE /scaledobjects/{namespace}/{name}/requests/{requestId}` を公開している。
  - 既存の `internal/kedalaunch` は accepted response のボタン metadata を change duration modal へ渡す構造をすでに持っており、cancel も同じ accepted response 起点で拡張できる。
  - この repo の既存方針では、KEDA 送信処理と Slack 通知処理を分け、Slack artifact は artifact の近くに保つことが重要である。

## Research Log

### `v0.1.4` の client 契約確認
- **Context**: cancel 設計では upstream client の公開 interface と HTTP 契約が boundary に直結する。
- **Sources Consulted**:
  - `https://github.com/Kotaro7750/keda-launcher-scaler/tags`
  - `https://raw.githubusercontent.com/Kotaro7750/keda-launcher-scaler/v0.1.4/pkg/client/client.go`
  - `https://raw.githubusercontent.com/Kotaro7750/keda-launcher-scaler/v0.1.4/pkg/client/http/client.go`
  - `https://raw.githubusercontent.com/Kotaro7750/keda-launcher-scaler/v0.1.4/internal/common/contracts/receivers/http/openapi.yaml`
- **Findings**:
  - `v0.1.4` tag は 2026-04-30 付けで、commit message は `feat: add scaledobject listing and request deletion`。
  - `pkg/client.Client` は `Launch` に加えて `ListScaledObjects` と `DeleteRequest` を持つ。
  - `DeleteRequest` は `requestId` と `ScaledObject` を入力に取り、成功時は `DeletedRequest` を返す。
  - HTTP client は `404` を domain error として返し、成功時は `200` と deleted response body を返す。
- **Implications**:
  - Lab Slack App は独自 HTTP 実装を追加せず、upstream client の `DeleteRequest` を採用する。
  - cancel 成功 message は `DeletedRequest` の `effectiveStart` / `effectiveEnd` を表示できる。
  - `404` を含む delete 失敗は repo 側で receiver 意味論を再解釈せず、Slack 利用者へ未取消を通知する境界に留める。

### 既存 `kedalaunch` フローとの整合
- **Context**: cancel を既存の file boundary と interaction flow に合わせる必要がある。
- **Sources Consulted**:
  - `internal/kedalaunch/accepted_response.go`
  - `internal/kedalaunch/change_duration.go`
  - `internal/kedalaunch/change_duration_modal.go`
  - `internal/kedalaunch/launch_request.go`
  - `internal/kedalaunch/command.go`
  - `internal/kedalaunch/slack_response.go`
  - `.kiro/steering/refactoring.md`
  - `.kiro/steering/tech.md`
  - `.kiro/steering/structure.md`
- **Findings**:
  - accepted response は現在 `kedaRequestMetadata` をボタンに埋め込み、change duration modal submit が同じ request id と ScaledObject を維持している。
  - KEDA 呼び出しは `launch_request.go` の timeout 付き helper に閉じており、Slack webhook posting は caller 側で行っている。
  - `slack_response.go` は ack と ephemeral error posting の小さな補助だけを持つ。
  - file boundary の方針は、ユーザーフローを主軸にしつつ、同じ Slack artifact を同じファイル近傍に保つことである。
- **Implications**:
  - cancel でも KEDA delete は helper に閉じ込め、Slack 成功/失敗通知は cancel flow 側が持つ。
  - accepted response 由来の metadata は change 専用名のままにせず、accepted request lifecycle metadata として扱う方が自然である。
  - modal を増やさず block action で完結させる方が、既存 boundary と今回の小さな機能追加に合う。

### テスト価値と検証面
- **Context**: この repo は外部ライブラリ挙動の薄いテストを避ける方針である。
- **Sources Consulted**:
  - `internal/kedalaunch/accepted_response_test.go`
  - `internal/kedalaunch/launch_modal_test.go`
  - `internal/kedalaunch/change_duration_modal_test.go`
  - `.kiro/steering/tech.md`
- **Findings**:
  - 既存テストは metadata 維持、request build、validation など repo-owned behavior に寄っている。
  - fake client の呼び出し回数だけを確かめるテストはこの repo では価値が低いと扱われている。
- **Implications**:
  - cancel でも message metadata、delete request 変換、成功 message artifact、失敗通知条件など純粋関数寄りの seam を主に検証する。
  - block action handler 全体の Slack SDK 内部依存テストは避ける。

## Architecture Pattern Evaluation

| Option | Description | Strengths | Risks / Limitations | Notes |
|--------|-------------|-----------|---------------------|-------|
| accepted response に cancel button を追加し block action で delete する | 既存 accepted response から直接 cancel する | 追加 artifact が最小、既存 metadata を再利用できる、既存 flow に沿う | 誤操作確認 modal はない | 採用 |
| cancel 確認 modal を追加する | button から modal を開いて confirm する | 誤操作確認ができる | artifact と callback が増え、今回の scope には過剰 | 不採用 |
| generic request lifecycle dispatcher を新設する | change と cancel を共通 dispatcher に集約する | 将来の操作追加に備えやすい | 現時点で abstraction が先走り、責務が広がる | 不採用 |

## Design Decisions

### Decision: accepted response metadata を request lifecycle 共通契約にする
- **Context**: change duration と cancel の両方が、accepted response から同じ request context を受け継ぐ必要がある。
- **Alternatives Considered**:
  1. change 用 metadata をそのまま流用する
  2. accepted response 用の共通 metadata 契約へ寄せる
- **Selected Approach**: accepted response が所有する request lifecycle metadata を共通契約とし、change と cancel の両方がそれを読む。
- **Rationale**: accepted response が follow-up 操作の起点であり、metadata ownership を message artifact 側へ寄せる方が boundary が明瞭になる。
- **Trade-offs**: 小さな rename や encode/decode の配置見直しは必要だが、cancel 追加後の責務が読みやすい。
- **Follow-up**: 実装時に metadata 必須項目を `requestId`, `namespace`, `name`, `responseURL` に保ち、change 専用の `duration` は optional 扱いにする。

### Decision: delete 呼び出しは launch と同じ gateway seam に閉じる
- **Context**: KEDA 送信と Slack 通知の責務分離は steering で明示されている。
- **Alternatives Considered**:
  1. cancel handler から upstream client を直接呼ぶ
  2. `launch_request.go` に delete 用 helper を追加する
- **Selected Approach**: timeout 管理を含む upstream 呼び出しは `launch_request.go` の helper に集約し、caller は Slack 通知だけを担当する。
- **Rationale**: 既存の `launchRequest` と同じ境界を保てる。
- **Trade-offs**: helper 名が launch 専用に見えやすくなるため、file responsibility の説明を design で補う必要がある。
- **Follow-up**: `kedaLauncher` interface を `DeleteRequest` まで拡張する。

### Decision: cancel 成功は元 message を置き換える
- **Context**: cancel 成功後に active request 用の follow-up button が残ると、利用者に誤った再操作余地を見せる。
- **Alternatives Considered**:
  1. 新しい ephemeral message を追加投稿する
  2. 元の accepted response を canceled 状態へ置き換える
- **Selected Approach**: success は `ReplaceOriginal` を使って元 message を canceled 状態へ置き換える。
- **Rationale**: 同じ request lifecycle message の最終状態として自然で、二重 cancel の誤解も減る。
- **Trade-offs**: 元の accepted state は置換されるが、Slack 上で request の現在状態を一意に示しやすい。
- **Follow-up**: 失敗時は置換せず、元 message を残して retry や change を継続可能にする。

## Risks & Mitigations
- accepted response metadata の rename や移動で change duration が壊れるリスク — change 既存テストを維持し、metadata decode の後方互換を保つ。
- delete 失敗時の Slack 表示が曖昧になるリスク — success と failure の message を分け、failure では request が未取消であることを明示する。
- upstream v0.1.4 への依存更新で interface mismatch が起きるリスク — `go.mod` 更新と `go test ./...` を同時に行い、`kedaLauncher` seam で compile error を先に顕在化させる。

## References
- [Kotaro7750/keda-launcher-scaler repository](https://github.com/Kotaro7750/keda-launcher-scaler) — upstream project root
- [v0.1.4 tag](https://github.com/Kotaro7750/keda-launcher-scaler/tags) — request deletion added on 2026-04-30
- [pkg/client/client.go at v0.1.4](https://raw.githubusercontent.com/Kotaro7750/keda-launcher-scaler/v0.1.4/pkg/client/client.go) — transport-agnostic client contract
- [pkg/client/http/client.go at v0.1.4](https://raw.githubusercontent.com/Kotaro7750/keda-launcher-scaler/v0.1.4/pkg/client/http/client.go) — HTTP delete behavior and error mapping
- [openapi.yaml at v0.1.4](https://raw.githubusercontent.com/Kotaro7750/keda-launcher-scaler/v0.1.4/internal/common/contracts/receivers/http/openapi.yaml) — canonical HTTP API contract
