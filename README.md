# mini-git

Git の内部構造を学ぶための簡易 Git 実装（Go）。blob オブジェクト、インデックス、参照の基本的な仕組みを実装している。

## 実行方法

```bash
go run main.go <command> [<args>]
```

## 実装されたコマンド

### init

`.mini_git/` ディレクトリを初期化する。以下の構造を作成する。

```bash
go run main.go init
```

- `objects/` — blob オブジェクトの保存先
- `refs/heads/` — ブランチの参照
- `HEAD` — 現在のブランチへの参照（デフォルト: `ref: refs/heads/main`）
- `index` — ステージング情報

### add

ファイルをステージングする。SHA-1 ハッシュを計算し、blob を保存し、index に記録する。

```bash
go run main.go add <file>
```

処理フロー：
1. ファイルを読み込む
2. SHA-1 ハッシュを計算する（40 文字の 16 進数）
3. `objects/<2文字>/<38文字>` に blob を保存する
4. `index` に `<hash> <filepath>` の形式で記録する

## .mini_git/ ディレクトリ構造

```
.mini_git/
  objects/     # blob オブジェクトの保存先
               # 例: objects/ab/cdef1234567890...
  refs/
    heads/     # ブランチの参照
  HEAD         # 現在のブランチへの参照
  index        # ステージング情報（<hash> <filepath> 形式）
```

## 学習対象

このプロジェクトでは以下の Git 内部概念を学べる：

- **オブジェクトストア** — SHA-1 ハッシュを基にしたコンテンツ構造化方式
- **インデックス** — ステージング領域の実装方法
- **参照システム** — ブランチと HEAD の管理方法
