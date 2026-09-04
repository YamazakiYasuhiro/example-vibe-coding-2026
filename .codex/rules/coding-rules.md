# コーディング規範 (Coding Rules)

本規範は、コードの記述スタイル、技術スタック、および実装上の制約を定め、コードの品質と保守性を確保することを目的とします。

## 1. 使用する技術

### 1.1 Backend (Python)
- **言語**: Python (v3.12+)
- **パッケージ管理**: `uv` + `pyproject.toml`（依存関係・仮想環境の正）
- **ライブラリ** (推奨):
  - Web API: `fastapi` + `uvicorn` (OpenAPI 自動生成)
  - Database: `sqlalchemy` 2.x + `sqlite` (開発/単体テスト用), `postgres` (本番想定)
  - Validation / Settings: `pydantic` / `pydantic-settings`
  - Testing: `pytest` + `pytest-asyncio`（非同期テスト時）

### 1.2 Frontend (VSCode/TypeScript)
- **言語**: TypeScript (v5.x+)
- **フレームワーク**: React (Webview), VSCode API (Extension)
- **Testing**:
  - Unit: Jest / Vitest (Webview components)
  - E2E: Playwright (VSCode Extension Testing)

## 2. ワークフローと開発プロセス

本プロジェクトは **Vibe Coding** スタイルを採用し、AIエージェントとの協働を前提とします。

1. **定型処理のスクリプト化**:
   ビルド、テスト、デプロイなどの操作は `scripts/` ディレクトリ配下にシェルスクリプトとして実装すること（例: `scripts/process/build.sh`, `scripts/process/integration_test.sh`）。これらは `--help` を実装し、AIが自律的に実行可能でなければならない。
   > [!IMPORTANT]
   > `npm run build`, `python -m pytest`, `uv run pytest` などの直接実行は、環境間の差異や成果物の配置不整合を招くため **禁止** します。必ず対応する `scripts/process/` 配下のスクリプトを使用してください。

2. **ワークフローによるオーケストレーション**:
   一連の作業手順（例: 修正→テスト→再テスト）は各エージェントの workflows / skills に定義し、定型化する。

3. **TDD (テスト駆動開発) の徹底**:
   開発は必ず以下のサイクルで進めること。
   1. **レッド**: 実装の前に、その機能の要件を満たすテストコード（Unit Test）を先に書く。この時点でテストは失敗する。
   2. **グリーン**: テストを通過させるための最小限の実装を行う。
   3. **リファクタリング**: テスト通過後、コードの品質を高めるためにリファクタリングを行う。
   *テストのない実装は原則として認めない。*

## 3. 実装上の共通ルール

### 3.1 コードの品質と保守性

*   **ファイル・関数の分割**: 単一のファイルまたは関数の行数が **800行** を超える場合、コードの可読性と保守性を確保するために、ファイルの分割や関数の抽出を積極的に検討すること。
*   **DRY 原則 (Don't Repeat Yourself)**: 重複したロジックは共通関数や共有パッケージへ抽出し、再利用性を高めること。同じロジックを複数箇所に書くことは保守コストを増加させるため禁止する。
*   **KISS 原則 (Keep It Simple, Stupid)**: 実装は常にシンプルに保つこと。複雑な抽象化や過剰な設計は可読性を損なう。問題を解決する最もシンプルな方法を選択し、不必要な複雑さを持ち込まないこと。
*   **YAGNI 原則 (You Aren't Gonna Need It)**: 現時点で必要のない機能を先読みして実装しないこと。「将来必要になるかもしれない」という理由だけで機能を追加することは禁止する。実際に要件として確定した時点で実装する。
*   **利用していないコードの削除**: どこからも参照されず、利用されない変数、関数、コードは可能な限り削除してください。

### 3.2 コメントと言語設定

*   **コメントの言語**: ソースコード内のコメント（ドキュメントコメント、インラインコメント含む）はすべて **英語** で記述すること。日本語の使用は **禁止** とする。
    *   **理由**: プロジェクトの国際的な可読性を確保し、AIモデルが意図を正確に解釈できるようにするため。

## 4. テスト実装規約

### 4.1 テストの構成

#### Backend (Python)
*   **単体テスト (Unit Test)**:
    *   対象: `src/` および `features/` 配下の各パッケージ。
    *   実行: `scripts/process/build.sh --skip-frontend --skip-etc`（内部で pytest を実行）。
    *   制約: **Fail Fast**。外部通信を行わず、モックを使用すること。
*   **統合テスト (Integration Test)**:
    *   対象: `tests/` ディレクトリ配下。
    *   実行: `scripts/process/integration_test.sh` (カテゴリ指定なし、または `llm` 等のバックエンドカテゴリ指定)。
    *   制約: 実際のコンテナやAPIと通信を行う。

#### Frontend (VSCode/TypeScript)
*   **単体テスト (Unit Test)**:
    *   対象: Webview (React) コンポーネント等。
    *   実行: `npm test` (Webviewディレクトリ内) は開発時のみ許容。CI/Verificationでは `scripts/process/build.sh` を使用すること。
*   **E2Eテスト (Integration Test)**:
    *   対象: VSCode拡張機能の挙動検証。
    *   実行: `scripts/process/integration_test.sh --categories gui`。
    *   制約: Test Driver Patternを使用し、VSCodeのインスタンスを起動して検証する。

### 4.2 テスト自動化

*   PR作成前やコミット前には必ず build-pipeline に相当するフロー（Unit Test -> Build -> Integration Test）を実行し、品質を担保すること。

### 4.3 E2Eテストの堅牢性 (E2E Test Robustness)

*   **Action-Verification パターン**:
    *   アクション（クリック、入力など）を行う際は、必ずその直後に期待する結果（要素の表示、状態変化など）を検証すること。
    *   非推奨: `await click(...)` -> `await timeout(...)`
    *   必須: `await driver.runAction(..., verification)` または `await click(...)` -> `await expect(...).toBeVisible()`
*   **Fail Fast**:
    *   アクション実行中にブラウザコンソールのエラーや例外が発生した場合は、即座にテストを失敗させること（`runAction` ヘルパーの使用を推奨）。
*   **Editor Cleanup**:
    *   テスト終了時や開始時のエディタ終了処理には、必ず `driver.closeAllEditors()` を使用し、保存ダイアログ等のハンドリングをドライバに委譲すること。生の `View: Close All Editors` コマンドの使用は禁止する。

## 5. ロギングと通信処理

*   **統一ログシステムの使用**: 全てのログ出力は、プロジェクト共通のロガーモジュールを通じて行うこと。標準の `print` や、アドホックな `logging.getLogger(__name__)` の散在は禁止とする（共通ロガー経由で取得すること）。
*   **ログレベルの使い分け**: `DEBUG`, `INFO`, `WARNING`, `ERROR` を適切に使い分けること。

### 5.2 Web Frontend API Request

*   **標準Utilsの使用**: Webview内でのAPIリクエストには、必ず `ide/webview/src/utils/api.ts` の `fetchWithRetry` と `ide/webview/src/utils/config.ts` の `getApiBaseUrl` を使用すること。
*   **禁止事項**:
    *   `fetch` の直接使用 (リトライやエラーハンドリングが統一されないため)
    *   `http://localhost` やポート番号のハードコード (環境依存を避けるため)


## 6. ファイル操作とパス

*   **設定ファイルの分離**: 機密情報や環境依存の設定は `settings/` 配下のYAMLファイルで管理し、コード内にハードコードしないこと。
*   **パス指定の標準**: 内部的なファイルパス操作には `pathlib.Path` を使用し、OS間の差異（セパレータ等）を吸収すること。文字列結合によるパス組み立ては禁止する。

## 7. Python のモダンな記述 (Modern Python Practices)

Python 3.12 以降の機能と型ヒントを活用し、以下の指針に従うこと。

1.  **Type Hints**:
    *   公開関数・メソッドには引数と戻り値の型注釈を付ける。
    *   `list[T]`, `dict[K, V]`, `X | Y` など組み込みジェネリック／ユニオン構文を優先し、`typing.List` / `Optional` 等の旧記法は避ける。
    *   実行時チェックが必要な境界（API入出力・設定）では `pydantic` モデルを用いる。

2.  **Protocols / ABC**:
    *   依存の差し替えが必要な箇所では、具象クラスではなく `typing.Protocol` または小さな ABC で契約を定義する。
    *   インターフェースは「定義する側」ではなく「利用する側」に寄せ、メソッド数は最小限（1-3個）に留める。

3.  **Async**:
    *   I/O 待ちが主体の処理には `async`/`await` を用いる。
    *   同期ブロッキング呼び出しをイベントループ内で直接実行しない（必要なら `asyncio.to_thread` 等で隔離）。

4.  **Dataclasses / Pydantic**:
    *   単純な値オブジェクトには `@dataclass(frozen=True)` を検討する。
    *   バリデーションやシリアライズが必要なモデルには Pydantic を使う。

5.  **Imports and Packages**:
    *   パッケージは `src/` レイアウト（または `features/<name>/` 配下のパッケージ）を正とする。
    *   相対インポートは同一パッケージ内に限定し、跨パッケージは絶対インポートを使う。
