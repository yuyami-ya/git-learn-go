#!/usr/bin/env python3
import os
import sys
import hashlib
import zlib

# 定数の定義
MINI_GIT_DIR = '.mini_git'  # mini-gitの管理ディレクトリ
OBJECTS_DIR = os.path.join(MINI_GIT_DIR, 'objects')  # オブジェクト保存用ディレクトリ
REFS_DIR = os.path.join(MINI_GIT_DIR, 'refs')  # 参照情報保存用ディレクトリ
HEADS_DIR = os.path.join(REFS_DIR, 'heads')  # ブランチ情報保存用ディレクトリ
HEAD_FILE = os.path.join(MINI_GIT_DIR, 'HEAD')  # HEADファイルのパス
INDEX_FILE = os.path.join(MINI_GIT_DIR, 'index')  # ステージングエリアのindexファイル

def main():
    if len(sys.argv) < 2:
        print("Usage: mini_git <command> [<args>]")
        sys.exit(1)

    command = sys.argv[1]

    if command == 'init':
        init_repo()
    elif command == 'add':
        if len(sys.argv) < 3:
            print("Usage: mini_git add <filename>")
            sys.exit(1)
        add_file(sys.argv[2])
    else:
        print(f"Unknown command: {command}")

def init_repo():
    """
    mini-gitリポジトリを初期化します。
    .mini_git/ ディレクトリと必要なサブディレクトリ、ファイルを作成します。
    """
    # 既に .mini_git が存在する場合は警告して終了
    if os.path.exists(MINI_GIT_DIR):
        print(f"{MINI_GIT_DIR} already exists")
        return

    # .mini_gitディレクトリとサブディレクトリを作成
    os.makedirs(OBJECTS_DIR)
    os.makedirs(REFS_DIR)
    os.makedirs(HEADS_DIR)

    # HEADファイルの初期化（masterブランチを指す）
    with open(HEAD_FILE, 'w') as f:
        f.write("ref: refs/heads/master\n")

    # indexファイル（空）を作成
    with open(INDEX_FILE, 'w') as f:
        f.write("")

    # デフォルトのブランチ: masterのファイルを作成（空）
    master_file = os.path.join(HEADS_DIR, 'master')
    with open(master_file, 'w') as f:
        f.write("")

    print(f"Initialized empty mini-git repository in {MINI_GIT_DIR}")

def hash_object(data):
    """
    data(バイナリ)を受け取り、SHA-1でハッシュ化して
    .mini_git/objects/ 以下に保存する。
    既に同名のオブジェクトが存在すれば保存はスキップ。
    戻り値: 計算したハッシュ値(40文字の16進数)
    """
    sha1 = hashlib.sha1(data).hexdigest()
    obj_path = os.path.join(OBJECTS_DIR, sha1)
    
    if not os.path.exists(obj_path):
        compressed = zlib.compress(data)
        with open(obj_path, 'wb') as f:
            f.write(compressed)
    
    return sha1

def read_index():
    """
    .mini_git/index を読み込んで、
    {filename: sha1, ...} の辞書を返す
    """
    index_dict = {}
    if os.path.exists(INDEX_FILE):
        with open(INDEX_FILE, 'r') as f:
            for line in f:
                line = line.strip()
                if not line:
                    continue
                fname, sha = line.split()
                index_dict[fname] = sha
    return index_dict

def write_index(index_dict):
    """
    {filename: sha1} の辞書を .mini_git/index に書き出す
    """
    with open(INDEX_FILE, 'w') as f:
        for fname, sha in index_dict.items():
            f.write(f"{fname} {sha}\n")

def add_file(filename):
    """
    指定されたファイルをステージングエリアに追加します。
    ファイルをハッシュ化し、オブジェクトとして保存し、indexを更新します。
    """
    # ファイルが存在するか確認
    if not os.path.isfile(filename):
        print(f"Error: {filename} does not exist.")
        return

    # ファイル内容をバイナリで読み込む
    with open(filename, 'rb') as f:
        data = f.read()

    # blobとしてハッシュを計算・objectsに保存
    sha1 = hash_object(data)

    # indexを読み込む
    index_dict = read_index()

    # indexにfilename: sha1 を登録（更新）
    index_dict[filename] = sha1

    # indexを書き出す
    write_index(index_dict)

    print(f"Added {filename} to index with hash {sha1}")

if __name__ == "__main__":
    main()