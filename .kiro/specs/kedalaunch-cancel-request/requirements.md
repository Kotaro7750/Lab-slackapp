# Requirements Document

## Project Description (Input)
# Brief: kedalaunch-cancel-request

## Problem
Slack から KEDA launch request を送った利用者は、accepted 後に duration を変更できる一方で、不要になった request を Slack 上から取り消せない。`keda-launcher-scaler` client は 2026年4月30日公開の `v0.1.4` で delete API を公開したため、このアプリ側もその導線を持てるようにしたい。

## Current State
現在の Lab Slack App は `github.com/Kotaro7750/keda-launcher-scaler v0.1.3` を利用し、`LaunchRequest` の送信だけを行っている。accepted response には duration 変更ボタンだけがあり、request のキャンセルは Slack から実行できない。

## Desired Outcome
依存を `v0.1.4` へ更新した上で、accepted response から `Cancel` 操作を実行できるようにする。利用者は同じ request id / ScaledObject を使って delete API を呼び出せ、成功時は Slack 上で request が取り消されたことを確認できる。

## Approach
accepted response に `Cancel` ボタンを追加し、既存の metadata に request cancellation に必要な情報を保持する。Slack callback は既存の `/launch` フロー配下で処理し、KEDA への delete 呼び出しと Slack への結果通知の責務を分けたまま実装する。

比較した方針:

1. accepted response に直接 `Cancel` ボタンを追加する
   - 既存の change duration と同じ導線上に置ける
   - 追加 UI が最小で、現在の metadata 運搬方式を再利用できる
2. 取消専用 modal を挟んで confirm する
   - 誤操作防止はしやすい
   - ただし Slack artifact と callback が増え、小さな機能追加としては過剰
3. 一覧 API も同時に使って launch modal を候補選択化する
   - 今回追加された新 API を広く活用できる
   - ただしキャンセル機能よりスコープが広がり、今回の要求から外れる

この spec では 1 を採用する。

## Scope
- **In**: `keda-launcher-scaler` を `v0.1.4` へ更新すること
- **In**: accepted response に cancel 導線を追加すること
- **In**: cancel callback で delete API を呼び、Slack に結果を返すこと
- **In**: cancel フローに必要な metadata とテストを追加すること
- **Out**: launch modal の入力 UI を一覧 API ベースの候補選択に変えること
- **Out**: KEDA launcher receiver 側の API 契約や削除ロジックをこの repo で変更すること
- **Out**: Slack App 全体の複数コマンド対応や大規模な構造変更

## Boundary Candidates
- 依存更新と client interface 追従
- accepted response artifact と cancel action の UI 導線
- delete API 呼び出しと Slack response posting

## Out of Boundary
- `GET /scaledobjects` を使った launch 対象一覧表示
- request 一覧表示、履歴表示、監査ログ表示
- KEDA launcher receiver の not found 判定や削除意味論の変更

## Upstream / Downstream
- **Upstream**: `github.com/Kotaro7750/keda-launcher-scaler v0.1.4` の `pkg/client` と `pkg/client/http`
- **Downstream**: Slack `/launch` 利用者の request lifecycle 操作、将来の request 管理 UI 拡張

## Existing Spec Touchpoints
- **Extends**: なし
- **Adjacent**: `internal/kedalaunch` の launch submission、change duration、accepted response、Slack callback 処理

## Constraints
既存の責務分割を保ち、KEDA 送信処理と Slack への通知処理を混ぜない。Slack interactive event は先に ack する。Go コードのコメントは英語、Markdown は日本語で書く。今回のスコープでは `GET /scaledobjects` は採用しない。

## Introduction

この機能は、Slack 上で accepted になった KEDA launch request を、同じ `/launch` 利用フローの中で取り消せるようにする。利用者は request id や ScaledObject を再入力せずに cancel を実行でき、結果を Slack 上で確認できる必要がある。

## Boundary Context

- **In scope**: accepted response への cancel 導線追加、accepted 済み request の cancel 実行、Slack 上での成功/失敗結果通知
- **Out of scope**: launch 対象一覧 UI、request 履歴表示、receiver 側の削除意味論変更、複数コマンド対応への拡張
- **Adjacent expectations**: 隣接する KEDA launcher client / receiver は既存 request id と ScaledObject に対する cancel を受け付け、この機能はその結果を `/launch` の利用者に見える形で返す

## Requirements

### Requirement 1: Cancel導線
**Objective:** As a `/launch` 利用者, I want accepted 済み request から直接 cancel を始めたい, so that 不要になった request をすぐ取り消せる

#### Acceptance Criteria
1. When KEDA launch request が accepted されて Slack response が表示される, the Lab Slack App shall 同じ response 内に request lifecycle 操作として cancel 導線を表示する
2. When 利用者が accepted response から cancel を始める, the Lab Slack App shall request id と対象 ScaledObject を既存 response の文脈から引き継ぎ、利用者に再入力を要求しない
3. Where accepted request の follow-up 操作が提供される, the Lab Slack App shall cancel 操作を既存の `/launch` フロー配下に保ち、別の request 一覧画面や候補選択フローを要求しない

### Requirement 2: Cancel実行と成功結果
**Objective:** As a `/launch` 利用者, I want accepted 済み request を Slack から取り消したい, so that 不要なスケーリング要求を終了できる

#### Acceptance Criteria
1. When 利用者が cancel を実行する, the Lab Slack App shall accepted response が表していた request id と ScaledObject に対して cancel を実行する
2. When 隣接する KEDA launcher 側が cancel を受け付ける, the Lab Slack App shall その request が取り消されたことを Slack 上で利用者に通知する
3. While cancel 結果を待っている, the Lab Slack App shall Slack interactive timeout により利用者が再試行を強いられないように応答する

### Requirement 3: Cancel失敗時の利用者通知
**Objective:** As a `/launch` 利用者, I want cancel できなかった場合も状態を判断したい, so that request が未取消のままかどうかを把握できる

#### Acceptance Criteria
1. If cancel 実行に必要な request 文脈が accepted response から復元できない, the Lab Slack App shall cancel を実行せず、操作を完了できなかったことを利用者に通知する
2. If 隣接する KEDA launcher 側で cancel が失敗または拒否される, the Lab Slack App shall request が取り消されていないことを利用者に通知する
3. The Lab Slack App shall cancel の成功結果と失敗結果のどちらでも、元の `/launch` 利用者が判断できる Slack 応答を返す
