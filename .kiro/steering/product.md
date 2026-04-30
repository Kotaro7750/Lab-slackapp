# Product Overview

updated_at: 2026-04-30

## Purpose

Lab Slack App は、Slack から KEDA launcher scaler へ launch request を送るための小さな Socket Mode アプリである。Slack の `/launch` コマンドを入口にして、利用者が Slack modal で対象 ScaledObject と duration を入力し、KEDA 側の receiver へリクエストを送信する。

このアプリは Slack UI と KEDA launcher client の接続点に集中する。KEDA launcher の API 契約や Kubernetes 側の制御ロジックは、このリポジトリ内で再定義しない。

## Core Capabilities

- `/launch` slash command から KEDA launch request の入力 modal を開く。
- modal submit から `LaunchRequest` を組み立て、KEDA launcher receiver へ送信する。
- 送信結果を Slack の ephemeral response として返す。
- accepted response から同じ request id / ScaledObject を保ったまま duration だけ変更して再送できる。
- Slack の timeout を避けるため、interactive event は先に ack し、外部送信は handler の外側で行う。

## Target Use Cases

- Slack だけを操作面として、一時的な KEDA scaling request を発行したい。
- `/launch` 実行者にだけ request の結果や変更ボタンを見せたい。
- 一度 accepted された request の duration を、元の対象を維持したまま変更したい。

## Value Proposition

Slack 上の短い対話で KEDA launch request を扱えることが価値である。アプリ自体は薄く保ち、Slack artifact の組み立て、入力検証、KEDA launcher client への橋渡し、response posting に責務を限定する。

## Out of Scope

- Slack App 側の設定不一致や workspace 固有の runtime 調査を、通常のコード整理に混ぜない。
- KEDA launcher receiver の API や Kubernetes controller の振る舞いをこのリポジトリで所有しない。
- 将来の多数コマンド対応を先取りして、現時点で不要な router framework を作らない。
