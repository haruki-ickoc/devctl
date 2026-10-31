package docker

import (
    "archive/tar"
    "compress/gzip"
    "os"
    "path/filepath"
    "reflect"
    "strings"
    "testing"
    "time"
)

// createTestTarGz はテスト用の .tar.gz アーカイブファイルを作成します。
func createTestTarGz(t *testing.T, archivePath string, files map[string]string) {
    t.Helper()
    if err := os.MkdirAll(filepath.Dir(archivePath), 0755); err != nil {
        t.Fatal(err)
    }
    f, err := os.Create(archivePath)
    if err != nil {
        t.Fatal(err)
    }
    defer f.Close()

    gz := gzip.NewWriter(f)
    defer gz.Close()

    tw := tar.NewWriter(gz)
    defer tw.Close()

    for name, content := range files {
        hdr := &tar.Header{
            Name: name,
            Mode: 0644,
            Size: int64(len(content)),
        }
        if err := tw.WriteHeader(hdr); err != nil {
            t.Fatal(err)
        }
        if _, err := tw.Write([]byte(content)); err != nil {
            t.Fatal(err)
        }
    }
}

func TestResolveScopeVolumes_AndSelectTargetVolumes(t *testing.T) {
    runner := NewRunner(true, false) // DryRun: true で YAML 解析パスを検証
    client := NewComposeClient(runner, "docker compose")

    tempDir := t.TempDir()
    composeContent := `name: sample
volumes:
    db-data:
    app-data:
        name: custom-app-vol
`
    if err := os.WriteFile(filepath.Join(tempDir, "compose.yml"), []byte(composeContent), 0644); err != nil {
        t.Fatal(err)
    }

    opts := ComposeOptions{
        WorkDir:      tempDir,
        ComposeFiles: []string{"compose.yml"},
    }

    defined, projName, err := client.ResolveScopeVolumes(opts, "sample")
    if err != nil {
        t.Fatalf("ResolveScopeVolumes failed: %v", err)
    }
    if projName != "sample" {
        t.Errorf("期待されるプロジェクト名は 'sample' ですが '%s' でした", projName)
    }
    if len(defined) != 2 {
        t.Fatalf("期待されるボリューム数は 2 ですが %d でした", len(defined))
    }

    t.Run("引数未指定時は定義済み全ボリュームが選択されること", func(t *testing.T) {
        selected, err := SelectTargetVolumes(defined, nil, "project 'sample'")
        if err != nil {
            t.Fatalf("SelectTargetVolumes failed: %v", err)
        }
        if len(selected) != 2 {
            t.Errorf("期待される選択数は 2 ですが %d でした", len(selected))
        }
    })

    t.Run("特定ボリュームのみを指定して絞り込めること", func(t *testing.T) {
        selected, err := SelectTargetVolumes(defined, []string{"db-data"}, "project 'sample'")
        if err != nil {
            t.Fatalf("SelectTargetVolumes failed: %v", err)
        }
        if len(selected) != 1 || selected[0].LogicalName != "db-data" || selected[0].ActualName != "sample_db-data" {
            t.Errorf("期待値と異なります: %+v", selected)
        }
    })

    t.Run("スコープ外のボリューム名を指定した場合はエラーとして拒否されること", func(t *testing.T) {
        _, err := SelectTargetVolumes(defined, []string{"other-project-vol"}, "project 'sample'")
        if err == nil {
            t.Errorf("スコープ外ボリューム指定時にエラーが返されませんでした")
        }
    })
}

func TestInspectBackupArchive(t *testing.T) {
    tempDir := t.TempDir()

    t.Run("正常なアーカイブから論理ボリューム一覧を取得できること", func(t *testing.T) {
        archivePath := filepath.Join(tempDir, "valid.tar.gz")
        createTestTarGz(t, archivePath, map[string]string{
            "db-data/mysql/ibdata1": "dummy-db",
            "app-data/env.txt":      "dummy-app",
        })

        vols, err := InspectBackupArchive(archivePath)
        if err != nil {
            t.Fatalf("InspectBackupArchive failed: %v", err)
        }
        expected := []string{"app-data", "db-data"}
        if !reflect.DeepEqual(vols, expected) {
            t.Errorf("不一致: 期待値 %v, 実際 %v", expected, vols)
        }
    })

    t.Run("存在しないファイルや空ファイル、破損ファイルはエラーになること", func(t *testing.T) {
        if _, err := InspectBackupArchive(filepath.Join(tempDir, "not-found.tar.gz")); err == nil {
            t.Errorf("存在しないファイルでエラーになりませんでした")
        }

        emptyPath := filepath.Join(tempDir, "empty.tar.gz")
        if err := os.WriteFile(emptyPath, nil, 0644); err != nil {
            t.Fatal(err)
        }
        if _, err := InspectBackupArchive(emptyPath); err == nil {
            t.Errorf("空ファイルでエラーになりませんでした")
        }

        corruptPath := filepath.Join(tempDir, "corrupt.tar.gz")
        if err := os.WriteFile(corruptPath, []byte("not-a-valid-gzip-stream"), 0644); err != nil {
            t.Fatal(err)
        }
        if _, err := InspectBackupArchive(corruptPath); err == nil {
            t.Errorf("破損アーカイブでエラーになりませんでした")
        }
    })

    t.Run("パストラバーサルを含む不正なアーカイブは拒否されること", func(t *testing.T) {
        evilPath := filepath.Join(tempDir, "evil.tar.gz")
        createTestTarGz(t, evilPath, map[string]string{
            "../outside.txt": "malicious",
        })
        if _, err := InspectBackupArchive(evilPath); err == nil {
            t.Errorf("パストラバーサルを含むアーカイブでエラーになりませんでした")
        }
    })
}

func TestBuildBackupAndRestoreArgs(t *testing.T) {
    targets := []VolumeMapping{
        {LogicalName: "db-data", ActualName: "sample_db-data", ComposeProject: "sample"},
    }

    t.Run("バックアップコマンドの構築で対象ボリュームが読み取り専用(:ro)マウントされること", func(t *testing.T) {
        args := buildBackupDockerArgs(targets, "/tmp/backups/sample.tar.gz")
        joined := strings.Join(args, " ")
        if !strings.Contains(joined, "-v sample_db-data:/volumes/db-data:ro") {
            t.Errorf("読み取り専用マウントが含まれていません: %s", joined)
        }
        if !strings.Contains(joined, "tar -czf /backup/sample.tar.gz -C /volumes db-data") {
            t.Errorf("tar コマンドが期待値と異なります: %s", joined)
        }
    })

    t.Run("ボリューム作成引数にCompose互換ラベルが付与されること", func(t *testing.T) {
        args := buildVolumeCreateArgs(targets[0])
        expected := []string{
            "volume", "create",
            "--label", "com.docker.compose.project=sample",
            "--label", "com.docker.compose.volume=db-data",
            "sample_db-data",
        }
        if !reflect.DeepEqual(args, expected) {
            t.Errorf("不一致: 期待値 %v, 実際 %v", expected, args)
        }
    })

    t.Run("デフォルトのバックアップパスが <workdir>/.devctl/backups/<scope>-<timestamp>.tar.gz になること", func(t *testing.T) {
        fixedTime := time.Date(2026, 10, 31, 19, 30, 0, 0, time.UTC)
        got := DefaultBackupArchivePath("/workspace/sample", "sample", fixedTime)
        expected := "/workspace/sample/.devctl/backups/sample-20261031-193000.tar.gz"
        if got != expected {
            t.Errorf("不一致: 期待値 %s, 実際 %s", expected, got)
        }
    })
}

func TestBackupAndRestoreVolumes_ExecutionAndSafety(t *testing.T) {
    tempDir := t.TempDir()
    composeContent := `name: sample
volumes:
    db-data:
    app-data:
`
    if err := os.WriteFile(filepath.Join(tempDir, "compose.yml"), []byte(composeContent), 0644); err != nil {
        t.Fatal(err)
    }

    t.Run("DryRunでBackupおよびRestoreが正常終了すること", func(t *testing.T) {
        runner := NewRunner(true, false)
        client := NewComposeClient(runner, "docker compose")

        vopts := VolumeBackupOptions{
            ComposeOpts: ComposeOptions{
                WorkDir:      tempDir,
                ComposeFiles: []string{"compose.yml"},
            },
            ScopeType: "project",
            ScopeName: "sample",
        }

        outPath, err := client.BackupVolumes(vopts)
        if err != nil {
            t.Fatalf("BackupVolumes DryRun failed: %v", err)
        }
        if !strings.Contains(outPath, filepath.Join(".devctl", "backups", "sample-")) {
            t.Errorf("デフォルト保存先パスが期待値と異なります: %s", outPath)
        }

        // テスト用アーカイブを作成して Restore を検証
        archivePath := filepath.Join(tempDir, "backup.tar.gz")
        createTestTarGz(t, archivePath, map[string]string{
            "db-data/dump.sql": "SELECT 1;",
        })

        vopts.ArchivePath = archivePath
        vopts.TargetVolumes = []string{"db-data"}
        if err := client.RestoreVolumes(vopts); err != nil {
            t.Fatalf("RestoreVolumes DryRun failed: %v", err)
        }
    })

    t.Run("アーカイブに含まれないボリュームのRestore指定はエラーになること", func(t *testing.T) {
        runner := NewRunner(true, false)
        client := NewComposeClient(runner, "docker compose")

        archivePath := filepath.Join(tempDir, "only-db.tar.gz")
        createTestTarGz(t, archivePath, map[string]string{
            "db-data/dump.sql": "SELECT 1;",
        })

        vopts := VolumeBackupOptions{
            ComposeOpts: ComposeOptions{
                WorkDir:      tempDir,
                ComposeFiles: []string{"compose.yml"},
            },
            ScopeType:     "project",
            ScopeName:     "sample",
            ArchivePath:   archivePath,
            TargetVolumes: []string{"app-data"},
        }
        if err := client.RestoreVolumes(vopts); err == nil {
            t.Errorf("アーカイブ未収録ボリュームの復元指定時にエラーになりませんでした")
        }
    })

    t.Run("他プロジェクトのボリュームを含むアーカイブの無指定Restoreは拒否されること", func(t *testing.T) {
        runner := NewRunner(true, false)
        client := NewComposeClient(runner, "docker compose")

        archivePath := filepath.Join(tempDir, "other-scope.tar.gz")
        createTestTarGz(t, archivePath, map[string]string{
            "traefik-certificates/acme.json": "{}",
        })

        vopts := VolumeBackupOptions{
            ComposeOpts: ComposeOptions{
                WorkDir:      tempDir,
                ComposeFiles: []string{"compose.yml"},
            },
            ScopeType:   "project",
            ScopeName:   "sample",
            ArchivePath: archivePath,
        }
        if err := client.RestoreVolumes(vopts); err == nil {
            t.Errorf("管理対象外ボリュームを含むアーカイブでエラーになりませんでした")
        }
    })

    t.Run("稼働中コンテナがある場合はBackupおよびRestoreが拒否され事前stopが案内されること", func(t *testing.T) {
        // `ps --services --status running` に対して稼働中サービス名 "db" を返すモックスクリプトを作成
        mockScript := filepath.Join(tempDir, "mock-compose.sh")
        scriptBody := `#!/bin/sh
echo "db"
`
        if err := os.WriteFile(mockScript, []byte(scriptBody), 0755); err != nil {
            t.Fatal(err)
        }

        runner := NewRunner(false, false)
        client := NewComposeClient(runner, mockScript)

        archivePath := filepath.Join(tempDir, "valid-for-running.tar.gz")
        createTestTarGz(t, archivePath, map[string]string{
            "db-data/dump.sql": "SELECT 1;",
        })

        vopts := VolumeBackupOptions{
            ComposeOpts: ComposeOptions{
                WorkDir:      tempDir,
                ComposeFiles: []string{"compose.yml"},
            },
            ScopeType:   "project",
            ScopeName:   "sample",
            ArchivePath: archivePath,
            AssumeYes:   true,
        }

        if _, err := client.BackupVolumes(vopts); err == nil || !strings.Contains(err.Error(), "devctl stop sample") {
            t.Errorf("稼働中コンテナ存在時のBackup拒否メッセージが期待値と異なります: %v", err)
        }

        if err := client.RestoreVolumes(vopts); err == nil || !strings.Contains(err.Error(), "devctl stop sample") {
            t.Errorf("稼働中コンテナ存在時のRestore拒否メッセージが期待値と異なります: %v", err)
        }
    })

    t.Run("Restore確認プロンプトでNが入力された場合や非対話EOF時は中止されること", func(t *testing.T) {
        // running サービスなし（空出力）を返すモックスクリプト
        mockScript := filepath.Join(tempDir, "mock-compose-stopped.sh")
        if err := os.WriteFile(mockScript, []byte("#!/bin/sh\nexit 0\n"), 0755); err != nil {
            t.Fatal(err)
        }

        runner := NewRunner(false, false)
        client := NewComposeClient(runner, mockScript)

        archivePath := filepath.Join(tempDir, "prompt-test.tar.gz")
        createTestTarGz(t, archivePath, map[string]string{
            "db-data/dump.sql": "SELECT 1;",
        })

        vopts := VolumeBackupOptions{
            ComposeOpts: ComposeOptions{
                WorkDir:      tempDir,
                ComposeFiles: []string{"compose.yml"},
            },
            ScopeType:     "project",
            ScopeName:     "sample",
            ArchivePath:   archivePath,
            AssumeYes:     false,
            ConfirmReader: strings.NewReader("n\n"),
        }

        err := client.RestoreVolumes(vopts)
        if err == nil || !strings.Contains(err.Error(), "キャンセル") {
            t.Errorf("確認プロンプト 'n' でキャンセルエラーが返されませんでした: %v", err)
        }

        vopts.ConfirmReader = strings.NewReader("")
        err = client.RestoreVolumes(vopts)
        if err == nil || !strings.Contains(err.Error(), "確認入力が読み取れなかった") {
            t.Errorf("確認プロンプト EOF 時にエラーが返されませんでした: %v", err)
        }
    })
}
