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

func initRepo() {
	// ここでinitコマンドの実装を行う
	fmt.Println("init command executed")
	os.MkdirAll(objectsDir, 0755)
	os.MkdirAll(refsDir, 0755)
	os.MkdirAll(headsDir, 0755)
	os.WriteFile(headFile, []byte("ref: refs/heads/main"), 0644)
	os.WriteFile(indexFile, []byte(""), 0644)
}

func hashObject(data []byte) string {
	h := sha1.New()
	h.Write(data)
	return hex.EncodeToString(h.Sum(nil))
}

func saveObject(hash string, data []byte) error {
	dir := filepath.Join(objectsDir, hash[:2])
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	path := filepath.Join(dir, hash[2:])
	return os.WriteFile(path, data, 0644)
}

func updateIndex(hash, path string) error {
	data, err := os.ReadFile(indexFile)
	if err != nil {
		return err
	}

	lines := strings.Split(string(data), "\n")
	newLine := hash + " " + path
	updated := false
	var result []string

	for _, line := range lines {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, " ", 2)
		if len(parts) == 2 && parts[1] == path {
			result = append(result, newLine)
			updated = true
		} else {
			result = append(result, line)
		}
	}

	if !updated {
		result = append(result, newLine)
	}

	return os.WriteFile(indexFile, []byte(strings.Join(result, "\n")+"\n"), 0644)
}

func addFile(path string) {
	data, err := os.ReadFile(path)
	if err != nil {
		fmt.Println(err)
		return
	}

	hash := hashObject(data)

	if err := saveObject(hash, data); err != nil {
		fmt.Println(err)
		return
	}

	if err := updateIndex(hash, path); err != nil {
		fmt.Println(err)
		return
	}

	fmt.Printf("add '%s'\n", path)
}
