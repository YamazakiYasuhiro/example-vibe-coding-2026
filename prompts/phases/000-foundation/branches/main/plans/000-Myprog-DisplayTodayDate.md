# 000-Myprog-DisplayTodayDate

> **Source Specification**: `prompts/phases/000-foundation/branches/main/ideas/000-Myprog-DisplayTodayDate.md`

## Goal Description

`features/myprog` CLI に、起動時の今日の日付表示と `-format` フラグによる表示形式切替（`iso` / `slash` / `jp`）を追加する。既存の `Hello, World!` 出力は維持する。

## User Review Required

None.

（任意要件「時刻表示」および自由形式 Go layout は仕様どおりスコープ外。`-format` の許可値は仕様の `iso` / `slash` / `jp` のみとする。）

## Requirement Traceability

| Requirement (from Spec) | Implementation Point (Section/File) |
| :--- | :--- |
| 必須1: 引数なし実行で今日の日付を表示 | Proposed Changes > `main.go` / Verification > 統合テスト `TestMyprogTodayDate_Default` |
| 必須2: 未指定時は `YYYY-MM-DD`（iso） | Proposed Changes > `date.go` の `iso` 分岐 / `main.go` の flag デフォルト `"iso"` |
| 必須3: `Hello, World!` を維持し日付は別行 | Proposed Changes > `main.go` Logic |
| 必須4: 出力順は Hello → 日付 | Proposed Changes > `main.go` Logic |
| 必須5: 引数なしでも動作し、切替は CLI フラグ | Proposed Changes > `main.go`（`flag.String("format", "iso", ...)`） |
| 必須6: `-format` で `iso`/`slash`/`jp` 切替、不正値は stderr + 非0 | Proposed Changes > `date.go` / 統合テスト invalid ケース |
| 任意1: 時刻表示 | **先送り**（仕様でスコープ外） |
| 非要件: TZ 明示・GUI/API・自由 layout | 実装しない（計画対象外） |
| 検証シナリオ 1–10 | Step-by-Step（TDD）+ Verification Plan の単体/統合テストでカバー（シナリオ10は固定時刻の単体で代替） |

## Proposed Changes

依存関係順: 単体テスト（先） → 整形ロジック → CLI エントリ → 統合テスト基盤 → 統合テスト。

### myprog (Go CLI)

#### [NEW] `features/myprog/date_test.go`

*   **Description**: `FormatToday` の単体テスト（TDD の Red 用）。`time.Now()` に依存せず固定時刻を渡す。
*   **Technical Design**:
    *   パッケージ: `main`（同一パッケージで未公開関数もテスト可能）
    *   テーブル駆動テストを用いる。
*   **Logic**:
    *   基準時刻: `time.Date(2026, 9, 4, 15, 30, 0, 0, time.Local)`
    *   ケース:
        | name | format | want | wantErr |
        | :--- | :--- | :--- | :--- |
        | iso | `"iso"` | `"2026-09-04"` | false |
        | slash | `"slash"` | `"2026/09/04"` | false |
        | jp | `"jp"` | `"2026年09月04日"` | false |
        | empty_as_iso | `""` | `"2026-09-04"` | false |
        | unknown | `"unknown"` | `""` | true |
    *   エラー時は `err != nil` を検証し、エラー文字列に不正な format 名が含まれることを確認する。

#### [NEW] `features/myprog/date.go`

*   **Description**: 日付整形の純粋関数。CLI から切り離して単体テスト可能にする。
*   **Technical Design**:
    ```go
    package main

    import (
        "fmt"
        "time"
    )

    // FormatToday formats t according to named format.
    // Supported format names: "iso" (default), "slash", "jp".
    // Empty format is treated as "iso".
    func FormatToday(t time.Time, format string) (string, error) {
        // ...
    }
    ```
*   **Logic**:
    1. `format == ""` のとき `format = "iso"` とする。
    2. `switch format` で分岐する:
       *   `"iso"` → `t.Format("2006-01-02")` を返す（例: `2026-09-04`）
       *   `"slash"` → `t.Format("2006/01/02")` を返す（例: `2026/09/04`）
       *   `"jp"` → `t.Format("2006年01月02日")` を返す（例: `2026年09月04日`）
       *   default → `("", fmt.Errorf("unsupported format: %q (want iso, slash, or jp)", format))` を返す
    3. `time.Now()` は呼ばない（呼び出し側が時刻を渡す）。

#### [MODIFY] `features/myprog/main.go`

*   **Description**: `flag` で `-format` を受け取り、`Hello, World!` の後に整形済み日付を出力する。不正 format 時は stderr + `os.Exit(1)`。
*   **Technical Design**:
    ```go
    package main

    import (
        "flag"
        "fmt"
        "os"
        "time"
    )

    func main() {
        format := flag.String("format", "iso", "date format: iso, slash, or jp")
        flag.Parse()

        dateStr, err := FormatToday(time.Now(), *format)
        if err != nil {
            fmt.Fprintln(os.Stderr, err)
            os.Exit(1)
        }

        fmt.Println("Hello, World!")
        fmt.Println(dateStr)
    }
    ```
*   **Logic**:
    1. `-format` を定義し、デフォルト値は `"iso"`。
    2. `flag.Parse()` を実行する。
    3. `FormatToday(time.Now(), *format)` を呼ぶ。
    4. エラー時: `fmt.Fprintln(os.Stderr, err)` のあと `os.Exit(1)`。標準出力には `Hello, World!` も日付も出さない（Fail Fast。不正入力で部分成功に見せない）。
    5. 成功時: 1 行目 `Hello, World!`、2 行目に `dateStr` を `fmt.Println` する。
    6. 正常終了時の終了コードは `0`。

### integration tests (CLI E2E)

現状 `tests/` が未整備のため、Go 統合テスト用の最小構成を新設する。GUI 向け E2E ヘルパー（`startE2EServer` 等）は本リポジトリに無く、本機能は CLI のため適用対象外。バイナリ起動テストが E2E 相当となる。

#### [NEW] `tests/go.mod`

*   **Description**: 統合テスト用 Go モジュール。
*   **Technical Design**:
    ```go
    module github.com/axsh/tokotachi/tests

    go 1.24.0
    ```
*   **Logic**: 外部依存は追加しない。`os/exec` で `bin/myprog` を起動する。

#### [NEW] `tests/myprog_today_date_test.go`

*   **Description**: `bin/myprog` の CLI 振る舞いを検証する統合（E2E）テスト。
*   **Technical Design**:
    *   テスト関数名に `MyprogTodayDate` を含め、`--specify "MyprogTodayDate"` で絞り込めるようにする。
    *   バイナリパス: リポジトリルートの `bin/myprog`（テスト内で `tests/` から見て `../bin/myprog`、または環境変数/カレントからの解決）。`integration_test.sh` は `tests/` で `go test` するため、実行前に `build.sh` で `bin/myprog` が生成済みである前提とする。
*   **Logic**（各ケース）:
    1. **`TestMyprogTodayDate_Default`**: 引数なしで実行。exit 0。stdout 1 行目 `Hello, World!`。2 行目は `time.Now()` のローカル日付を `2006-01-02` で整形した文字列と一致。stderr 空。
    2. **`TestMyprogTodayDate_FormatIso`**: `-format iso`。Default と同じ日付形式。
    3. **`TestMyprogTodayDate_FormatSlash`**: `-format slash`。2 行目が `time.Now().Format("2006/01/02")`。
    4. **`TestMyprogTodayDate_FormatJp`**: `-format jp`。2 行目が `time.Now().Format("2006年01月02日")`。
    5. **`TestMyprogTodayDate_InvalidFormat`**: `-format unknown`。exit code ≠ 0。stderr に `unsupported format` 等のエラー文言。stdout に日付行を出さない（Hello も出さない方針に合わせる）。

## Step-by-Step Implementation Guide

1. **Add unit tests (Red)**: [x]
    *   Create `features/myprog/date_test.go` with the table-driven cases above.
    *   Confirm failure via `./scripts/process/build.sh`（`FormatToday` 未定義でコンパイルまたはテスト失敗）。

2. **Implement FormatToday (Green)**: [x]
    *   Create `features/myprog/date.go` with `FormatToday` and the `iso` / `slash` / `jp` / empty / default error logic described in Proposed Changes.
    *   Re-run `./scripts/process/build.sh` until unit tests pass.

3. **Wire CLI in main**: [x]
    *   Edit `features/myprog/main.go` to add `flag`, call `FormatToday(time.Now(), *format)`, Fail Fast on error, then print `Hello, World!` and the date line.
    *   Re-run `./scripts/process/build.sh` to ensure `bin/myprog` builds.

4. **Scaffold integration test module**: [x]
    *   Create `tests/go.mod` with module path `github.com/axsh/tokotachi/tests` and Go `1.24.0`.

5. **Add CLI integration tests (Red then Green)**: [x]
    *   Create `tests/myprog_today_date_test.go` with the five `TestMyprogTodayDate_*` cases.
    *   Run Verification Plan commands; fix until green.
    *   Note: Windows では `os/exec` が `.exe` を要求するため、`build.sh` は `GOEXE` を付与し、テストは `myprog.exe` / `myprog` の両方を探索する。

6. **Run Verification Plan**: [x]
    *   Execute all Automated Verification steps below, then perform §12 総合判定.

### 総合判定結果

**判定**: ✅ 動作確認完了

#### テスト結果サマリ
- 全テスト数: 6 件（単体 1 親 + 5 サブケース相当を 1 テスト関数、統合 5 件）
- 単体: `TestFormatToday` PASS（iso/slash/jp/empty_as_iso/unknown）
- 統合: `TestMyprogTodayDate_*` 5 件すべて PASS
- 失敗: 0 件
- 事実上スキップ: 0 件

#### チェック項目の結果
| # | チェック項目 | 結果 | 備考 |
|---|------------|------|------|
| 1 | スキップされたテスト | ✅ | SKIP/TODO なし |
| 2 | 部分的なエラー | ✅ | ログに ERROR/Exception なし |
| 3 | 迂回処理による偽成功 | ✅ | 実バイナリ exec。形式ごとに異なる期待文字列 |
| 4 | アダプタ・コンフィグの誤適用 | ✅ | `-format` が出力に反映されることを確認 |
| 5 | テスト間の依存・順序問題 | ✅ | ステートレス。フィルタ単独実行で PASS |
| 6 | カバレッジの妥当性 | ✅ | 必須要件 1–6 を単体+統合でカバー |
| 7 | 外部システムの状態 | ✅ | ローカル `bin/myprog(.exe)` のみ |

#### 判定理由
`./scripts/process/build.sh` と `./scripts/process/integration_test.sh --specify "MyprogTodayDate"` がいずれも成功し、期待する stdout/stderr/exit code を明示的に検証しているため、動作確認完了と判断する。


## Verification Plan

### Test Item Design (testing-rules §11)

#### Bottom-up order

| Step | Layer | What is verified |
| :--- | :--- | :--- |
| 1 | C: `FormatToday` | 名前付き format → 文字列 / 不正値 → error（単体） |
| 2 | B: `main` + `flag` | CLI 引数解釈と stdout/stderr/exit（統合） |
| 3 | A: ビルド成果物 | `build.sh` で生成した `bin/myprog` 経由の実動作（統合） |

#### §11.3 checklist coverage

| # | 観点 | 本計画での扱い |
|---|------|----------------|
| 1 | 正常系 | iso/slash/jp/デフォルトの単体 + 統合 |
| 2 | 異常系・境界 | `unknown`、空文字→iso |
| 3 | 外部連携 | CLI バイナリ起動（プロセス実行）を統合で確認。DB/API なし |
| 4 | データ一貫性 | 該当薄（変換の往復なし）。整形が一方向で仕様例と一致することを確認 |
| 5 | 状態遷移 | 該当薄（ステートレス CLI） |
| 6 | 設定・構成 | `-format` の選択が出力に反映されることを統合で確認 |
| 7 | 副作用 | 不正時に stdout を汚さない / 終了コード |

#### §11.4 Self-Review result

1. **網羅性**: 必須要件 1–6 は単体（形式ロジック）と統合（CLI 入出力・exit）の両方でカバー。時刻表示は意図的に除外。→ 十分。
2. **証拠の十分性**: 「エラーが出ない」だけでなく期待文字列・exit code・stderr 内容を断言する。→ 十分。
3. **迂回排除**: 統合は実バイナリ `bin/myprog` を exec するため、単体だけ通って CLI 未配線、という抜けを防げる。→ 十分。
4. **依存関係**: 単体成功後に統合を実行する順序（build.sh → integration_test.sh）を Verification に固定。→ 整合。

### Automated Verification

1. **Build & Unit Tests**:
    ```bash
    ./scripts/process/build.sh
    ```
    *   **Log Verification**: `features/myprog` の unit tests PASS、`bin/myprog` 生成成功。

2. **Integration Tests**:
    *   本リポジトリの `scripts/process/integration_test.sh` は `--categories` をサポートしない（`--specify` と `--help` のみ）。カテゴリ絞り込みは不可のため、`--specify` で myprog 日付テストに限定する。
    ```bash
    ./scripts/process/build.sh && ./scripts/process/integration_test.sh --specify "MyprogTodayDate"
    ```
    *   **Log Verification**:
        *   `TestMyprogTodayDate_Default` / `_FormatIso` / `_FormatSlash` / `_FormatJp` が PASS
        *   `TestMyprogTodayDate_InvalidFormat` が PASS（非0終了と stderr を確認していること）
        *   `SKIP` / 予期しない `FAIL` が無いこと

3. **E2E Tests (新規)**:
    *   GUI E2E は不要（本機能は CLI のみ。理由: 仕様の非要件で GUI 対象外、かつ既存 GUI E2E ヘルパーがリポジトリに存在しない）。
    *   CLI の E2E 相当は上記 `tests/myprog_today_date_test.go` とする。

    #### [NEW] `tests/myprog_today_date_test.go`
    *   **テストケース**: `TestMyprogTodayDate_Default`, `_FormatIso`, `_FormatSlash`, `_FormatJp`, `_InvalidFormat`
    *   **検証ポイント**: stdout 行内容、stderr、exit code が仕様の検証シナリオ 2–9 と一致すること

### Post-Test Comprehensive Verdict (testing-rules §12)

全 Automated Verification 成功後、機械的に完了とせず、次を実施して計画書または walkthrough に記録する。

1. ログに `SKIP` / `WARN` / `TODO` が無いか確認する。
2. 成功ログ内に無視された `ERROR` / `Exception` が無いか確認する。
3. フォールバックや別経路での偽成功が無いか（`-format` 未配線のまま固定文字列だけ、等）を確認する。
4. `TestMyprogTodayDate_*` が実際に実行されたか（フィルタ漏れで 0 件成功になっていないか）を確認する。
5. 単独でも再現するか（必要なら `--specify` で個別名を再実行）を確認する。
6. 新規機能用テストが欠落していないかを Traceability と突合する。
7. 外部依存（本機能はローカルバイナリのみ）の前提が満たされていたかを確認する。

記録フォーマット:

```markdown
### 総合判定結果

**判定**: ✅ 動作確認完了 / ⚠️ 条件付き確認完了 / ❌ 追加確認必要

#### テスト結果サマリ
- 全テスト数: [N] 件
- 成功: [N] 件
- 失敗: [N] 件
- 事実上スキップ: [N] 件

#### チェック項目の結果
| # | チェック項目 | 結果 | 備考 |
|---|------------|------|------|
| 1 | スキップされたテスト | | |
| 2 | 部分的なエラー | | |
| 3 | 迂回処理による偽成功 | | |
| 4 | アダプタ・コンフィグの誤適用 | | |
| 5 | テスト間の依存・順序問題 | | |
| 6 | カバレッジの妥当性 | | |
| 7 | 外部システムの状態 | | |

#### 判定理由
[記述]
```

## Documentation

`prompts/specifications` 配下に既存ドキュメントは存在しない。本計画で更新対象の仕様ドキュメントはなし（Source Specification 自体の更新も不要）。

#### 更新対象

None.
