# Project Structure

updated_at: 2026-04-30

## Organization Philosophy

小さな Go Slack App として、起動配線と feature logic を明確に分ける。最上位の理解軸はユーザー操作フロー、補助軸は Slack modal / message などの artifact 単位とする。

新しいファイルは「どのユーザー操作や Slack artifact を実現するか」で命名・配置する。単なる helper 形状や外部ライブラリ wrapper だけを理由に package を増やさない。

## Directory Patterns

### Startup Wiring

**Location**: repository root, mainly `main.go`  
**Purpose**: 設定読み込み、Slack client と Socket Mode handler の生成、共通 Socket Mode event のログ、アプリの lifecycle 管理。  
**Pattern**: command ごとの詳細処理は `main.go` に戻さず、feature package の `Register` に渡す。

### Feature Package

**Location**: `internal/<feature>/`  
**Purpose**: Slack command / callback / response posting など、1 つの user-facing feature に属する処理をまとめる。  
**Example**: `internal/kedalaunch` は `/launch`、launch modal submit、change duration、response posting を所有する。

### Steering

**Location**: `.kiro/steering/`  
**Purpose**: 長期的な project memory。実装パターン、責務境界、技術判断を記録する。  
**Pattern**: 仕様や自動化 metadata の一覧ではなく、新しいコードが従うべき判断基準を書く。

## Naming Conventions

- **Files**: snake_case。ユーザー操作フローや Slack artifact を表す名前にする。
- **Packages**: 小文字の単一 domain 名。現時点では feature ごとの `internal/kedalaunch` を基本にする。
- **Handlers**: `handle<Flow>` の形で Socket Mode event の入口を表す。
- **Builders/Parsers**: Slack artifact は `build<Artifact>`、入力読み取りは `parse<Submission>` の形に寄せる。
- **Metadata**: Slack `private_metadata` 用の型は、それを使う modal / action の近くに置く。

## Import Organization

Go 標準 library、repo 内 package、外部 package の順で gofmt に任せる。repo 内で広く使う package alias は必要最小限にする。

```go
import (
    "context"
    "time"

    domainclient "github.com/Kotaro7750/keda-launcher-scaler/pkg/client"
    "github.com/slack-go/slack"
)
```

## Code Organization Principles

- `Register` は handler 登録順をユーザー体験の順序に近づける。
- Slack modal の block/action ID、metadata、入力値読み取り、submit 検証は同じ artifact の近くに置く。
- Slack message の shape は response artifact としてまとめ、KEDA launcher 送信ロジックとは混ぜない。
- 外部送信用の依存は interface / function field として注入し、repo-owned behavior をテストしやすくする。
- ack は Slack timeout 回避を最優先にし、handler の冒頭または validation response 返却時に一貫して行う。
- 新しい共通 package は、複数 feature で同じ責務が実際に必要になってから作る。
