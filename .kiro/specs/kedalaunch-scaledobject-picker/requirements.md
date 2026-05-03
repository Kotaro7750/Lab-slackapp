# Requirements Document

## Project Description (Input)
# Brief: kedalaunch-scaledobject-picker

## Problem
`/launch` を使う利用者は、現在の modal で namespace と ScaledObject 名を自由記載しなければならず、入力ミスや存在しない対象の指定が起こりうる。`keda-launcher-scaler v0.1.4` では ScaledObject 一覧取得 API が追加されたため、Slack 側でも既知の候補から選べる導線にしたい。

## Current State
現在の Lab Slack App は `/launch` 受信後に固定の modal を開き、`ui.BuildLaunchModal()` が namespace、ScaledObject name、duration の自由記載入力を組み立てている。handler は初回 modal 表示時に KEDA launcher へ問い合わせを行わず、submit 時にだけ入力値を `LaunchRequest` として送信している。

## Desired Outcome
`/launch` 実行時に handler が KEDA launcher から ScaledObject 一覧を取得し、その結果を使って modal 上で launch 対象をドロップダウン選択できるようにする。利用者は既知の対象から選ぶだけで launch request を作成でき、UI 組み立てと外部 API 呼び出しの責務分離は維持される。

## Approach
初回 slash command handler が `ListScaledObjects` を呼び、modal 構築に必要な候補一覧を `ui` へ渡す。`ui` は受け取った一覧を Slack option に変換して modal を構築し、submit 時は選択された値と duration を解釈して `LaunchRequest` を組み立てる。

比較した方針:

1. handler で一覧取得し、ui は渡された候補で静的ドロップダウンを組み立てる
   - 既存の `handler` と `ui` の責務境界にそのまま乗る
   - modal build 側は外部 I/O を持たず、Slack artifact の構築に集中できる
2. namespace と ScaledObject を段階的に選ばせる複数 modal に分ける
   - 候補数が多い場合の整理には向く
   - ただし callback と metadata が増え、現状の小さな `/launch` フローには過剰
3. 一覧取得失敗時だけ自由記載にフォールバックする
   - launch を止めない利点はある
   - UI と validation の二重化で境界が曖昧になる

この spec では 1 を採用する。

## Scope
- **In**: `keda-launcher-scaler` の一覧取得 API を `/launch` 初回 handler から呼び出せるようにすること
- **In**: launch modal を自由記載から ScaledObject 候補のドロップダウン中心 UI に変更すること
- **In**: ui package は渡された一覧を使って modal を構築し、submit 時は選択値から request を復元すること
- **In**: 一覧取得成功時と失敗時の Slack 応答方針、および関連テストを定義すること
- **Out**: accepted response の cancel / duration change フローを作り直すこと
- **Out**: KEDA launcher receiver 側の API 仕様や一覧内容の意味論を変更すること
- **Out**: request 一覧表示、検索 UI、複数段階 wizard 化など大きな UX 拡張

## Boundary Candidates
- slash command handler における一覧取得と失敗時応答
- ui package における dropdown modal の構築と submit parsing
- keda launcher client 境界への一覧取得メソッド追加と timeout / error handling

## Out of Boundary
- ScaledObject 候補の検索やページング
- Slack App 全体のコマンド体系や router の変更
- KEDA launcher 側で一覧に含める対象のフィルタリングロジック追加

## Upstream / Downstream
- **Upstream**: `github.com/Kotaro7750/keda-launcher-scaler v0.1.4` の `ListScaledObjects` API と HTTP client 実装
- **Downstream**: `/launch` 利用者の入力体験改善、将来の launch 対象選択 UX の拡張

## Existing Spec Touchpoints
- **Extends**: なし
- **Adjacent**: `kedalaunch-cancel-request` の accepted response lifecycle、`internal/kedalaunch` の slash command / launch modal / launcher client

## Constraints
handler が外部 API 呼び出しを担当し、ui package は渡された一覧を用いて Slack modal を構築する責務に留める。Slack interactive event の ack ルールは維持し、Go コードのコメントは英語、Markdown は日本語で書く。現時点では複数 modal に分割せず、単一の `/launch` 初回 modal の改善に閉じる。

## Introduction

この機能は、`/launch` の初回 modal で起動対象を自由記載ではなく候補選択できるようにする。利用者は KEDA launcher が現在認識している ScaledObject から対象を選び、入力ミスを避けながら launch request を送れる必要がある。

## Boundary Context

- **In scope**: `/launch` 開始時の候補取得、候補ベースの launch modal 表示、選択済み対象での launch request 作成、候補取得失敗時や候補なし時の利用者通知
- **Out of scope**: accepted response の cancel / duration change フロー変更、自由記載入力へのフォールバック、候補検索や複数段階 wizard、KEDA launcher 側の一覧生成ロジック変更
- **Adjacent expectations**: 隣接する KEDA launcher 側は launch 対象として選択可能な ScaledObject 一覧を返し、この機能はその一覧を `/launch` 利用者が選択できる形で提示する

## Requirements

### Requirement 1: Launch対象の候補選択
**Objective:** As a `/launch` 利用者, I want 起動対象を候補から選びたい, so that namespace や ScaledObject 名を手入力せずに正しい対象を選べる

#### Acceptance Criteria
1. When 利用者が `/launch` を実行する, the Lab Slack App shall launch 対象として選択可能な ScaledObject 候補を含む modal を表示する
2. When launch modal が表示される, the Lab Slack App shall namespace と ScaledObject 名の自由記載を要求せず、候補選択として対象を示す
3. Where launch 対象候補が表示される, the Lab Slack App shall 利用者がどの namespace と ScaledObject を選ぶのか識別できる表示を行う

### Requirement 2: 選択済み対象でのlaunch実行
**Objective:** As a `/launch` 利用者, I want 選んだ対象に対してそのまま launch request を送りたい, so that 候補選択から送信までを一つの `/launch` フローで完結できる

#### Acceptance Criteria
1. When 利用者が launch modal で対象候補と duration を送信する, the Lab Slack App shall 選択された対象に対する launch request を作成する
2. When 選択済み対象で launch request が受け付けられる, the Lab Slack App shall その対象に対して request が accepted されたことを利用者へ通知する
3. If 利用者が対象候補を選ばずに送信しようとする, the Lab Slack App shall 対象選択が必要であることを利用者へ示し、未完了の送信を受け付けない

### Requirement 3: 候補を提示できない場合の利用者通知
**Objective:** As a `/launch` 利用者, I want 候補を選べない理由を把握したい, so that launch をやり直すべきか別途確認すべきか判断できる

#### Acceptance Criteria
1. If launch 対象候補の取得に失敗する, the Lab Slack App shall launch modal を開かず、利用者に候補を取得できなかったことを通知する
2. If launch 対象候補の取得結果が空である, the Lab Slack App shall launch modal を開かず、選択可能な ScaledObject が存在しないことを利用者に通知する
3. While launch 対象候補の取得結果を待っている, the Lab Slack App shall Slack interactive timeout により利用者が再試行を強いられないように応答する
