package docker

import (
    "reflect"
    "testing"
)

func TestNetworkManager_BuildCreateArgs(t *testing.T) {
    runner := NewRunner(true, false)
    nm := NewNetworkManager(runner)

    t.Run("デフォルトのComposeプロジェクト名(toolbox)で互換ラベルが付与されること", func(t *testing.T) {
        args := nm.buildCreateArgs("core-network", "bridge", "10.10.0.0/24", "10.10.0.1", true)
        expected := []string{
            "network", "create",
            "--driver", "bridge",
            "--attachable",
            "--subnet", "10.10.0.0/24",
            "--gateway", "10.10.0.1",
            "--label", "com.docker.compose.network=core-network",
            "--label", "com.docker.compose.project=toolbox",
            "core-network",
        }
        if !reflect.DeepEqual(args, expected) {
            t.Errorf("引数の不一致:\n期待値: %v\n実際値: %v", expected, args)
        }
    })

    t.Run("SetComposeProjectでカスタムプロジェクト名がラベルに反映されること", func(t *testing.T) {
        nm.SetComposeProject("custom-core")
        args := nm.buildCreateArgs("dev-net", "bridge", "", "", false)
        expected := []string{
            "network", "create",
            "--driver", "bridge",
            "--label", "com.docker.compose.network=dev-net",
            "--label", "com.docker.compose.project=custom-core",
            "dev-net",
        }
        if !reflect.DeepEqual(args, expected) {
            t.Errorf("引数の不一致:\n期待値: %v\n実際値: %v", expected, args)
        }
    })

    t.Run("EnsureおよびCreateがDryRunで正常に実行できること", func(t *testing.T) {
        if err := nm.Ensure("core-network", "bridge", "10.10.0.0/24", "10.10.0.1", true); err != nil {
            t.Errorf("Ensure の DryRun 実行に失敗しました: %v", err)
        }
    })
}
