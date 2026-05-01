# Implementation Plan

- [ ] 1. Foundation: upstream delete 契約と request gateway の前提を揃える
- [x] 1.1 `keda-launcher-scaler v0.1.4` へ依存を更新し、delete 契約を feature seam に露出する
  - `go.mod` と `go.sum` を更新し、`DeleteRequest` / `DeletedRequest` をこの repo から参照できるようにする。
  - `kedaLauncher` interface と request gateway に cancel 用の呼び出し口を追加し、launch と同じ timeout policy を共有させる。
  - 完了時には `internal/kedalaunch` から upstream delete 契約を使った実装がコンパイル可能な状態になる。
  - _Requirements: 2.1, 2.3_

- [ ] 2. Core: accepted request artifact と metadata を cancel 対応へ広げる
- [x] 2.1 accepted request metadata を follow-up action 共通契約へ整理する
  - accepted response が所有する metadata に request id、ScaledObject、response URL を集約し、change duration 側がそれを引き続き消費できるようにする。
  - metadata decode failure を利用者通知へつなげられる前提を作り、request target を follow-up action で上書きしない形に保つ。
  - 完了時には change duration が追加の再入力なしで従来どおり request id と ScaledObject を維持できる。
  - _Requirements: 1.2, 3.1, 3.3_

- [x] 2.2 accepted response に cancel 導線と canceled 状態の message artifact を追加する
  - accepted response に change duration と並ぶ cancel button を追加する。
  - cancel 成功後の replaced message を定義し、active request 用の follow-up action が残らないようにする。
  - 完了時には accepted response から cancel を開始でき、成功後は canceled 状態だけが Slack 上に残る。
  - _Requirements: 1.1, 2.2, 3.3_

- [ ] 3. Integration: cancel action を `/launch` フローへ接続する
- [x] 3.1 cancel action handler を登録し、ack 後に delete を実行する
  - `/launch` 関連 callback の登録順に cancel action を追加し、accepted response 配下の flow として扱う。
  - cancel button payload から metadata を読み取り、Slack interactive timeout を避けるため upstream delete 前に ack を返す。
  - metadata が壊れている場合は delete を呼ばず、利用者へ操作失敗を返す。
  - 完了時には cancel 操作が `/launch` フロー内で一度だけ delete 実行へ進む。
  - _Requirements: 1.3, 2.1, 2.3, 3.1_

- [x] 3.2 cancel の成功結果と失敗結果を Slack 応答へ変換する
  - delete 成功時は元の accepted response を replaced canceled message へ置き換える。
  - delete 失敗時は request が未取消であることを伝える ephemeral error を返し、元 message は残す。
  - webhook response posting failure は log に残し、delete 結果の扱いとは分離する。
  - 完了時には利用者が Slack 上で cancel の成否と request の残存状態を判断できる。
  - _Requirements: 2.2, 3.2, 3.3_

- [ ] 4. Validation: repo-owned behavior と回帰を確認する
- [x] 4.1 accepted response と cancel flow の repo-owned test を追加・更新する
  - accepted response test を拡張し、change と cancel の両 button metadata を検証する。
  - cancel request 変換と canceled success message の artifact を検証するテストを追加する。
  - change duration 既存テストを metadata 共通化後の契約に合わせて更新する。
  - 完了時には cancel 導線、metadata 維持、canceled artifact がテストで可視化される。
  - _Requirements: 1.1, 1.2, 2.2, 3.1, 3.3_

- [x] 4.2 launch / change / cancel の回帰をまとめて確認する
  - repo-local `GOCACHE` を使った `go test ./...` で既存 launch、change duration、追加された cancel flow をまとめて検証する。
  - 失敗時は dependency update、metadata 共通化、cancel integration のどこで壊れたか切り分けられるようにする。
  - 完了時にはテストスイートが通り、既存 `/launch` 操作が cancel 追加で回帰していないことを確認できる。
  - _Requirements: 1.3, 2.1, 2.3, 3.2_
