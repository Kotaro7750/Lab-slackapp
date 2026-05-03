# Implementation Plan

- [ ] 1. Foundation: ScaledObject 一覧取得の feature-local seam を整える
- [ ] 1.1 `keda-launcher-client` 境界に一覧取得を追加する
  - upstream client の一覧 API を feature-local interface と launcher wrapper から呼べるようにする。
  - launch / cancel と同じ timeout policy を一覧取得にも適用する。
  - 一覧取得の deadline を観測できるテストが追加され、gateway 単体で list support が確認できる状態にする。
  - _Requirements: 1.1, 3.3_

- [ ] 2. Core: launch modal を候補選択 UI に置き換える
- [ ] 2.1 ScaledObject 候補を namespace 単位で選べる modal を組み立てる
  - `ui` で `[]ScaledObject` を namespace group 付き dropdown option に変換する。
  - option value は namespace と name を含む JSON 文字列 contract に統一する。
  - `/launch` 初回 modal が自由記載ではなく、候補選択 UI と duration 入力を持つ形で構築される状態にする。
  - _Requirements: 1.2, 1.3_

- [ ] 2.2 候補選択 modal の submit 値を既存 `LaunchRequest` へ復元する
  - selected option の JSON value を decode して `ScaledObject` を復元する。
  - target 未選択、壊れた option value、invalid duration を field-level validation error として返す。
  - valid な modal submit から、既存と同じ shape の `LaunchRequest` が生成される状態にする。
  - _Requirements: 2.1, 2.3_

- [ ] 3. Integration: slash command と launch submit を新しい modal contract に接続する
- [ ] 3.1 slash command で候補取得結果に応じて modal か通知へ分岐する
  - ack 後に candidate 一覧を取得し、成功かつ non-empty の場合だけ modal を開く。
  - 一覧取得失敗時は modal を開かず、候補取得失敗を利用者へ通知する。
  - 候補ゼロ時は modal を開かず、選択可能な対象がないことを利用者へ通知する。
  - slash command 実行後に、候補ありなら modal、失敗/空なら ephemeral error のいずれかが必ず観測できる状態にする。
  - _Requirements: 1.1, 3.1, 3.2, 3.3_

- [ ] 3.2 launch submit を新しい parse 結果へ接続し、accepted response の挙動を維持する
  - handler が新しい `ParseLaunchModal` の返り値を使って launch 実行できるようにする。
  - accepted response artifact と response posting の既存挙動は変更しない。
  - 候補選択 modal の submit から accepted response まで、既存 `/launch` フローと同じ利用者通知が成立する状態にする。
  - _Depends: 2.2_
  - _Requirements: 2.1, 2.2, 2.3_

- [ ] 4. Validation: repo-owned behavior を回帰テストで固定する
- [ ] 4.1 UI と handler の回帰テストを候補選択フロー前提に更新する
  - UI テストで option group build、JSON value decode、未選択 validation を確認する。
  - handler テストで一覧取得成功、一覧取得失敗、候補ゼロ、launch submit success/validation failure を確認する。
  - 候補選択フローの主要な成功系・失敗系が、repo-owned behavior としてテストで観測できる状態にする。
  - _Requirements: 1.1, 1.2, 1.3, 2.1, 2.2, 2.3, 3.1, 3.2_

- [ ] 4.2 実行順序と timeout 前提を含む最終回帰を通す
  - gateway、UI、handler の変更が統合された状態でテストを整理し、不要な古い自由記載前提を残さない。
  - list timeout、ack-first、error notification の前提が崩れていないことを確認する。
  - `internal/kedalaunch` のテスト群で、この feature の追加後も launch flow の回帰が検知できる状態にする。
  - _Requirements: 3.3_
