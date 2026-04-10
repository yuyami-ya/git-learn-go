package main

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// 管理用ディレクトリとファイルのパスを定義
var (
	miniGitDir = ".mini_git"
	objectsDir = filepath.Join(miniGitDir, "objects")
	refsDir    = filepath.Join(miniGitDir, "refs")
	headsDir   = filepath.Join(refsDir, "heads")
	headFile   = filepath.Join(miniGitDir, "HEAD")
	indexFile  = filepath.Join(miniGitDir, "index")
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: mini_git <command> [<args>]")
		os.Exit(1)
	}

	command := os.Args[1]

	switch command {
	case "init":
		initRepo()
	case "add":
		if len(os.Args) < 3 {
			fmt.Println("Usage: mini_git add <file>")
			os.Exit(1)
		}
		addFile(os.Args[2])
	case "commit":
		// commitコマンド: ステージングエリアの内容を履歴として確定する
		// 使い方: mini_git commit "コミットメッセージ"
		if len(os.Args) < 3 {
			fmt.Println("Usage: mini_git commit <message>")
			os.Exit(1)
		}
		commitChanges(os.Args[2])
	default:
		fmt.Printf("Unknown command: %s\n", command)
	}
}

// initRepo はmini-gitリポジトリを初期化する。
// .mini_git/ ディレクトリと必要なサブディレクトリ、ファイルを作成する。
func initRepo() {
	// 既に .mini_git が存在する場合は警告して終了
	if _, err := os.Stat(miniGitDir); err == nil {
		fmt.Printf("%s already exists\n", miniGitDir)
		return
	}

	// .mini_gitディレクトリとサブディレクトリを作成
	os.MkdirAll(objectsDir, 0755)
	os.MkdirAll(refsDir, 0755)
	os.MkdirAll(headsDir, 0755)

	// HEADファイルの初期化（masterブランチを指す）
	os.WriteFile(headFile, []byte("ref: refs/heads/master\n"), 0644)

	// indexファイル（空）を作成
	os.WriteFile(indexFile, []byte(""), 0644)

	// デフォルトのブランチ: masterのファイルを作成（空）
	masterFile := filepath.Join(headsDir, "master")
	os.WriteFile(masterFile, []byte(""), 0644)

	fmt.Printf("Initialized empty mini-git repository in %s\n", miniGitDir)
}

// hashObject はデータのSHA-1ハッシュを計算して返す。
func hashObject(data []byte) string {
	h := sha1.New()
	h.Write(data)
	return hex.EncodeToString(h.Sum(nil))
}

// saveObject はオブジェクトを .mini_git/objects/<hash> に保存する。
// 既に同名のオブジェクトが存在すれば保存はスキップする。
func saveObject(hash string, data []byte) error {
	objPath := filepath.Join(objectsDir, hash)

	// 既に存在すればスキップ
	if _, err := os.Stat(objPath); err == nil {
		return nil
	}

	return os.WriteFile(objPath, data, 0644)
}

// readIndex は .mini_git/index を読み込んで、
// {filename: sha1} の map を返す。
func readIndex() map[string]string {
	indexMap := make(map[string]string)

	data, err := os.ReadFile(indexFile)
	if err != nil {
		return indexMap
	}

	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, " ", 2)
		if len(parts) == 2 {
			indexMap[parts[0]] = parts[1]
		}
	}

	return indexMap
}

// writeIndex は {filename: sha1} の map を .mini_git/index に書き出す。
func writeIndex(indexMap map[string]string) error {
	var lines []string
	for fname, sha := range indexMap {
		lines = append(lines, fmt.Sprintf("%s %s", fname, sha))
	}

	content := ""
	if len(lines) > 0 {
		content = strings.Join(lines, "\n") + "\n"
	}

	return os.WriteFile(indexFile, []byte(content), 0644)
}

// addFile は指定されたファイルをステージングエリアに追加する。
// ファイルをハッシュ化し、オブジェクトとして保存し、indexを更新する。
func addFile(path string) {
	// ファイルが存在するか確認
	if _, err := os.Stat(path); os.IsNotExist(err) {
		fmt.Printf("Error: %s does not exist.\n", path)
		return
	}

	// ファイル内容をバイナリで読み込む
	data, err := os.ReadFile(path)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	// blobとしてハッシュを計算
	sha1Hash := hashObject(data)

	// objectsに保存
	if err := saveObject(sha1Hash, data); err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	// indexを読み込む
	indexMap := readIndex()

	// indexにfilename: sha1 を登録（更新）
	indexMap[path] = sha1Hash

	// indexを書き出す
	if err := writeIndex(indexMap); err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Added %s to index with hash %s\n", path, sha1Hash)
}

// =============================================================
// ここから commit コマンドの実装
// =============================================================

// commitChanges はステージングエリア(index)にあるファイルを
// コミットとして確定し、コミットオブジェクトを作成して保存する。
//
// コミットの処理は以下の4ステップで行われる:
//   1. indexを読み込む（ステージングされたファイル一覧を取得）
//   2. 親コミットを調べる（履歴のつながりを作るため）
//   3. コミットオブジェクトを作成してobjectsに保存する
//   4. ブランチファイルを更新する（ブランチの先端を新しいコミットにする）
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
	// コミットオブジェクトは以下の3行で構成されるテキストである:
	//   tree {"hello.txt":"abc123..."}   ← indexの内容をJSON化したもの
	//   parent 556bd5b4...               ← 親コミットのハッシュ
	//   message 初回のコミット            ← コミットメッセージ

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
	// 新しいコミットのハッシュを書き込むことで、
	// 「このブランチの最新コミットはこれだ」という状態を作る。
	currentBranch := getCurrentBranch()
	if currentBranch != "" {
		// ブランチが存在する場合: ブランチファイルにコミットハッシュを書き込む
		// 例: .mini_git/refs/heads/master に "abc123..." と書く
		branchFile := filepath.Join(headsDir, currentBranch)
		os.WriteFile(branchFile, []byte(commitHash), 0644)
	} else {
		// デタッチ状態の場合: HEADファイルに直接コミットハッシュを書き込む
		// （デタッチ状態 = 特定のブランチではなく、直接コミットを指している状態）
		os.WriteFile(headFile, []byte(commitHash), 0644)
	}

	fmt.Printf("Committed as %s\n", commitHash)
}

// getHeadCommit はHEADが指している最新のコミットハッシュを返す。
//
// HEADファイルの中身は2パターンある:
//   パターン1: "ref: refs/heads/master" → ブランチを経由してコミットを指す（通常状態）
//   パターン2: "abc123..."              → コミットハッシュを直接指す（デタッチ状態）
//
// まだ一度もコミットしていない場合は空文字を返す。
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
			// ファイルが読めない = まだコミットがない
			return ""
		}
		return strings.TrimSpace(string(refData))
	}

	// パターン2: HEADがコミットハッシュを直接指している（デタッチ状態）
	return ref
}

// getCurrentBranch はHEADが指しているブランチ名を返す。
//
// 例: HEADの中身が "ref: refs/heads/master" なら "master" を返す。
// デタッチ状態（HEADがコミットハッシュを直接指している）なら空文字を返す。
func getCurrentBranch() string {
	data, err := os.ReadFile(headFile)
	if err != nil {
		return ""
	}

	ref := strings.TrimSpace(string(data))

	if strings.HasPrefix(ref, "ref:") {
		// "ref: refs/heads/master" から "master" を取り出す
		refPath := strings.TrimSpace(strings.TrimPrefix(ref, "ref:"))
		// filepath.Base はパスの最後の要素を返す
		// 例: "refs/heads/master" → "master"
		return filepath.Base(refPath)
	}

	// "ref:" で始まらない = デタッチ状態
	return ""
}
