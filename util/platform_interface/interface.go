package platform_interface

import "telego/util"

// Platform -> {GeminiPlatform, VastPlatform...}
type Platform interface {
	GetPlatformName() string
}

type GeminiPlatform struct{}

func (g GeminiPlatform) GetPlatformName() string {
	return "gemini"
}

// UserStorageInfoGetter -> {GeminiServer}
type UserStorageInfoGetter interface {
	GetAllStorageByUser(username, password string) ([]util.UserOneStorageSet, error)
}
