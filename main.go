package main

import (
	"crypto/sha1"
	"encoding/hex"
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
