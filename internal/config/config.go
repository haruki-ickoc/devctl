package config

import (
    "fmt"
    "os"
    "path/filepath"
    "strings"

    "gopkg.in/yaml.v3"
)

// Config は devctl の全体設定を表す構造体です。
type Config struct {
    Version  string                 `yaml:"version"`
    Defaults DefaultsConfig         `yaml:"defaults"`
    Network  NetworkConfig          `yaml:"network"`
    Core     CoreConfig             `yaml:"core"`
    Projects map[string]ProjectItem `yaml:"projects"`

    // LoadedPath は読み込まれた設定ファイルのパスを保持します
    LoadedPath string `yaml:"-"`
}

// DefaultsConfig はグローバルのデフォルト設定を定義します。
type DefaultsConfig struct {
    ComposeCmd string `yaml:"compose_cmd"` // 例: "docker compose" または "docker-compose"
    EnvFile    string `yaml:"env_file"`    // デフォルトの環境変数ファイル名
}

// NetworkConfig は共通 Docker ネットワークの定義を表します。
type NetworkConfig struct {
    Name       string `yaml:"name"`
    Driver     string `yaml:"driver"`
    Attachable bool   `yaml:"attachable"`
    Subnet     string `yaml:"subnet"`
    Gateway    string `yaml:"gateway"`
}

// CoreConfig は共通基盤（リバースプロキシ、ログ、共通DB等）の定義を表します。
type CoreConfig struct {
    WorkDir      string                 `yaml:"workdir"`
    ComposeFiles []string               `yaml:"compose_files"`
    EnvFile      string                 `yaml:"env_file"`
    Services     map[string]ServiceItem `yaml:"services"`
}

// ServiceItem は Core 内のサービスメタデータを表します。
type ServiceItem struct {
    Description string `yaml:"description"`
}

// ProjectItem は個々の管理対象プロジェクトを表します。
type ProjectItem struct {
    Name         string              `yaml:"-"` // マップのキー
    Description  string              `yaml:"description"`
    WorkDir      string              `yaml:"workdir"`
    ComposeFiles []string            `yaml:"compose_files"`
    EnvFile      string              `yaml:"env_file"`
    DependsOn    DependsOnConfig     `yaml:"depends_on"`
    Tasks        map[string]TaskItem `yaml:"tasks"`
}

// DependsOnConfig はプロジェクト起動に必要な前提インフラ依存を定義します。
type DependsOnConfig struct {
    Network bool     `yaml:"network"` // 共通ネットワークの存在を要求するか
    Core    []string `yaml:"core"`    // 依存する Core サービス名（空の場合は不要）
}

// TaskItem はプロジェクト固有のカスタムタスク（Makefile代替コマンド）を定義します。
type TaskItem struct {
    Description string `yaml:"description"`
    Command     string `yaml:"command"`
}

// candidateComposeFiles は Compose ファイル自動検出時の候補ファイル名一覧（優先順位順）です。
var candidateComposeFiles = []string{
    "compose.yaml",
    "compose.yml",
    "docker-compose.yaml",
    "docker-compose.yml",
}

// detectComposeFiles は指定された作業ディレクトリ内に存在する Compose ファイルを自動検出します。
// 見つからない場合は現代の推奨デフォルトである compose.yml を返します。
func detectComposeFiles(workDir string) []string {
    if workDir != "" {
        for _, candidate := range candidateComposeFiles {
            target := filepath.Join(workDir, candidate)
            if _, err := os.Stat(target); err == nil {
                return []string{candidate}
            }
        }
    }
    return []string{"compose.yml"}
}

// ValidateFile は指定されたパスの設定ファイルを読み込み、YAML 構文および設定構造の正当性を検証します。
func ValidateFile(filePath string) (*Config, error) {
    targetPath := ExpandPath(filePath)
    data, err := os.ReadFile(targetPath)
    if err != nil {
        return nil, fmt.Errorf("設定ファイルの読み込みに失敗しました (%s): %w", targetPath, err)
    }

    // 環境変数の展開
    expanded := os.ExpandEnv(string(data))

    var cfg Config
    if err := yaml.Unmarshal([]byte(expanded), &cfg); err != nil {
        return nil, fmt.Errorf("設定ファイルの YAML 構文エラー: %w", err)
    }

    if cfg.Version == "" {
        return nil, fmt.Errorf("devctl 設定ファイルとして無効です ('version' フィールドが必要です)")
    }

    cfg.LoadedPath = targetPath
    cfg.applyDefaultsAndResolvePaths()

    return &cfg, nil
}

// Load は設定ファイルを探索してパースします。
// customPath が空文字列の場合は既定の候補パスを自動探索します。
func Load(customPath string) (*Config, error) {
    targetPath, err := resolveConfigPath(customPath)
    if err != nil {
        return nil, err
    }

    return ValidateFile(targetPath)
}

// resolveConfigPath は読み込むべき設定ファイルのパスを解決します。
func resolveConfigPath(customPath string) (string, error) {
    if customPath != "" {
        return ExpandPath(customPath), nil
    }

    // 1. 環境変数 DEVCTL_CONFIG の確認
    if envPath := os.Getenv("DEVCTL_CONFIG"); envPath != "" {
        return ExpandPath(envPath), nil
    }

    // 2. カレントディレクトリの確認
    candidates := []string{
        ".devctl.yaml",
        ".devctl.yml",
        "devctl.yaml",
        "devctl.yml",
    }
    for _, c := range candidates {
        if _, err := os.Stat(c); err == nil {
            abs, _ := filepath.Abs(c)
            return abs, nil
        }
    }

    // 3. ~/.config/devctl/config.yaml の確認
    homeDir, err := os.UserHomeDir()
    if err == nil {
        defaultPath := filepath.Join(homeDir, ".config", "devctl", "config.yaml")
        if _, err := os.Stat(defaultPath); err == nil {
            return defaultPath, nil
        }
        // config.yml も確認
        altPath := filepath.Join(homeDir, ".config", "devctl", "config.yml")
        if _, err := os.Stat(altPath); err == nil {
            return altPath, nil
        }
        return defaultPath, fmt.Errorf("設定ファイルが見つかりません。'devctl config init' を実行して初期化してください (%s)", defaultPath)
    }

    return "", fmt.Errorf("設定ファイルが見つかりません。--config を指定するか ~/.config/devctl/config.yaml を作成してください")
}

// DefaultConfigPath は標準の設定ファイル配置パス (~/.config/devctl/config.yaml) を返します。
func DefaultConfigPath() (string, error) {
    homeDir, err := os.UserHomeDir()
    if err != nil {
        return "", err
    }
    return filepath.Join(homeDir, ".config", "devctl", "config.yaml"), nil
}

// applyDefaultsAndResolvePaths は未設定項目へデフォルト値を設定し、パスの解決・Composeファイルの自動検出を行います。
func (c *Config) applyDefaultsAndResolvePaths() {
    if c.Defaults.ComposeCmd == "" {
        c.Defaults.ComposeCmd = "docker compose"
    }
    if c.Network.Name == "" {
        c.Network.Name = "internal-network"
    }
    if c.Network.Driver == "" {
        c.Network.Driver = "bridge"
    }

    // 共通基盤 (Core) のパス解決と Compose ファイル自動検出
    if c.Core.WorkDir != "" {
        c.Core.WorkDir = ExpandPath(c.Core.WorkDir)
    }
    if len(c.Core.ComposeFiles) == 0 {
        c.Core.ComposeFiles = detectComposeFiles(c.Core.WorkDir)
    }

    // 各プロジェクトのパス解決と Compose ファイル自動検出
    for name, proj := range c.Projects {
        proj.Name = name
        if proj.WorkDir != "" {
            proj.WorkDir = ExpandPath(proj.WorkDir)
        }
        if len(proj.ComposeFiles) == 0 {
            proj.ComposeFiles = detectComposeFiles(proj.WorkDir)
        }
        c.Projects[name] = proj
    }
}

// FindProjectByWorkingDir は指定ディレクトリ（通常はカレントディレクトリ）から合致する登録プロジェクトを判定します。
func (c *Config) FindProjectByWorkingDir(currentDir string) (*ProjectItem, bool) {
    cleanCurrent, err := filepath.Abs(currentDir)
    if err != nil {
        return nil, false
    }

    for _, p := range c.Projects {
        if p.WorkDir == "" {
            continue
        }
        cleanProj, err := filepath.Abs(p.WorkDir)
        if err != nil {
            continue
        }
        // 完全一致、またはカレントディレクトリがプロジェクト作業ディレクトリ配下の場合
        if cleanCurrent == cleanProj || strings.HasPrefix(cleanCurrent, cleanProj+string(filepath.Separator)) {
            projCopy := p
            return &projCopy, true
        }
    }
    return nil, false
}

// ExpandPath はチルダ (~) をユーザーのホームディレクトリに展開し、絶対パスに変換します。
func ExpandPath(path string) string {
    if strings.HasPrefix(path, "~/") || path == "~" {
        homeDir, err := os.UserHomeDir()
        if err == nil {
            if path == "~" {
                return homeDir
            }
            return filepath.Join(homeDir, path[2:])
        }
    }
    abs, err := filepath.Abs(path)
    if err != nil {
        return path
    }
    return abs
}
