package platform_interface

import "telego/util"

func GetAllStorageByUser(infoGetter UserStorageInfoGetter, username, password string) ([]util.UserOneStorageSet, error) {
	return infoGetter.GetAllStorageByUser(username, password)
}
