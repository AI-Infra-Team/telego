package app

import (
	"telego/util"
	"testing"
)

func TestBinManagerWinfsp_CompleteFlow(t *testing.T) {
	manager := BinManagerWinfsp{}

	// 1. 首先判断是不是win，不是win就直接返回成功
	{
		if !util.IsWindows() {
			result := manager.CheckInstalled()
			if !result {
				t.Fatalf("CheckInstalled() on non-Windows system should return true, got %v", result)
			}
			t.Logf("Non-Windows system: CheckInstalled() = %v (expected true)", result)
			return
		}
		t.Log("Running on Windows system, proceeding with full test flow...")
	}

	// 2. 是win的话，异步启动一次telego cmd --cmd dummycmd
	// telego会安装rclone，然后校验rclone有没有安装
	{
		t.Log("Step 2: Installing rclone via telego...")

		// 执行 telego cmd --cmd dummycmd 来触发 rclone 安装
		util.ModRunCmd.NewBuilder("telego", "cmd", "--cmd", "echo dummy").BlockRun()

		// 校验 rclone 是否安装
		_, err := util.ModRunCmd.NewBuilder("rclone", "version").BlockRun()
		if err != nil {
			t.Fatalf("rclone is not available after telego installation: %v", err)
		}
		t.Log("rclone is available")
	}

	// 3. 执行对应的CheckInstalled，记录初始状态
	var initialResult bool
	{
		t.Log("Step 3: Checking initial WinFSP installation status...")
		initialResult = manager.CheckInstalled()
		t.Logf("Initial CheckInstalled() result: %v", initialResult)

		if initialResult {
			t.Log("WinFSP appears to be already installed")
		} else {
			t.Log("WinFSP appears to be not installed")
		}
	}

	// 4. 安装winfsp
	{
		t.Log("Step 4: Installing WinFSP...")

		if !initialResult {
			// 只有在未安装时才尝试安装
			t.Log("Attempting to install WinFSP...")

			// 这里应该实际安装WinFSP，但由于需要管理员权限和下载，我们先检查基本条件
			// 在实际测试中，这里应该调用实际的安装逻辑

			// 检查是否有基本的安装条件（比如网络连接等）
			if !util.HasNetwork() {
				t.Fatal("No network connection available for WinFSP installation")
			}

			t.Log("WinFSP installation prerequisites checked")
			// TODO: 在这里添加实际的WinFSP安装逻辑
			t.Log("WinFSP installation completed (simulated)")
		} else {
			t.Log("WinFSP already installed, skipping installation step")
		}
	}

	// 5. 再次执行CheckInstalled，判断是否为true
	var finalResult bool
	{
		t.Log("Step 5: Checking WinFSP installation status after installation...")
		finalResult = manager.CheckInstalled()
		t.Logf("Final CheckInstalled() result: %v", finalResult)

		if !finalResult {
			t.Fatal("WinFSP installation verification failed - CheckInstalled() returned false")
		}
		t.Log("WinFSP installation verification successful")
	}
}
