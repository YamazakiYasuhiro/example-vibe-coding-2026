# 001-Clock-DesktopAnalog

> **Source Specification**: `prompts/phases/000-foundation/branches/main/ideas/001-Clock-DesktopAnalog.md`

## Goal Description

独立 feature `features/clock` として、Windows 向けの丸型アナログ時計デスクトップアクセサリを追加する。円外透過（layered window）、常時最前面、ドラッグ移動、タスクバー非表示、システムトレイ常駐（終了）を満たし、`bin/clock.exe` を `scripts/process/build.sh` で生成する。

## User Review Required

- 自動検証用に CLI フラグ `-quit-after <duration>`（例: `2s`）を追加する。本番利用では未指定のままトレイ「終了」で閉じる。仕様の必須要件外だが、統合スモークに必要。
- トレイ実装は `getlantern/systray` ではなく、**`golang.org/x/sys/windows` 経由の Win32（`Shell_NotifyIconW`）のみ**とする（依存を最小化し、既存 `go build` と整合）。

## Requirement Traceability

| Requirement (from Spec) | Implementation Point (Section/File) |
| :--- | :--- |
| 必須1: `features/clock` + `bin/clock(.exe)` | Proposed Changes > module scaffold / `main.go` / build.sh 既存経路 |
| 必須2: 丸いアナログ時計（時・分・秒針） | `internal/face` の `HandAngles` + `Render` |
| 必須3: 円の外側透過 | `Render` の alpha=0 + `winui` の `UpdateLayeredWindow` |
| 必須4: 常に最前面 | `WS_EX_TOPMOST` / `HWND_TOPMOST` |
| 必須5: ドラッグ移動 | `WM_LBUTTONDOWN`〜`WM_MOUSEMOVE` で `SetWindowPos` |
| 必須6: タスクバー非表示 | `WS_EX_TOOLWINDOW` |
| 必須7: トレイ常駐と終了 | `Shell_NotifyIconW` + コンテキストメニュー「Exit」 |
| 必須8: ローカル時刻・最低 1 秒更新 | `time.Ticker(1*time.Second)` + `face.Render(now)` |
| 必須9: myprog と独立 | 独自 `go.mod`、myprog を import しない / 統合テストで確認 |
| 任意1–4 | **先送り**（仕様どおり任意・本計画対象外） |
| 非要件（他 OS 等） | `!windows` はスタブで非 0 終了。GUI 本体は `windows` ビルドタグ |
| 検証シナリオ 1–10 | Step-by-Step + Verification（見た目系 3–8 は自動化不可部分を単体描画＋スモークで代替し、計画に明記） |

## Proposed Changes

依存関係順: 単体テスト（face）→ face 実装 → winui（Windows）→ main → 統合テスト。

### clock feature (Go)

#### [NEW] `features/clock/go.mod`

*   **Description**: 独立 Go モジュール。
*   **Technical Design**:
    ```go
    module github.com/axsh/tokotachi/features/clock

    go 1.24.0

    require golang.org/x/sys v0.33.0
    ```
*   **Logic**: `features/myprog` への `require` / `replace` は置かない。

#### [NEW] `features/clock/internal/face/angles_test.go`

*   **Description**: `HandAngles` のテーブル駆動単体テスト（TDD Red）。
*   **Technical Design**: パッケージ `face`。角度は **12 時方向を 0°、時計回りを正**、単位は度（`float64`）。許容誤差 `1e-6` または `0.01` 度。
*   **Logic**（仕様の代表ケースを継承）:
    | name | time (Local) | hourDeg | minuteDeg | secondDeg |
    | :--- | :--- | ---: | ---: | ---: |
    | midnight | `2026-09-18 00:00:00` | 0 | 0 | 0 |
    | six_hours | `2026-09-18 06:00:00` | 180 | 0 | 0 |
    | thirty_sec | `2026-09-18 00:00:30` | 0 | 0 | 180 |
    | thirty_min | `2026-09-18 00:30:00` | 15 | 180 | 0 |
    | second_ticks | `t` vs `t+1s` | secondDeg が変わること |

#### [NEW] `features/clock/internal/face/angles.go`

*   **Description**: 針角度の純関数。`time.Now()` は呼ばない。
*   **Technical Design**:
    ```go
    package face

    import "time"

    // Angles holds hand angles in degrees from 12 o'clock, clockwise.
    type Angles struct {
        Hour   float64
        Minute float64
        Second float64
    }

    // HandAngles returns analog hand angles for t in the local wall-clock fields.
    func HandAngles(t time.Time) Angles
    ```
*   **Logic**:
    1. `sec := float64(t.Second())`（ナノ秒は無視してよい。MVP は秒精度）
    2. `min := float64(t.Minute())`
    3. `hour := float64(t.Hour() % 12)`
    4. `Second = sec * 6`（360/60）
    5. `Minute = min * 6 + sec * 0.1`
    6. `Hour = hour * 30 + min * 0.5`（秒による時針微動は任意。MVP は分までで十分だが、仕様例と一致させるため分項は必須）

#### [NEW] `features/clock/internal/face/render_test.go`

*   **Description**: `Render` の透過・不透明サンプル点テスト。
*   **Technical Design**: サイズ `200`。基準時刻 `2026-09-18 00:00:00`。
*   **Logic**:
    1. `img := Render(200, t)` の `Bounds` が `0,0,200,200`。
    2. 四隅 `(0,0)`, `(199,0)`, `(0,199)`, `(199,199)` の `A == 0`。
    3. 中心 `(100,100)` の `A != 0`（文字盤または針のハブ）。
    4. 中心から十分外側だが矩形内の点で、円外なら `A == 0`（例: 中心からの距離 > radius の点）。

#### [NEW] `features/clock/internal/face/render.go`

*   **Description**: 正方形 RGBA に正円の文字盤と 3 針を描画。円外は完全透明。
*   **Technical Design**:
    ```go
    package face

    import (
        "image"
        "image/color"
        "time"
    )

    // DefaultSize is the default clock face edge length in pixels.
    const DefaultSize = 200

    // Render draws a circular analog clock of the given size (width=height=size).
    // Pixels outside the circle have alpha 0.
    func Render(size int, t time.Time) *image.RGBA
    ```
*   **Logic**:
    1. `size < 2` のときは `size = DefaultSize`（または最小 2 にクランプ）。
    2. `img := image.NewRGBA(image.Rect(0, 0, size, size))`（ゼロ値で全面透明）。
    3. 中心 `cx, cy := (size-1)/2.0` 相当、半径 `r := float64(size)*0.48`。
    4. 円内を不透明な文字盤色（例: 白っぽい `color.RGBA{R:240,G:240,B:240,A:255}`）で塗りつぶす（距離 `<= r`）。
    5. 外周リングを濃色で 1〜2px 描いてもよい（任意装飾。MVP では細い円周線を推奨）。
    6. `a := HandAngles(t)` を取得。
    7. 針を中心から描画（12 時が -Y 方向）:
       *   角度 `θ` 度 → ラジアン `rad = (θ - 90) * π/180` は使わず、**12 時基準時計回り**なら:
           `x = cx + len * sin(θ*π/180)`, `y = cy - len * cos(θ*π/180)`
       *   秒針: 長さ `0.90*r`、細く赤系
       *   分針: 長さ `0.75*r`、中太・暗色
       *   時針: 長さ `0.55*r`、太め・暗色
    8. 中心に小さな不透明ドットを描く。
    9. `time.Now()` は呼ばない。

#### [NEW] `features/clock/internal/winui/run_stub.go`

*   **Description**: 非 Windows 用スタブ（`//go:build !windows`）。
*   **Technical Design**:
    ```go
    //go:build !windows

    package winui

    import (
        "fmt"
        "time"
    )

    type Options struct {
        Size      int
        QuitAfter time.Duration
    }

    func Run(opts Options) error {
        return fmt.Errorf("clock GUI requires Windows")
    }
    ```

#### [NEW] `features/clock/internal/winui/run_windows.go`（および必要なら分割ファイル）

*   **Description**: Win32 layered window + トレイ + ドラッグ + 1 秒更新。`//go:build windows`。
*   **Technical Design**:
    ```go
    //go:build windows

    package winui

    import "time"

    type Options struct {
        Size      int           // <=0 なら face.DefaultSize
        QuitAfter time.Duration // >0 ならその時間で自動終了（統合テスト用）
    }

    // Run blocks until the user chooses Exit on the tray menu, QuitAfter elapses, or an error occurs.
    func Run(opts Options) error
    ```
*   **Logic**:
    1. `size := opts.Size`; `size <= 0` なら `face.DefaultSize`。
    2. `user32` / `gdi32` / `shell32` を `windows.NewLazySystemDLL` で解決。最低限次を Proc 化:
       *   `RegisterClassExW`, `CreateWindowExW`, `DestroyWindow`, `ShowWindow`, `UpdateWindow`
       *   `GetMessageW`, `TranslateMessage`, `DispatchMessageW`, `PostQuitMessage`, `DefWindowProcW`
       *   `SetWindowPos`, `GetWindowRect`, `ScreenToClient` / `ClientToScreen`（またはドラッグはスクリーン座標で `SetWindowPos`）
       *   `UpdateLayeredWindow`, `GetDC`, `ReleaseDC`, `CreateCompatibleDC`, `CreateDIBSection`, `SelectObject`, `DeleteObject`, `DeleteDC`
       *   `Shell_NotifyIconW`, `CreatePopupMenu`, `AppendMenuW`, `TrackPopupMenu`, `DestroyMenu`, `SetForegroundWindow`
       *   `SetCapture`, `ReleaseCapture`, `GetCursorPos`
    3. ウィンドウクラス登録（クラス名例: `TokotachiClockFace`）。`WndProc` で:
       *   `WM_LBUTTONDOWN`: キャプチャ開始、ドラッグ原点を記録
       *   `WM_MOUSEMOVE`: ドラッグ中なら `SetWindowPos` で位置更新（サイズ変更なし、`SWP_NOZORDER` は使わず TOPMOST 維持に注意。`HWND_TOPMOST` を維持）
       *   `WM_LBUTTONUP`: キャプチャ解除
       *   `WM_TRAYICON`（自前定義 `WM_APP+1`）: 右クリックでポップアップ「Exit」→ 選択で `PostQuitMessage(0)` とトレイ削除
       *   `WM_DESTROY`: `PostQuitMessage(0)`
    4. `CreateWindowExW`:
       *   style: `WS_POPUP`
       *   exStyle: `WS_EX_LAYERED | WS_EX_TOPMOST | WS_EX_TOOLWINDOW`
       *   初期サイズ `size x size`、初期位置は画面中央付近で可
    5. トレイ: `NOTIFYICONDATAW` で `NIM_ADD`。ツールチップ `"Clock"`。アイコンは簡易生成（小さな HICON）または LoadImage。メニューに `&Exit`（表示文字列 `"Exit"`）を 1 つ。
    6. 描画更新関数 `blit(hwnd)`:
       1. `img := face.Render(size, time.Now())`
       2. RGBA → **premultiplied BGRA** のトップダウン DIB（`BI_RGB`, height 負でトップダウン）へコピー
       3. `UpdateLayeredWindow` with `ULW_ALPHA`, `BLENDFUNCTION{BlendOp: AC_SRC_OVER, SourceConstantAlpha: 255, AlphaFormat: AC_SRC_ALPHA}`
    7. 起動直後に `blit` し、`time.NewTicker(1 * time.Second)` で `blit`（メッセージループと共存: `SetTimer` / `WM_TIMER` でも可。Ticker なら別 goroutine から `PostMessage` で更新要求）。
    8. `opts.QuitAfter > 0` なら同じ仕組みで一度だけ終了を Post。
    9. メッセージループ。終了時 `NIM_DELETE`、ウィンドウ破棄、リソース解放。
    10. DEBUG ログ: 起動、トレイ Exit、QuitAfter 発火、blit エラー時は ERROR（標準ライブラリ `log` で可）。

#### [NEW] `features/clock/main.go`

*   **Description**: エントリポイント。フラグ解析後 `winui.Run`。
*   **Technical Design**:
    ```go
    package main

    import (
        "flag"
        "log"
        "os"
        "time"

        "github.com/axsh/tokotachi/features/clock/internal/face"
        "github.com/axsh/tokotachi/features/clock/internal/winui"
    )

    func main() {
        quitAfter := flag.Duration("quit-after", 0, "auto-exit after duration (for tests); 0 disables")
        size := flag.Int("size", face.DefaultSize, "clock face size in pixels")
        flag.Parse()

        if err := winui.Run(winui.Options{Size: *size, QuitAfter: *quitAfter}); err != nil {
            log.Printf("ERROR: clock exited: %v", err)
            os.Exit(1)
        }
    }
    ```
*   **Logic**:
    1. `-quit-after` 未指定（0）なら通常動作。
    2. エラー時は非 0 終了。成功（トレイ Exit / QuitAfter）は 0。

### integration tests

#### [NEW] `tests/clock_desktop_analog_test.go`

*   **Description**: `bin/clock` のビルド成果物スモーク + myprog 非依存の確認。テスト名に `ClockDesktopAnalog` を含め `--specify` 可能にする。
*   **Technical Design**: `myprogPath` と同様に `clock.exe` / `clock` を探索。
*   **Logic**:
    1. **`TestClockDesktopAnalog_BinaryExists`**: `bin/clock(.exe)` が存在する。
    2. **`TestClockDesktopAnalog_IndependentFromMyprog`**: リポジトリの `features/clock` 配下の `.go` ファイルを読み、`myprog` 文字列を import パスとして含まないこと（簡易: `features/clock/go.mod` に `myprog` が無い、かつ `go list` が使えれば module の deps を確認。最低限 go.mod + ソースの `github.com/axsh/tokotachi/features/myprog` 非含有）。
    3. **`TestClockDesktopAnalog_SmokeQuitAfter`**: **Windows のみ**実行（`runtime.GOOS != "windows"` なら `t.Skip`）。`bin/clock.exe -quit-after 2s` を起動し、およそ 10 秒以内に **exit code 0** で終了すること。stderr に fatal が無いこと（`ERROR:` 連続は失敗扱いでよいが、起動失敗は FAIL）。
    4. 非 Windows では 3 を Skip。1–2 は OS 不問（バイナリは Windows CI 以外では `.exe` が無い可能性 → `build.sh` はカレント OS 向けにビルドするため、非 Windows では `clock` バイナリがスタブ動作の成果物として存在しうる。スタブ `Run` は即エラー終了するため、`-quit-after` スモークは Windows 限定が正しい）。

## Step-by-Step Implementation Guide

1. **Scaffold module**: [ ]
    *   Create `features/clock/go.mod`（上記）。`go get golang.org/x/sys@v0.33.0` 相当で依存を確定。

2. **Unit tests for HandAngles (Red)**: [ ]
    *   Create `internal/face/angles_test.go` with the table cases.
    *   Confirm failure via `./scripts/process/build.sh`.

3. **Implement HandAngles (Green)**: [ ]
    *   Create `internal/face/angles.go` with the formulas above.
    *   Re-run build until face angle tests pass.

4. **Unit tests for Render (Red)**: [ ]
    *   Create `internal/face/render_test.go`.

5. **Implement Render (Green)**: [ ]
    *   Create `internal/face/render.go`（円塗り + 3 針 + 中心点）。
    *   Re-run build until render tests pass.

6. **Implement winui stub + Windows Run**: [ ]
    *   Create `run_stub.go` and Windows implementation（layered window, tray Exit, drag, 1s blit, QuitAfter）。
    *   Create `main.go` with flags.
    *   Re-run `./scripts/process/build.sh` until `bin/clock.exe` builds and unit tests pass.

7. **Add integration tests**: [ ]
    *   Create `tests/clock_desktop_analog_test.go` with the three cases.
    *   Run Verification Plan; fix until green.

8. **Run Verification Plan + §12 総合判定**: [ ]
    *   Execute Automated Verification below and record 総合判定 in this plan file.

## Verification Plan

### Test Item Design (testing-rules §11)

#### Bottom-up order

| Step | Layer | What is verified |
| :--- | :--- | :--- |
| 1 | C: `HandAngles` | 既知時刻 → 時/分/秒角（単体） |
| 2 | C: `Render` | 四隅透明・中心不透明・サイズ（単体） |
| 3 | B: `winui.Run` + `main` | Windows で起動し `-quit-after` で 0 終了（統合スモーク） |
| 4 | A: 成果物・独立性 | `bin/clock` 存在、myprog 非依存（統合） |

見た目の最前面・ドラッグ・トレイ・タスクバー非表示（検証シナリオ 5–8）はヘッドレス自動検証が困難なため、本計画の自動テスト対象外とし、実装で Win32 スタイルを仕様どおり設定することで担保する。自動テストは描画透過（単体）と起動終了（スモーク）に集中する。

#### §11.3 checklist coverage

| # | 観点 | 本計画での扱い |
|---|------|----------------|
| 1 | 正常系 | 角度・描画単体、Windows スモーク起動終了 |
| 2 | 異常系・境界 | `size` クランプ、非 Windows はエラー/Skip |
| 3 | 外部連携 | OS Win32 / 実バイナリ exec（統合） |
| 4 | データ一貫性 | 時刻→角度→画素の一方向。サンプル点で描画一貫性 |
| 5 | 状態遷移 | QuitAfter / Exit でプロセス終了 |
| 6 | 設定・構成 | `-quit-after` / `-size` が動作に反映 |
| 7 | 副作用 | スモーク終了後にプロセスが残らない（Wait 成功） |

#### §11.4 Self-Review result

1. **網羅性**: 必須のうち計算・透過描画・バイナリ生成・独立性・起動終了は自動カバー。最前面/ドラッグ/トレイ UI は OS 設定の実装で満たし、自動 E2E はスモークに限定（仕様もフル GUI E2E を必須としていない）。→ 計画として十分。
2. **証拠の十分性**: 角度の数値断言、alpha サンプル、exit code 0 を要求。→ 十分。
3. **迂回排除**: 統合は実 `bin/clock` を exec。描画は `face.Render` を単体で直接検証。→ 十分。
4. **依存関係**: build.sh（単体含む）→ integration `--specify ClockDesktopAnalog`。→ 整合。

### Automated Verification

1. **Build & Unit Tests**:
    ```bash
    ./scripts/process/build.sh
    ```
    *   **Log Verification**: `features/clock` と `features/myprog` の unit PASS。`bin/clock.exe`（Windows）生成成功。

2. **Integration Tests**:
    ```bash
    ./scripts/process/build.sh && ./scripts/process/integration_test.sh --specify "ClockDesktopAnalog"
    ```
    *   **Log Verification**:
        *   `TestClockDesktopAnalog_BinaryExists` PASS
        *   `TestClockDesktopAnalog_IndependentFromMyprog` PASS
        *   `TestClockDesktopAnalog_SmokeQuitAfter` PASS（Windows）または Skip（非 Windows）
        *   予期しない FAIL が無いこと

3. **E2E Tests (新規)**:
    *   GUI フル E2E（ドラッグ・最前面の視覚確認）は既存インフラが無く、ヘッドレス不可のため **コード化しない**（理由明記）。
    *   バイナリ起動スモークを E2E 相当として `tests/clock_desktop_analog_test.go` に置く。

    #### [NEW] `tests/clock_desktop_analog_test.go`
    *   **テストケース**: `TestClockDesktopAnalog_BinaryExists`, `_IndependentFromMyprog`, `_SmokeQuitAfter`
    *   **検証ポイント**: 成果物存在、myprog 非依存、Windows で `-quit-after 2s` が 0 終了

### Post-Test Comprehensive Verdict (testing-rules §12)

全 Automated Verification 成功後、機械的完了とせず §12 チェックを実施し、本計画書に「総合判定結果」を追記する。

## Documentation

`prompts/specifications` 配下に更新対象なし。

#### 更新対象

None.
