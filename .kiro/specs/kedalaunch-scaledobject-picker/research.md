# Research & Design Decisions

## Summary
- **Feature**: `kedalaunch-scaledobject-picker`
- **Discovery Scope**: Extension
- **Key Findings**:
  - 既存 `/launch` フローでは `handler/slash_command.go` が modal を開く唯一の入口であり、一覧取得の追加点として最も自然である。
  - upstream `keda-launcher-scaler v0.1.4` の HTTP client は `ListScaledObjects(ctx)` を公開しており、この repo 側で独自 HTTP 契約を再実装する必要はない。
  - Slack の static select は typeahead を持ち、option group も使えるため、namespace ごとに候補を束ねると現在の要求を満たしやすい。

## Research Log

### 既存 `/launch` フローの拡張点
- **Context**: どこで候補取得し、どこで UI artifact を差し替えるのが既存境界に合うかを確認する必要があった。
- **Sources Consulted**:
  - `internal/kedalaunch/handler/slash_command.go`
  - `internal/kedalaunch/handler/launch_submission.go`
  - `internal/kedalaunch/ui/launch_modal.go`
  - `internal/kedalaunch/ui/helper.go`
  - `internal/kedalaunch/handler/handler_test.go`
  - `.kiro/steering/tech.md`
  - `.kiro/steering/structure.md`
- **Findings**:
  - slash command は ack 後に `metadata.BuildLaunchModal()` を呼んでおり、ここに候補取得結果を渡す形が最小変更となる。
  - launch submit 側は modal state を `domainclient.LaunchRequest` に変換する責務を already own している。
  - `ui/helper.go` は現状 text input 前提の state 読み取りだけを持つため、select の selected option を読む helper 追加が必要になる。
- **Implications**:
  - 一覧取得は `HandleSlashCommand` に追加し、modal の生成と parse は `ui/launch_modal.go` に留める。
  - 新しい共通 package は作らず、既存 `ui` と `handler` の責務に寄せる。

### upstream 一覧 API 契約の確認
- **Context**: この feature が依存してよい upstream 契約を明確化する必要があった。
- **Sources Consulted**:
  - `/Users/koutarou/go/pkg/mod/github.com/!kotaro7750/keda-launcher-scaler@v0.1.4/pkg/client/http/client.go`
  - `/Users/koutarou/develop/Lab-slackapp/go.mod`
- **Findings**:
  - この repo は既に `github.com/Kotaro7750/keda-launcher-scaler v0.1.4` を利用している。
  - upstream HTTP client は `ListScaledObjects(ctx)` を公開し、`[]domainclient.ScaledObject` を返す。
  - 返却要素は namespace/name の domain 型であり、launch modal に必要な情報はそれだけで足りる。
- **Implications**:
  - feature-local client seam に `ListScaledObjects` を追加し、launch と cancel と同じ timeout policy を適用する。
  - UI は upstream response body 全体を知る必要がなく、`[]domainclient.ScaledObject` だけを受け取る。

### Slack select menu の制約確認
- **Context**: static dropdown で requirements を満たせるか、件数制約や UX 前提を確認する必要があった。
- **Sources Consulted**:
  - [Slack Developer Docs: Select menu element](https://docs.slack.dev/reference/block-kit/block-elements/select-menu-element/)
  - `/Users/koutarou/go/pkg/mod/github.com/slack-go/slack@v0.23.0/block.go`
  - `/Users/koutarou/go/pkg/mod/github.com/slack-go/slack@v0.23.0/block_element_test.go`
- **Findings**:
  - static select は options を modal 定義時に渡せる。
  - 公式 docs では static select の options は最大 100、option groups は最大 100 とされている。
  - `slack-go` の `BlockAction` は `SelectedOption` を持つため、view state から選択値を復元できる。
  - select menu 自体に typeahead があるため、現要求の範囲では external select を導入しなくても候補選択 UX は成立する。
- **Implications**:
  - 今回は modal open 前に一覧取得する static select を採用する。
  - namespace ごとの option group を使うことで、利用者に namespace/name を識別可能に見せやすい。
  - group 数や option 数の上限を超える運用が必要になった場合は再設計が必要であり、revalidation trigger に含める。

## Architecture Pattern Evaluation

| Option | Description | Strengths | Risks / Limitations | Notes |
|--------|-------------|-----------|---------------------|-------|
| slash command 前段で一覧取得し static select modal を開く | handler が一覧を取得し、ui が静的候補を描画する | 既存責務に一致、追加 callback 不要、submit parse が単純 | 候補が多すぎる場合は Slack 制約に触れる | 採用 |
| external select を使って入力時に候補ロードする | Slack の option load で動的に候補取得する | 大量候補に伸ばしやすい | interactivity endpoint 設計が増え、現 Socket Mode フローから逸れる | 不採用 |
| 一覧取得失敗時に自由記載 modal へフォールバックする | 既存 modal を残して使い分ける | 障害時も launch 自体は継続しやすい | 今回の要求と境界に反し、validation を二重化する | 不採用 |

## Design Decisions

### Decision: namespace group 付き static select を使う
- **Context**: 利用者が namespace と ScaledObject を識別しつつ、一つの modal で対象選択を完結させたい。
- **Alternatives Considered**:
  1. 1 本の options 配列に `namespace/name` を平坦に並べる
  2. namespace ごとの option group で static select を構築する
- **Selected Approach**: namespace を group label、ScaledObject 名を option text とする static select を採用する。
- **Rationale**: Requirement 1.3 の識別性を UI 上で自然に満たせる。Slack の typeahead もそのまま使える。
- **Trade-offs**: 候補数上限は Slack 制約に従う。大規模一覧の最適化は current scope 外となる。
- **Follow-up**: option 数や namespace group 数が制約に近づく場合の実運用確認をテストと validation に含める。

### Decision: select value は namespace と name を含む JSON 文字列にする
- **Context**: submit 時に選択された候補から `LaunchRequest` を復元する必要がある。区切り文字連結は参照形式として脆い。
- **Alternatives Considered**:
  1. JSON 文字列を option value に入れる
  2. `namespace/name` 形式の参照文字列を使う
- **Selected Approach**: option value は `{"namespace":"...","name":"..."}` 相当の JSON 文字列とし、ui 側で decode して `ScaledObject` を復元する。
- **Rationale**: field 境界が明示され、区切り文字の扱いに依存しない。decode failure も一意に扱える。
- **Trade-offs**: 参照文字列より少し長くなるが、現スコープのデータ量では問題になりにくい。
- **Follow-up**: decode error は field validation error として扱い、submit を拒否する。

### Decision: 一覧取得失敗と候補ゼロは modal を開かずに終了する
- **Context**: requirements で自由記載フォールバックを採らず、失敗時挙動も明示された。
- **Alternatives Considered**:
  1. 失敗時も modal を開く
  2. 失敗または空結果では modal を開かず、ephemeral error を返す
- **Selected Approach**: `HandleSlashCommand` で ack 後に一覧取得を行い、失敗や空結果では modal を開かず短い利用者通知だけを返す。
- **Rationale**: modal 内に操作不能状態を持ち込まず、要件 3.1 と 3.2 を最短で満たせる。
- **Trade-offs**: 起動操作は一覧 API 可用性に依存する。
- **Follow-up**: handler テストで `openViewCalls == 0` と error message を検証する。

## Risks & Mitigations
- Slack option/group 上限を超える ScaledObject 数が来るリスク — namespace group 採用を前提にしつつ、制約超過は再検証トリガーとして扱う。
- 一覧取得結果と submit 時点の upstream 状態がずれるリスク — この feature は選択 UI のみを所有し、launch 失敗時は既存 `Launch request failed.` 通知に委ねる。
- select parse と text input parse が混在してテストが壊れるリスク — `ui/launch_modal_test.go` と `handler/handler_test.go` を候補選択前提へ更新し、JSON value decode を含む repo-owned behavior だけを確認する。

## References
- [Slack Developer Docs: Select menu element](https://docs.slack.dev/reference/block-kit/block-elements/select-menu-element/) — static select と option group の制約確認
