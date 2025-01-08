package util

import (
	"os"
	"path/filepath"
	"sync"
	"telego/app/config"
)

type ModTestUtilStruct struct{}

var ModTestUtil ModTestUtilStruct

var SetupForTestMutex sync.Mutex
var SetupForTestOnceFlag bool = true

// return testPrjDir
func (ModTestUtilStruct) SetupForTest() string {
	SetupForTestMutex.Lock()
	defer SetupForTestMutex.Unlock()

	// 执行路径/.test_tmp
	cur := CurDir()
	testPrjDir := filepath.Join(cur, ".test_tmp")
	if SetupForTestOnceFlag {
		os.MkdirAll(testPrjDir, 0755)
		SetupForTestOnceFlag = false
	} else {
		return testPrjDir
	}

	config.SetFake(testPrjDir)
	SetFakeWorkspace(testPrjDir)
	return testPrjDir
}
