# mini-git 実装計画

## add コマンド

### 方針

- zlib 圧縮なし（保存内容をそのまま確認できるので学習に最適）
- index は `<hash> <filepath>` のテキスト形式（1行1エントリ）
- blob ヘッダなし（まずは内容のみで理解しやすくする）

### 処理フロー

```
go run main.go add <filepath>
```

1. 引数からファイルパスを取得する
2. ファイルの内容を読み込む
3. SHA-1 ハッシュを計算する
4. `objects/<先頭2文字>/<残り38文字>` にファイル内容を保存する
5. `index` ファイルを更新する（同じパスがあれば上書き、なければ追加）

### 実装する関数

| 関数名 | 役割 |
|--------|------|
| `addFile(path string)` | add コマンドのエントリポイント |
| `hashObject(data []byte) string` | SHA-1 ハッシュを計算して返す |
| `saveObject(hash string, data []byte)` | objects/ 配下に blob を保存する |
| `updateIndex(hash, filepath string)` | index ファイルを更新する |
