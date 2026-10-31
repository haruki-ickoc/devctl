package docker

import (
    "archive/tar"
    "bufio"
    "compress/gzip"
    "encoding/json"
    "fmt"
    "io"
    "os"
    "path/filepath"
    "sort"
    "strings"
    "time"

    "gopkg.in/yaml.v3"
    "github.com/haruki-ickoc/devctl/internal/ui"
)

const backupHelperImage = "alpine:latest"

// VolumeMapping は Compose 定義上の論理ボリューム名と実 Docker ボリューム名の対応を表します。
type VolumeMapping struct {
    LogicalName    string
    ActualName     string
    ComposeProject string
    External       bool
}

// VolumeBackupOptions はボリュームのバックアップ・リストア実行時のオプションです。
type VolumeBackupOptions struct {
    ComposeOpts    ComposeOptions
    ScopeType      string // "project" または "core"
    ScopeName      string // プロジェクト名 または "core"
    TargetVolumes  []string
    ArchivePath    string
    AssumeYes      bool
    ConfirmReader  io.Reader
}

// composeConfigJSON は `docker compose config --format json` の出力パース用構造体です。
type composeConfigJSON struct {
    Name    string `json:"name"`
    Volumes map[string]struct {
        Name     string `json:"name"`
        External any    `json:"external"`
    } `json:"volumes"`
}

// composeYAMLFile は Compose YAML ファイルから name と volumes を静的に読み取るための構造体です。
type composeYAMLFile struct {
    Name    string               `yaml:"name"`
    Volumes map[string]yaml.Node `yaml:"volumes"`
}

// resolveComposeFilePaths は buildBaseArgs と同じ優先順位で適用対象の Compose ファイルパス一覧を返します。
func (c *ComposeClient) resolveComposeFilePaths(opts ComposeOptions) []string {
    _, baseArgs := c.buildBaseArgs(opts)
    var files []string
    for i := 0; i < len(baseArgs)-1; i++ {
        if baseArgs[i] == "-f" {
            files = append(files, baseArgs[i+1])
            i++
        }
    }
    return files
}

// expandSimpleEnvVar は `${VAR:-default}` 形式の簡易デフォルト値を展開します。
func expandSimpleEnvVar(s string) string {
    for {
        start := strings.Index(s, "${")
        if start < 0 {
            break
        }
        end := strings.Index(s[start:], "}")
        if end < 0 {
            break
        }
        end += start
        expr := s[start+2 : end]
        var replacement string
        if idx := strings.Index(expr, ":-"); idx >= 0 {
            varName := expr[:idx]
            defVal := expr[idx+2:]
            if val := os.Getenv(varName); val != "" {
                replacement = val
            } else {
                replacement = defVal
            }
        } else {
            replacement = os.Getenv(expr)
        }
        s = s[:start] + replacement + s[end+1:]
    }
    return s
}

// ResolveScopeVolumes は対象スコープ（project / core）の Compose 設定で定義されている Named Volume 一覧を解決します。
func (c *ComposeClient) ResolveScopeVolumes(opts ComposeOptions, defaultProjectName string) ([]VolumeMapping, string, error) {
    // 1. 非 DryRun 時は `docker compose config --format json` での解決を試みる
    if !c.runner.DryRun {
        bin, baseArgs := c.buildBaseArgs(opts)
        args := append(baseArgs, "config", "--format", "json")
        if out, err := c.runner.RunCommandOutput(opts.WorkDir, bin, args...); err == nil && strings.HasPrefix(strings.TrimSpace(out), "{") {
            var parsed composeConfigJSON
            if jsonErr := json.Unmarshal([]byte(out), &parsed); jsonErr == nil {
                projName := parsed.Name
                if projName == "" {
                    projName = defaultProjectName
                }
                var mappings []VolumeMapping
                for logical, v := range parsed.Volumes {
                    actual := v.Name
                    if actual == "" {
                        actual = fmt.Sprintf("%s_%s", projName, logical)
                    }
                    isExt := false
                    if b, ok := v.External.(bool); ok && b {
                        isExt = true
                    }
                    mappings = append(mappings, VolumeMapping{
                        LogicalName:    logical,
                        ActualName:     actual,
                        ComposeProject: projName,
                        External:       isExt,
                    })
                }
                sort.Slice(mappings, func(i, j int) bool {
                    return mappings[i].LogicalName < mappings[j].LogicalName
                })
                return mappings, projName, nil
            }
        }
    }

    // 2. Compose YAML ファイルの直接パース（DryRun 時およびフォールバック）
    files := c.resolveComposeFilePaths(opts)
    projName := ""
    volMap := make(map[string]VolumeMapping)

    for _, relFile := range files {
        fullPath := relFile
        if opts.WorkDir != "" && !filepath.IsAbs(fullPath) {
            fullPath = filepath.Join(opts.WorkDir, relFile)
        }
        data, err := os.ReadFile(fullPath)
        if err != nil {
            continue
        }
        var parsed composeYAMLFile
        if err := yaml.Unmarshal(data, &parsed); err != nil {
            return nil, "", fmt.Errorf("Compose ファイル (%s) の解析に失敗しました: %w", relFile, err)
        }
        if parsed.Name != "" {
            projName = expandSimpleEnvVar(strings.TrimSpace(parsed.Name))
        }
        for logical, node := range parsed.Volumes {
            vm := VolumeMapping{
                LogicalName: logical,
            }
            if node.Kind == yaml.MappingNode {
                var detail struct {
                    Name     string `yaml:"name"`
                    External bool   `yaml:"external"`
                }
                if err := node.Decode(&detail); err == nil {
                    vm.ActualName = expandSimpleEnvVar(strings.TrimSpace(detail.Name))
                    vm.External = detail.External
                }
            }
            volMap[logical] = vm
        }
    }

    if projName == "" {
        if defaultProjectName != "" && defaultProjectName != "core" {
            projName = defaultProjectName
        } else if opts.WorkDir != "" {
            projName = filepath.Base(opts.WorkDir)
        } else {
            projName = defaultProjectName
        }
    }

    var mappings []VolumeMapping
    for _, vm := range volMap {
        vm.ComposeProject = projName
        if vm.ActualName == "" {
            vm.ActualName = fmt.Sprintf("%s_%s", projName, vm.LogicalName)
        }
        mappings = append(mappings, vm)
    }
    sort.Slice(mappings, func(i, j int) bool {
        return mappings[i].LogicalName < mappings[j].LogicalName
    })
    return mappings, projName, nil
}

// SelectTargetVolumes は定義済みボリューム一覧から、ユーザー指定のボリューム（未指定時は全件）を抽出・検証します。
// スコープ外のボリューム名が指定された場合はエラーを返します。
func SelectTargetVolumes(defined []VolumeMapping, requested []string, scopeLabel string) ([]VolumeMapping, error) {
    if len(defined) == 0 {
        return nil, fmt.Errorf("%s の Compose 定義に管理対象の Named Volume が存在しません", scopeLabel)
    }

    byLogical := make(map[string]VolumeMapping, len(defined))
    byActual := make(map[string]VolumeMapping, len(defined))
    var availableNames []string
    for _, v := range defined {
        byLogical[v.LogicalName] = v
        byActual[v.ActualName] = v
        availableNames = append(availableNames, v.LogicalName)
    }

    if len(requested) == 0 {
        return defined, nil
    }

    var selected []VolumeMapping
    seen := make(map[string]bool)
    for _, req := range requested {
        req = strings.TrimSpace(req)
        if req == "" {
            continue
        }
        vm, ok := byLogical[req]
        if !ok {
            vm, ok = byActual[req]
        }
        if !ok {
            return nil, fmt.Errorf("指定されたボリューム '%s' は %s の管理対象外です (利用可能なボリューム: %s)", req, scopeLabel, strings.Join(availableNames, ", "))
        }
        if !seen[vm.LogicalName] {
            seen[vm.LogicalName] = true
            selected = append(selected, vm)
        }
    }
    return selected, nil
}

// VerifyVolumesNotInUse は対象スコープのサービスおよび対象ボリュームを使用中のコンテナが稼働していないことを検証します。
func (c *ComposeClient) VerifyVolumesNotInUse(opts ComposeOptions, scopeType, scopeName string, targets []VolumeMapping) error {
    stopGuide := fmt.Sprintf("devctl stop %s", scopeName)
    if scopeType == "core" {
        stopGuide = "devctl core stop"
    }

    if c.runner.DryRun {
        ui.Dim("[dry-run] %s の稼働状態およびボリューム使用状況チェックをシミュレーション（停止済みと仮定）", scopeName)
        return nil
    }

    // 1. Compose スコープ内の稼働中サービスチェック
    runningServices, err := c.RunningServices(opts)
    if err != nil {
        return err
    }
    if len(runningServices) > 0 {
        return fmt.Errorf(
            "%s '%s' のコンテナが稼働中です (%s)。データの整合性を保護するため、稼働中の状態では実行できません。先に '%s' で一時停止してから再実行してください",
            scopeType, scopeName, strings.Join(runningServices, ", "), stopGuide,
        )
    }

    // 2. 各対象ボリュームをマウントして稼働中のコンテナがないかチェック
    for _, vol := range targets {
        out, err := c.runner.RunCommandOutput("", "docker", "ps", "--filter", fmt.Sprintf("volume=%s", vol.ActualName), "--format", "{{.Names}}")
        if err != nil {
            return fmt.Errorf("ボリューム '%s' の使用状況確認に失敗しました: %w", vol.ActualName, err)
        }
        if strings.TrimSpace(out) != "" {
            containers := strings.Join(strings.Split(strings.TrimSpace(out), "\n"), ", ")
            return fmt.Errorf(
                "ボリューム '%s' (%s) は稼働中のコンテナ (%s) で使用中です。先に '%s' でコンテナを一時停止してから再実行してください",
                vol.LogicalName, vol.ActualName, containers, stopGuide,
            )
        }
    }

    return nil
}

// VolumeExists は指定した実 Docker ボリュームが存在するか確認します。
func (c *ComposeClient) VolumeExists(actualName string) (bool, error) {
    if c.runner.DryRun {
        ui.Dim("[dry-run] Docker ボリューム '%s' の存在確認シミュレーション", actualName)
        return true, nil
    }
    out, err := c.runner.RunCommandOutput("", "docker", "volume", "ls", "--filter", fmt.Sprintf("name=^%s$", actualName), "--format", "{{.Name}}")
    if err != nil {
        return false, fmt.Errorf("Docker ボリューム一覧の取得に失敗しました: %w", err)
    }
    for _, line := range strings.Split(out, "\n") {
        if strings.TrimSpace(line) == actualName {
            return true, nil
        }
    }
    return false, nil
}

// EnsureVolumeExists は復元対象の Docker ボリュームが存在しない場合に Compose 互換ラベル付きで作成します。
func (c *ComposeClient) EnsureVolumeExists(vol VolumeMapping) error {
    if c.runner.DryRun {
        args := buildVolumeCreateArgs(vol)
        ui.Dim("[dry-run] ボリューム未存在時は自動作成: docker %s", strings.Join(args, " "))
        return nil
    }

    exists, err := c.VolumeExists(vol.ActualName)
    if err != nil {
        return err
    }
    if exists {
        return nil
    }

    ui.Info("復元先の Docker ボリューム '%s' (論理名: %s) が存在しないため新規作成します...", vol.ActualName, vol.LogicalName)
    args := buildVolumeCreateArgs(vol)
    if err := c.runner.RunCommand("", "docker", args...); err != nil {
        return fmt.Errorf("ボリューム '%s' の作成に失敗しました: %w", vol.ActualName, err)
    }
    return nil
}

func buildVolumeCreateArgs(vol VolumeMapping) []string {
    args := []string{"volume", "create"}
    if vol.ComposeProject != "" {
        args = append(args, "--label", fmt.Sprintf("com.docker.compose.project=%s", vol.ComposeProject))
    }
    if vol.LogicalName != "" {
        args = append(args, "--label", fmt.Sprintf("com.docker.compose.volume=%s", vol.LogicalName))
    }
    args = append(args, vol.ActualName)
    return args
}

// DefaultBackupArchivePath はデフォルトのバックアップ保存先パス（<WorkDir>/.devctl/backups/<scopeName>-<timestamp>.tar.gz）を生成します。
func DefaultBackupArchivePath(workDir, scopeName string, now time.Time) string {
    ts := now.Format("20060102-150405")
    fileName := fmt.Sprintf("%s-%s.tar.gz", scopeName, ts)
    return filepath.Join(workDir, ".devctl", "backups", fileName)
}

// buildBackupDockerArgs はバックアップ用の一時コンテナ実行引数を構築します。
func buildBackupDockerArgs(targets []VolumeMapping, absArchivePath string) []string {
    archiveDir := filepath.Dir(absArchivePath)
    archiveFile := filepath.Base(absArchivePath)

    args := []string{"run", "--rm"}
    var logicalNames []string
    for _, vol := range targets {
        args = append(args, "-v", fmt.Sprintf("%s:/volumes/%s:ro", vol.ActualName, vol.LogicalName))
        logicalNames = append(logicalNames, vol.LogicalName)
    }
    args = append(args, "-v", fmt.Sprintf("%s:/backup", archiveDir))
    args = append(args, backupHelperImage)
    args = append(args, "tar", "-czf", fmt.Sprintf("/backup/%s", archiveFile), "-C", "/volumes")
    args = append(args, logicalNames...)
    return args
}

// buildRestoreDockerArgs はリストア用の一時コンテナ実行引数を構築します。
func buildRestoreDockerArgs(targets []VolumeMapping, absArchivePath string) []string {
    archiveDir := filepath.Dir(absArchivePath)
    archiveFile := filepath.Base(absArchivePath)

    args := []string{"run", "--rm"}
    var cleanParts []string
    var logicalNames []string
    for _, vol := range targets {
        args = append(args, "-v", fmt.Sprintf("%s:/volumes/%s", vol.ActualName, vol.LogicalName))
        cleanParts = append(cleanParts, fmt.Sprintf("rm -rf /volumes/%s/* /volumes/%s/.[!.]* /volumes/%s/..?*", vol.LogicalName, vol.LogicalName, vol.LogicalName))
        logicalNames = append(logicalNames, vol.LogicalName)
    }
    args = append(args, "-v", fmt.Sprintf("%s:/backup:ro", archiveDir))
    args = append(args, backupHelperImage)

    tarExtract := fmt.Sprintf("tar -xzf /backup/%s -C /volumes %s", archiveFile, strings.Join(logicalNames, " "))
    fullScript := fmt.Sprintf("%s && %s", strings.Join(cleanParts, " && "), tarExtract)
    args = append(args, "sh", "-c", fullScript)
    return args
}

// InspectBackupArchive はバックアップアーカイブ (.tar.gz) の整合性を検証し、収録されている論理ボリューム名一覧を返します。
func InspectBackupArchive(archivePath string) ([]string, error) {
    info, err := os.Stat(archivePath)
    if err != nil {
        if os.IsNotExist(err) {
            return nil, fmt.Errorf("指定されたバックアップファイルが存在しません: %s", archivePath)
        }
        return nil, fmt.Errorf("バックアップファイルの確認に失敗しました: %w", err)
    }
    if info.IsDir() {
        return nil, fmt.Errorf("指定されたパスはディレクトリです。バックアップアーカイブファイル (.tar.gz) を指定してください: %s", archivePath)
    }
    if info.Size() == 0 {
        return nil, fmt.Errorf("バックアップファイルが空 (0 byte) です: %s", archivePath)
    }

    f, err := os.Open(archivePath)
    if err != nil {
        return nil, fmt.Errorf("バックアップファイルを開けませんでした: %w", err)
    }
    defer f.Close()

    gz, err := gzip.NewReader(f)
    if err != nil {
        return nil, fmt.Errorf("不正な gzip アーカイブ形式です (%s): %w", archivePath, err)
    }
    defer gz.Close()

    tr := tar.NewReader(gz)
    volSet := make(map[string]bool)
    entryCount := 0

    for {
        hdr, err := tr.Next()
        if err == io.EOF {
            break
        }
        if err != nil {
            return nil, fmt.Errorf("バックアップアーカイブ (%s) が破損しているか不正な tar 形式です: %w", archivePath, err)
        }
        entryCount++

        cleanName := filepath.Clean(strings.TrimPrefix(hdr.Name, "./"))
        if cleanName == "." || cleanName == "" {
            continue
        }
        if strings.HasPrefix(cleanName, "/") || strings.HasPrefix(cleanName, "..") || strings.Contains(cleanName, "/../") {
            return nil, fmt.Errorf("バックアップアーカイブに不正なパス (%s) が含まれています", hdr.Name)
        }

        parts := strings.Split(cleanName, "/")
        topDir := strings.TrimSpace(parts[0])
        if topDir != "" {
            volSet[topDir] = true
        }
    }

    if entryCount == 0 || len(volSet) == 0 {
        return nil, fmt.Errorf("バックアップアーカイブ (%s) にボリュームデータが含まれていません", archivePath)
    }

    var volumes []string
    for v := range volSet {
        volumes = append(volumes, v)
    }
    sort.Strings(volumes)
    return volumes, nil
}

// BackupVolumes は対象スコープの Named Volume をアーカイブ (.tar.gz) に退避します。
func (c *ComposeClient) BackupVolumes(vopts VolumeBackupOptions) (string, error) {
    scopeLabel := fmt.Sprintf("%s '%s'", vopts.ScopeType, vopts.ScopeName)
    if vopts.ScopeType == "core" {
        scopeLabel = "共通基盤 (core)"
    }

    defined, _, err := c.ResolveScopeVolumes(vopts.ComposeOpts, vopts.ScopeName)
    if err != nil {
        return "", err
    }

    targets, err := SelectTargetVolumes(defined, vopts.TargetVolumes, scopeLabel)
    if err != nil {
        return "", err
    }

    // 稼働中コンテナの存在チェック（コンテナ削除は行わず、稼働中なら拒否して事前 stop を案内）
    if err := c.VerifyVolumesNotInUse(vopts.ComposeOpts, vopts.ScopeType, vopts.ScopeName, targets); err != nil {
        return "", err
    }

    // 実 Docker Volume の存在確認
    if !c.runner.DryRun {
        for _, vol := range targets {
            exists, err := c.VolumeExists(vol.ActualName)
            if err != nil {
                return "", err
            }
            if !exists {
                return "", fmt.Errorf("バックアップ対象の Docker ボリューム '%s' (論理名: %s) が存在しません。コンテナが一度も作成されていない可能性があります", vol.ActualName, vol.LogicalName)
            }
        }
    }

    archivePath := vopts.ArchivePath
    if archivePath == "" {
        archivePath = DefaultBackupArchivePath(vopts.ComposeOpts.WorkDir, vopts.ScopeName, time.Now())
    }
    absArchivePath, err := filepath.Abs(archivePath)
    if err != nil {
        return "", fmt.Errorf("出力先パスの解決に失敗しました: %w", err)
    }

    if !c.runner.DryRun {
        if err := os.MkdirAll(filepath.Dir(absArchivePath), 0755); err != nil {
            return "", fmt.Errorf("バックアップ保存先ディレクトリの作成に失敗しました: %w", err)
        }
    }

    ui.Info("%s のボリュームバックアップを開始します:", scopeLabel)
    for _, vol := range targets {
        ui.Info("  - 対象ボリューム: %s (Docker Volume: %s)", vol.LogicalName, vol.ActualName)
    }
    ui.Info("  - 保存先アーカイブ: %s", absArchivePath)

    dockerArgs := buildBackupDockerArgs(targets, absArchivePath)
    if err := c.runner.RunCommand(vopts.ComposeOpts.WorkDir, "docker", dockerArgs...); err != nil {
        return "", fmt.Errorf("ボリュームのバックアップ実行に失敗しました: %w", err)
    }

    if !c.runner.DryRun {
        if _, err := InspectBackupArchive(absArchivePath); err != nil {
            return "", fmt.Errorf("作成されたバックアップアーカイブの検証に失敗しました: %w", err)
        }
    }

    ui.Success("ボリュームのバックアップが完了しました: %s", absArchivePath)
    return absArchivePath, nil
}

// RestoreVolumes はバックアップアーカイブ (.tar.gz) から対象スコープの Named Volume を復元します。
func (c *ComposeClient) RestoreVolumes(vopts VolumeBackupOptions) error {
    scopeLabel := fmt.Sprintf("%s '%s'", vopts.ScopeType, vopts.ScopeName)
    if vopts.ScopeType == "core" {
        scopeLabel = "共通基盤 (core)"
    }

    if strings.TrimSpace(vopts.ArchivePath) == "" {
        return fmt.Errorf("復元元のバックアップアーカイブファイルを指定してください")
    }

    absArchivePath, err := filepath.Abs(vopts.ArchivePath)
    if err != nil {
        return fmt.Errorf("バックアップファイルパスの解決に失敗しました: %w", err)
    }

    defined, _, err := c.ResolveScopeVolumes(vopts.ComposeOpts, vopts.ScopeName)
    if err != nil {
        return err
    }

    // アーカイブの事前検証（DryRun でもファイルが存在する場合は検証を実施）
    var archivedVolumes []string
    if _, statErr := os.Stat(absArchivePath); statErr == nil || !c.runner.DryRun {
        archivedVolumes, err = InspectBackupArchive(absArchivePath)
        if err != nil {
            return err
        }
    }

    var targets []VolumeMapping
    if len(vopts.TargetVolumes) > 0 {
        targets, err = SelectTargetVolumes(defined, vopts.TargetVolumes, scopeLabel)
        if err != nil {
            return err
        }
        if len(archivedVolumes) > 0 {
            archivedSet := make(map[string]bool, len(archivedVolumes))
            for _, av := range archivedVolumes {
                archivedSet[av] = true
            }
            for _, t := range targets {
                if !archivedSet[t.LogicalName] {
                    return fmt.Errorf("指定されたボリューム '%s' はバックアップアーカイブ (%s) 内に含まれていません (収録ボリューム: %s)", t.LogicalName, filepath.Base(absArchivePath), strings.Join(archivedVolumes, ", "))
                }
            }
        }
    } else if len(archivedVolumes) > 0 {
        // アーカイブ内の全ボリュームが当該スコープの管理対象か検証
        targets, err = SelectTargetVolumes(defined, archivedVolumes, scopeLabel)
        if err != nil {
            return fmt.Errorf("バックアップアーカイブ (%s) に %s の管理対象外のボリュームが含まれています: %w", filepath.Base(absArchivePath), scopeLabel, err)
        }
    } else {
        targets, err = SelectTargetVolumes(defined, nil, scopeLabel)
        if err != nil {
            return err
        }
    }

    // 稼働中コンテナの存在チェック（使用中の Volume への restore は拒否）
    if err := c.VerifyVolumesNotInUse(vopts.ComposeOpts, vopts.ScopeType, vopts.ScopeName, targets); err != nil {
        return err
    }

    // 復元対象とアーカイブの事前表示
    ui.Warn("%s のボリューム復元（上書き）を実行しようとしています:", scopeLabel)
    ui.Info("  - 復元元アーカイブ: %s", absArchivePath)
    for _, vol := range targets {
        ui.Info("  - 上書き対象ボリューム: %s (Docker Volume: %s)", vol.LogicalName, vol.ActualName)
    }

    // 確認プロンプト（--yes 未指定かつ非 DryRun 時）
    if !vopts.AssumeYes && !c.runner.DryRun {
        reader := vopts.ConfirmReader
        if reader == nil {
            reader = os.Stdin
        }
        fmt.Print("対象ボリュームの既存データは上書きされます。復元を実行しますか？ [y/N]: ")
        scanner := bufio.NewScanner(reader)
        if !scanner.Scan() {
            return fmt.Errorf("確認入力が読み取れなかったため、復元を中止しました（非対話環境では -y / --yes を指定してください）")
        }
        answer := strings.ToLower(strings.TrimSpace(scanner.Text()))
        if answer != "y" && answer != "yes" {
            return fmt.Errorf("ユーザーにより復元処理がキャンセルされました")
        }
    }

    // ボリュームが存在しない場合は自動作成
    for _, vol := range targets {
        if err := c.EnsureVolumeExists(vol); err != nil {
            return err
        }
    }

    dockerArgs := buildRestoreDockerArgs(targets, absArchivePath)
    if err := c.runner.RunCommand(vopts.ComposeOpts.WorkDir, "docker", dockerArgs...); err != nil {
        return fmt.Errorf("ボリュームの復元実行に失敗しました: %w", err)
    }

    ui.Success("ボリュームの復元が正常に完了しました。")
    return nil
}
