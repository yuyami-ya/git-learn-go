# commitコマンドの実装

## はじめに

前章では `add` コマンドを使って、ファイルをハッシュ化（blobの作成）し、ステージングエリア `.mini_git/index` に登録する仕組みを実装した。
これにより、「どのファイルをコミット対象とするか」が決まった。

しかし、まだ「コミット（履歴として確定）」する段階が残っている。コミットを実装しないと、ステージングに置いたファイルは履歴として残らず、バージョン管理の意味がなくなってしまう。

そこで今回は、本家Gitの `git commit` に相当する「commit」コマンドを作成し、ステージングされたファイル一式をスナップショットとして保存し、Gitライクな履歴管理を実現していく。

---

## コミットは何をしているのか

実際のGitでは、コミットを行うと以下の一連の処理が行われる:

1. **treeオブジェクトを作成する** - プロジェクト内のファイルやディレクトリの構造を記録する
2. **commitオブジェクトを生成する** - treeを指し、親コミットとのつながりを記録する

### treeオブジェクトとは

treeオブジェクトは、プロジェクト内のファイルやディレクトリの構造を表現する。
これはまるでプロジェクト全体の「写真」のようなもので、どのファイルがどのディレクトリにあるかを詳細に記録する。

### commitオブジェクトとは

commitオブジェクトは、以下の情報を含む:

- どのtreeを使っているか
- 親コミット（前の履歴）は何か
- コミットメッセージは何か

これにより、履歴が一つ一つのコミットとして積み重なっていく。

---

## mini-gitでのコミット

今回のmini-gitでは、treeオブジェクトの実装を省略し、もっとシンプルにしている。

具体的には、ステージングエリア（index）の内容（ファイル名とblobハッシュの対応）をそのままコミット時のスナップショットとして採用し、それを直接保存する。

つまり、「コミットオブジェクトの中にindexの内容をそのまま詰め込む」形である。

---

## commitオブジェクトの構造（mini-git版）

mini-gitで作るcommitオブジェクトは、以下のような3行のテキストである:

```
tree {"hello.txt":"d44a2379..."}
parent 556bd5b4...
message 初回のコミットメッセージ
```

| 行 | 内容 |
|----|------|
| `tree` | indexの内容をJSON化したもの（その時点のファイル一覧） |
| `parent` | 親コミットのハッシュ（最初のコミットなら空） |
| `message` | コミットメッセージ |

このテキストのSHA-1ハッシュを計算し、`.mini_git/objects/` に保存する。
その後、現在のブランチファイル（例: `.mini_git/refs/heads/master`）を、このcommitのハッシュ値で更新することで「今のブランチの先端が新しいコミットになった」状態を作る。

---

## commitの実装ステップ

commitの処理は以下の4ステップで構成される。

### ステップ1: ステージングエリアの読み込み

コミットする対象は「ステージングされているファイル」なので、まず `.mini_git/index` を読み込む。
もしステージングエリアが空（indexが空）なら、「コミットするものがない」というエラーを出して終了する。

### ステップ2: 親コミットを調べる

Gitにおける「コミットの履歴」は、parentをたどるリスト構造で表現される。

現在のブランチ（HEADが参照しているブランチ）のファイルを読めば、そこに最新コミットIDが書いてある。
もしまだコミットがない場合は空文字になるが、それは「このコミットが最初」だということを意味する。

### ステップ3: commitオブジェクトの作成と保存

以下の流れでcommitオブジェクトを作成する:

1. `.mini_git/index` の中身（ファイル名 → ハッシュの対応）をJSONで文字列化し、`tree ...` の行として記録する
2. `parent <親コミットID>` の行を作る
3. `message <コミットメッセージ>` の行を作る
4. これらをまとめた文字列のSHA-1ハッシュを計算し、`.mini_git/objects/` に保存する

### ステップ4: ブランチ（HEAD）の更新

コミットが成功したら、そのコミットIDを現在のブランチファイルに書き込む。

例えば、masterブランチなら `.mini_git/refs/heads/master` にコミットIDを書く。
HEADが `ref: refs/heads/master` を指しているなら、それが「今のmasterブランチが指すコミットIDはこれだ」という更新になる。

---

## Goでの実装

### commitChanges関数

4つのステップをそのままGoコードにしたものが以下である。
前章で作った `hashObject()`, `saveObject()`, `readIndex()` を再利用する。
また、`getHeadCommit()` と `getCurrentBranch()` は後ほど実装する。

```go
func commitChanges(message string) {
    // ---- ステップ1: ステージングエリアの読み込み ----
    // indexにはaddコマンドで登録された「ファイル名 → ハッシュ」の対応が入っている。
    // これがコミット時のスナップショット（その時点のファイル一覧）になる。
    indexMap := readIndex()

    // indexが空 = addされたファイルがない = コミットするものがない
    if len(indexMap) == 0 {
        fmt.Println("Nothing to commit (index is empty).")
        return
    }

    // ---- ステップ2: 親コミットの取得 ----
    // 「親コミット」とは、今回のコミットの直前のコミットのこと。
    // これにより、コミット同士が数珠つなぎになり「履歴」が生まれる。
    // まだ一度もコミットしていない場合は空文字が返る（= 最初のコミット）。
    parent := getHeadCommit()

    // ---- ステップ3: コミットオブジェクトの作成と保存 ----
    // indexの内容（map）をJSON文字列に変換する。
    // これが「その時点でどのファイルがどのハッシュだったか」の記録になる。
    treeJSON, _ := json.Marshal(indexMap)

    // コミットオブジェクトの文字列を組み立てる
    commitStr := fmt.Sprintf("tree %s\nparent %s\nmessage %s\n", string(treeJSON), parent, message)

    // コミット文字列のSHA-1ハッシュを計算する
    commitHash := hashObject([]byte(commitStr))

    // objectsディレクトリにコミットオブジェクトを保存する
    // blobもcommitも同じobjectsに保存される（Gitと同じ仕組み）
    if err := saveObject(commitHash, []byte(commitStr)); err != nil {
        fmt.Printf("Error: %v\n", err)
        return
    }

    // ---- ステップ4: ブランチファイルの更新 ----
    // 現在のブランチファイル（例: refs/heads/master）に
    // 新しいコミットのハッシュを書き込む
    currentBranch := getCurrentBranch()
    if currentBranch != "" {
        branchFile := filepath.Join(headsDir, currentBranch)
        os.WriteFile(branchFile, []byte(commitHash), 0644)
    } else {
        os.WriteFile(headFile, []byte(commitHash), 0644)
    }

    fmt.Printf("Committed as %s\n", commitHash)
}
```

#### コードの解説

- `parent` フィールドに前のコミットハッシュを書き込む。これにより、コミット履歴をどんどんつなげていける
- `tree` にはindexの内容をJSON化したものを保存する。これでコミット時点の「ファイル名とblobハッシュ」がわかる
- `message` は引数のメッセージを挿入する。`go run main.go commit "初回のコミット"` のように実行した場合、コミットメッセージを引数から取得する

### getHeadCommit関数

HEADが指しているコミットハッシュを取得する関数である。

```go
func getHeadCommit() string {
    // HEADファイルが存在しなければ、まだ初期化されていない
    data, err := os.ReadFile(headFile)
    if err != nil {
        return ""
    }

    ref := strings.TrimSpace(string(data))

    if strings.HasPrefix(ref, "ref:") {
        // パターン1: "ref: refs/heads/master" の形式
        // "ref:" の後ろの部分（"refs/heads/master"）を取り出す
        refPath := strings.TrimSpace(strings.TrimPrefix(ref, "ref:"))

        // ブランチファイルのフルパスを組み立てる
        // 例: .mini_git/refs/heads/master
        fullRefPath := filepath.Join(miniGitDir, refPath)

        // ブランチファイルを読み込んで、中に書かれたコミットハッシュを返す
        refData, err := os.ReadFile(fullRefPath)
        if err != nil {
            return ""
        }
        return strings.TrimSpace(string(refData))
    }

    // パターン2: HEADがコミットハッシュを直接指している（デタッチ状態）
    return ref
}
```

#### getHeadCommit関数の解説

この関数は、現在のHEADが指している最新のコミットIDを取得する役割を持っている。

HEADファイルの中身には2つのパターンがある:

| パターン | HEADの中身 | 意味 |
|---------|-----------|------|
| 通常状態 | `ref: refs/heads/master` | ブランチを経由してコミットを指す |
| デタッチ状態 | `abc123...` | コミットハッシュを直接指す |

- **通常状態**: HEADがブランチを指している場合、そのブランチの最新コミットIDを返す
- **デタッチ状態**: HEADが直接特定のコミットIDを指しているときは、そのコミットID自体を返す
- **初期状態**: まだ一度もコミットが行われていない場合は空文字を返す

### getCurrentBranch関数

現在作業しているブランチ名を取得するための関数である。

```go
func getCurrentBranch() string {
    data, err := os.ReadFile(headFile)
    if err != nil {
        return ""
    }

    ref := strings.TrimSpace(string(data))

    if strings.HasPrefix(ref, "ref:") {
        // "ref: refs/heads/master" から "master" を取り出す
        refPath := strings.TrimSpace(strings.TrimPrefix(ref, "ref:"))
        return filepath.Base(refPath)
    }

    // "ref:" で始まらない = デタッチ状態
    return ""
}
```

#### getCurrentBranch関数の解説

- HEADが `ref: refs/heads/master` の形なら、ブランチ名 `master` を返す
- HEADがデタッチ状態（コミットハッシュを直接指している）なら、空文字を返す

`filepath.Base()` はパスの最後の要素を返すGoの標準関数である。
例: `refs/heads/master` → `master`

### main関数へのcommitコマンド追加

`main()` 関数の `switch` 文に `case "commit"` を追加する:

```go
case "commit":
    if len(os.Args) < 3 {
        fmt.Println("Usage: mini_git commit <message>")
        os.Exit(1)
    }
    commitChanges(os.Args[2])
```

---

## 動作確認

### 1. リポジトリの初期化からaddまで（前回実行していない場合）

```
$ go run main.go init
Initialized empty mini-git repository in .mini_git
```

続いて、ファイルを作成して、ステージングエリアに追加する:

```
$ echo "Hello, mini-git!" > hello.txt
$ go run main.go add hello.txt
Added hello.txt to index with hash d44a2379...
```

### 2. commitコマンドの動作確認

今回作成したcommitコマンドを実行する:

```
$ go run main.go commit "初回のコミットメッセージ"
Committed as 7aad8901...
```

コマンドの文法は `go run main.go commit <コミットメッセージ>` である。
`<コミットメッセージ>` には、どのような変更をしたのかを記述する。

### 3. コミットが正しく作成されたかを確認

masterブランチが指すコミットハッシュを確認する:

```
$ cat .mini_git/refs/heads/master
7aad8901...
```

`.mini_git/refs/heads/master` ファイルには、masterブランチが指している最新のコミットハッシュが書かれている。先ほど生成されたハッシュが書かれているはずである。

objectsフォルダを覗くと、blobオブジェクト（addで保存したもの）に加えて、commitオブジェクトが増えている:

```
$ ls .mini_git/objects/
7aad8901d66ee036c39116b306dcf62216727671   <- commitオブジェクト
d44a23794b3d85c85fbae4edb6c227825718a067   <- blobオブジェクト（hello.txt）
```

commitオブジェクトの中身を確認する:

```
$ cat .mini_git/objects/7aad8901d66ee036c39116b306dcf62216727671
tree {"hello.txt":"d44a2379..."}
parent
message 初回のコミットメッセージ
```

最初のコミットなので `parent` が空になっている。

### 4. 2回目のコミットでparentがつながる

ファイルに変更を加えて、再度addとcommitを実行する:

```
$ echo "Hello, mini-git! v2" > hello.txt
$ go run main.go add hello.txt
$ go run main.go commit "2回目のコミットメッセージ"
Committed as 44da46c7...
```

2回目のコミットオブジェクトの中身を確認する:

```
$ cat .mini_git/objects/44da46c7...
tree {"hello.txt":"f2e9a085..."}
parent 7aad8901...
message 2回目のコミットメッセージ
```

`parent` に1回目のコミットハッシュが記録されている。
これにより、コミット履歴が連続してつながり、まるでチェーンのように一貫した履歴が形成される。

---

## indexの中身はクリアされる?

今回の実装では、コミットが終わっても `.mini_git/index` の内容は消していない（本家Gitではコミット後にステージングがクリアされる）。
mini-gitをシンプルにするため、そこまで再現していないが、気になる方は「コミット後にindexを空にする」処理を追加してみるとよい。

---

## まとめ

この章で学んだことを整理する:

- **commitは「ステージングされたファイルのスナップショット」をオブジェクト化して保存する作業である**
- mini-gitでは、indexの内容をまとめてJSON化し、それを含むテキストを一つのcommitオブジェクトとして `.mini_git/objects/` に保存する
- **parentというフィールドを持たせることで、コミット履歴をたどることができる**
- コミット完了後、現在のブランチファイルに新しいコミットIDを書き込むことで、ブランチの先端を更新する

これで「init → add → commit」といった、Gitの基本的な「バージョンを積み重ねていく」流れをひととおり再現できた。
