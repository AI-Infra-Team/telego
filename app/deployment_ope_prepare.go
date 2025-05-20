package app

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"telego/util"

	"github.com/fatih/color"
	"github.com/mholt/archiver/v3"
)

func DeploymentPrepare(project string, deployment *Deployment) error {
	curDir0 := util.CurDir()
	defer os.Chdir(curDir0)
	os.Chdir(ConfigLoad().ProjectDir)
	fmt.Println("deploymentPrepare", project)
	os.Chdir(project)

	DeploymentOpePretreatment(project, deployment)

	// Step2: Process prepare items
	// fmt.Printf("yml mapped as %v", deployment)
	// os.Chdir("prepare")
	for _, item := range deployment.Prepare {
		fmt.Println()
		if *item.Image != "" {
			// prepare image
			fmt.Printf("Preparing image: %s\n", *item.Image)
			err := ModJobImgPrepare.PrepareImages([]string{*item.Image})
			if err != nil {
				return fmt.Errorf("failed to prepare image %s: %w", *item.Image, err)
			}
		} else if item.Pyscript != nil && *item.Pyscript != "" {
			err := DeploymentPreparePyscript(path.Join(ConfigLoad().ProjectDir, project), &item)
			if err != nil {
				return fmt.Errorf("failed to prepare pyscript %s: %w", *item.Pyscript, err)
			}
		} else if *item.URL != "" {
			err := DeploymentPrepareUrl(path.Join(ConfigLoad().ProjectDir, project), &item)
			if err != nil {
				return fmt.Errorf("failed to prepare url %s: %w", *item.URL, err)
			}
		} else if item.FileMap != nil {
			err := item.FileMap.WriteToFile()
			if err != nil {
				fmt.Println(color.RedString("filemap write failed %v", err))
			}
		}

		// if item.FileMap != nil {
		// 	fmt.Printf("Creating file from map: %s\n", item.FileMap.Path)
		// 	err := createFileFromMap(item.FileMap)
		// 	if err != nil {
		// 		return fmt.Errorf("failed to create file %s: %w", item.FileMap.Path, err)
		// 	}
		// }
	}

	return nil
}

const PrepareCacheDir = "prepare_cache"

func DeploymentPreparePyscript(prjdir string, item *DeploymentPrepareItem) error {
	util.PrintStep("prepare pyscript", fmt.Sprintf("prj: %s", prjdir))
	prjcachedir := path.Join(prjdir, PrepareCacheDir)
	// mkdir
	err := os.MkdirAll(prjcachedir, 0755)
	if err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// remove old pyscript
	prepare_script_prefix := "prepare_script_"
	_, err = util.ModRunCmd.NewBuilder("rm", "-rf", prepare_script_prefix+"*").ShowProgress().SetDir(prjcachedir).BlockRun()
	if err != nil {
		return fmt.Errorf("failed to remove %s_*.py: %w", prepare_script_prefix, err)
	}

	// create new pyscript
	temp, err := os.CreateTemp(prjcachedir, prepare_script_prefix+"*.py")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	temp.WriteString(*item.Pyscript)
	temp.Close()
	// defer os.RemoveAll(temp.Name())

	// run pyscript
	cmds := []string{"python3", temp.Name()}
	if util.IsWindows() {
		cmds[0] = "python"
	}
	_, err = util.ModRunCmd.
		NewBuilder(cmds[0], cmds[1:]...).
		WithRoot().
		SetDir(prjcachedir).
		ShowProgress().
		BlockRun()
	if err != nil {
		return fmt.Errorf("failed to run pyscript %s: %w", *item.Pyscript, err)
	}
	err = DeploymentPrepareHandleCached(item, prjcachedir)
	if err != nil {
		return fmt.Errorf("failed to handle cached: %w", err)
	}
	os.RemoveAll(temp.Name())
	return nil
}

func DeploymentPrepareUrl(prjdir string, item *DeploymentPrepareItem) error {
	util.PrintStep("prepare url", fmt.Sprintf("Downloading file from URL: %s", *item.URL))

	// download to download_cache
	// downloadCachePath := path.Join(PrepareCacheDir, filepath.Base(*item.URL))
	prjcachedir := path.Join(prjdir, PrepareCacheDir)
	downloadCachePath := path.Join(prjcachedir, filepath.Base(*item.URL))
	if _, err := os.Stat(downloadCachePath); err != nil {
		err := util.DownloadFile(*item.URL, downloadCachePath)
		if err != nil {
			// fmt.Println(color.RedString("Failed to download file from %s\n   err: %v", *item.URL, err))
			return fmt.Errorf("failed to download file from %s: %w", *item.URL, err)
		}
	}
	return DeploymentPrepareHandleCached(item, downloadCachePath)
}

func DeploymentPrepareHandleCached(item *DeploymentPrepareItem, downloadCachePath string) error {
	_, err := os.Stat(downloadCachePath)
	if err == nil {
		// Execute "trans" command if provided
		if len(item.Trans) > 0 {
			defer os.RemoveAll("extract")
			extractAppeared := false
			for _, step := range item.Trans {
				switch step := step.(type) {
				case DeploymentTransformExtract:
					extractAppeared = true
					os.RemoveAll("extract")
					fmt.Println(color.BlueString("Extracting file: %s", downloadCachePath))
					err := os.MkdirAll("extract", 0755)
					if err != nil {
						return fmt.Errorf("failed to create directory: %w", err)
					}
					err = archiver.Unarchive(downloadCachePath, "extract")
					if err != nil {
						return fmt.Errorf("failed to extract file: %w", err)
					}
				case DeploymentTransformCopy:
					for _, copyStep := range step.Copy {
						src := path.Join("extract", *copyStep.from)
						if !extractAppeared {
							src = path.Join(downloadCachePath, *copyStep.from)
						}

						dest := *copyStep.to

						fmt.Println(color.BlueString("Copying %s to %s", src, dest))
						newSrc := path.Join(path.Dir(src), path.Base(dest))
						err := os.Rename(src, newSrc)
						if err != nil {
							filetree, err := util.GetFileTree(10, "extract")
							filetreeStr := ""
							if err != nil {
								filetreeStr = fmt.Sprintf("get filetree failed %v", err)
							} else {
								filetreeStr = filetree.GetDebugStr()
							}
							return fmt.Errorf("failed to rename %s to %s: %w, filetree in extract: \n%s", src, dest, err, filetreeStr)
						}

						util.ModRunCmd.CopyDirContentOrFileTo(newSrc, path.Dir(dest))
					}
				}

			}

			// fmt.Println(color.BlueString("Executing user defined transformation: %s at %s\n", item.Trans, CurDir()))
			// err := ModRunCmd.RunCommandShowProgress(item.Trans)
			// if err != nil {
			// 	return fmt.Errorf("failed to execute transformation %s: %w", item.Trans, err)
			// }
		} else if *item.As != "" {
			util.ModRunCmd.CopyDirContentOrFileTo(downloadCachePath, path.Dir(*item.As))
			src := path.Join(path.Dir(*item.As), path.Base(downloadCachePath))
			dest := *item.As
			err := os.Rename(src, dest)
			if err != nil {
				return fmt.Errorf("failed to rename %s to %s: %w", src, dest, err)
			}
		}
	}
	return nil
}
