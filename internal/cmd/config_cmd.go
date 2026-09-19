package cmd

import (
    "fmt"
    "os"
    "path/filepath"

    "github.com/spf13/cobra"
    "github.com/haruki-ickoc/devctl/internal/config"
    "github.com/haruki-ickoc/devctl/internal/ui"
    "gopkg.in/yaml.v3"
)

var configCmd = &cobra.Command{
    Use:   "config",
    Short: "devctl 設定ファイルの確認、初期化、構文およびパス検証",
}

var configPathCmd = &cobra.Command{
    Use:   "path",
    Short: "現在読み込まれている設定ファイルのパスを表示",
    RunE: func(cmd *cobra.Command, args []string) error {
        if cfg != nil && cfg.LoadedPath != "" {
            fmt.Println(cfg.LoadedPath)
            return nil
        }
        defaultPath, _ := config.DefaultConfigPath()
        fmt.Printf("既定のパス: %s (現在は未作成)\n", defaultPath)
        return nil
    },
}

var configViewCmd = &cobra.Command{
    Use:   "view",
    Short: "現在有効な設定内容を YAML 形式でダンプ出力",
    RunE: func(cmd *cobra.Command, args []string) error {
        if cfg == nil {
            return fmt.Errorf("有効な設定が読み込まれていません")
        }
        data, err := yaml.Marshal(cfg)
        if err != nil {
            return err
        }
        ui.Info("読み込み元: %s の有効な設定内容:\n", cfg.LoadedPath)
        fmt.Println(string(data))
        return nil
    },
}

var (
    configInitFrom  string
    configInitForce bool
)

var configInitCmd = &cobra.Command{
    Use:   "init [source-file]",
    Short: "devctl 設定ファイルの初期化または既存設定ファイルのインポート",
    Long: `~/.config/devctl/config.yaml に設定ファイルを生成または配置します。

以下の2つの機能に対応しています:
  1. 汎用サンプルの初期配置:
     引数を指定せずに実行すると、汎用的なサンプル設定（DefaultConfigYAML）を配置します。
  2. 既存設定ファイルのインポート:
     チーム等で配布された設定ファイルを引数（または --from）で指定すると、構文検証を行った上で配置します。`,
    Example: `  # 汎用サンプル設定を新規生成
  devctl config init

  # 配布された設定ファイルを指定して配置
  devctl config init ./shared-config.yaml
  devctl config init --from ./configs/team.yaml

  # 既存設定ファイルを強制上書き
  devctl config init ./shared-config.yaml -f`,
    RunE: func(cmd *cobra.Command, args []string) error {
        destPath, err := config.DefaultConfigPath()
        if err != nil {
            return err
        }

        // インポート元の指定判定（引数または --from フラグ）
        sourcePath := configInitFrom
        if len(args) > 0 {
            sourcePath = args[0]
        }

        // 既に配置先ファイルが存在するか確認
        if _, err := os.Stat(destPath); err == nil && !configInitForce {
            ui.Warn("設定ファイルは既に存在します: %s", destPath)
            ui.Info("上書きして再配置する場合は --force (-f) フラグを指定してください。")
            return nil
        }

        dir := filepath.Dir(destPath)
        if err := os.MkdirAll(dir, 0755); err != nil {
            return fmt.Errorf("ディレクトリ %s の作成に失敗しました: %w", dir, err)
        }

        if sourcePath != "" {
            // モード1: 既存設定ファイルの検証とコピー配置
            resolvedSource := config.ExpandPath(sourcePath)
            if _, err := os.Stat(resolvedSource); os.IsNotExist(err) {
                return fmt.Errorf("指定された設定ファイルが見つかりません: %s", sourcePath)
            }

            // YAML 構文および構造の事前バリデーション
            if _, err := config.ValidateFile(resolvedSource); err != nil {
                return fmt.Errorf("指定された設定ファイルの構文検証に失敗しました: %w", err)
            }

            data, err := os.ReadFile(resolvedSource)
            if err != nil {
                return fmt.Errorf("設定ファイルの読み込みに失敗しました: %w", err)
            }

            if err := os.WriteFile(destPath, data, 0644); err != nil {
                return fmt.Errorf("設定ファイルの配置に失敗しました: %w", err)
            }

            ui.Success("指定された設定ファイル (%s) を正常に配置しました: %s", sourcePath, destPath)
            ui.Info("'devctl config check' を実行してパスや構成を検証してください。")
            return nil
        }

        // モード2: 汎用サンプルの初期生成
        if err := os.WriteFile(destPath, []byte(config.DefaultConfigYAML), 0644); err != nil {
            return fmt.Errorf("デフォルト設定ファイルの書き込みに失敗しました: %w", err)
        }

        ui.Success("汎用サンプル設定ファイルを初期化しました: %s", destPath)
        ui.Info("各開発環境に合わせてこのファイル (%s) を編集してください。", destPath)
        return nil
    },
}

var configCheckCmd = &cobra.Command{
    Use:   "check",
    Short: "設定ファイルの構文と指定ディレクトリ・Compose ファイルの実在性を検証",
    RunE: func(cmd *cobra.Command, args []string) error {
        if cfg == nil {
            return fmt.Errorf("設定ファイルを読み込めませんでした。'devctl config init' を実行してください")
        }

        ui.Success("設定ファイルの構文は正常です: %s", cfg.LoadedPath)
        fmt.Println()

        // 共通ネットワークの検証
        ui.Step("共通ネットワーク設定の検証:")
        ui.Info("  ネットワーク名: %s", cfg.Network.Name)
        ui.Info("  ドライバ:       %s", cfg.Network.Driver)

        // 共通基盤 (Core) の検証
        ui.Step("共通基盤 (Core) の検証:")
        if cfg.Core.WorkDir == "" {
            ui.Warn("  共通基盤の作業ディレクトリ (workdir) が未設定です")
        } else {
            if info, err := os.Stat(cfg.Core.WorkDir); err != nil || !info.IsDir() {
                ui.Warn("  共通基盤の作業ディレクトリが存在しません: %s", cfg.Core.WorkDir)
            } else {
                ui.Success("  共通基盤の作業ディレクトリが存在します: %s", cfg.Core.WorkDir)
                for _, f := range cfg.Core.ComposeFiles {
                    fullPath := filepath.Join(cfg.Core.WorkDir, f)
                    if _, err := os.Stat(fullPath); err != nil {
                        ui.Warn("    Compose ファイルが見つかりません: %s", fullPath)
                    } else {
                        ui.Success("    Compose ファイルを確認: %s", f)
                    }
                }
            }
        }

        // 管理対象プロジェクトの検証
        ui.Step("管理対象プロジェクトの検証 (%d 件登録):", len(cfg.Projects))
        for name, proj := range cfg.Projects {
            fmt.Printf("  [%s]:\n", name)
            if info, err := os.Stat(proj.WorkDir); err != nil || !info.IsDir() {
                ui.Warn("    作業ディレクトリが存在しません: %s", proj.WorkDir)
            } else {
                ui.Success("    作業ディレクトリが存在します: %s", proj.WorkDir)
                for _, f := range proj.ComposeFiles {
                    fullPath := filepath.Join(proj.WorkDir, f)
                    if _, err := os.Stat(fullPath); err != nil {
                        ui.Warn("      Compose ファイルが見つかりません: %s", fullPath)
                    } else {
                        ui.Success("      Compose ファイルを確認: %s", f)
                    }
                }
            }
            if len(proj.Tasks) > 0 {
                ui.Info("    登録タスク数: %d", len(proj.Tasks))
            }
        }

        fmt.Println()
        ui.Success("設定ファイルの検証が完了しました。")
        return nil
    },
}

func init() {
    RootCmd.AddCommand(configCmd)
    configCmd.AddCommand(configPathCmd)
    configCmd.AddCommand(configViewCmd)
    configCmd.AddCommand(configInitCmd)
    configCmd.AddCommand(configCheckCmd)

    configInitCmd.Flags().StringVar(&configInitFrom, "from", "", "配置元の設定ファイルパスを指定")
    configInitCmd.Flags().BoolVarP(&configInitForce, "force", "f", false, "既存ファイルが存在する場合でも強制上書き")
}
