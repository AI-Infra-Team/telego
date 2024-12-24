package app

import (
	"os"
	"strings"
	"telego/app/config"
	"telego/util"
)

func (i *MenuItem) LoadSubPrjs() {
	if i.Name == "deploy" { // this is mapped to project dir

		// i.Children = []*MenuItem{}
		listPrjDir, err := os.ReadDir(config.Load().ProjectDir)
		if err == nil {
			for _, entry := range listPrjDir {
				if entry.IsDir() {
					{
						//sort mappedRes
						var mi *MenuItem
						if strings.HasPrefix(entry.Name(), "bin_") {
							mi = &MenuItem{
								Name:     entry.Name(),
								Comment:  "安装项目",
								Children: []*MenuItem{},
							}
						} else if strings.HasPrefix(entry.Name(), "k8s_") {
							mi = &MenuItem{
								Name:     entry.Name(),
								Comment:  "部署项目",
								Children: []*MenuItem{},
							}
						} else {
							// mi = &MenuItem{
							// 	Name:     entry.Name(),
							// 	Comment:  "未知类型项目",
							// 	Children: []*MenuItem{},
							// }
						}

						if strings.HasPrefix(entry.Name(), "bin_") || strings.HasPrefix(entry.Name(), "k8s_") {
							if util.HasNetwork() {
								mi.Children = append(mi.Children, &MenuItem{
									Name:     "prepare",
									Comment:  "准备资源",
									Children: []*MenuItem{},
								})
							}
							if util.FileServerAccessible() {
								mi.Children = append(mi.Children, &MenuItem{
									Name:     "upload",
									Comment:  "上传文件",
									Children: []*MenuItem{},
								}, &MenuItem{
									Name:     "apply",
									Comment:  "部署、安装",
									Children: []*MenuItem{},
								})
							}
						}

						if mi != nil {
							i.Children = append(i.Children, mi)
						}
					}
				}
			}
		}
	}
}
