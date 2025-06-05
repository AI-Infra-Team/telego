package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
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
	for idx, item := range deployment.Prepare {
		fmt.Println()
		if *item.Image != "" {
			// prepare image
			fmt.Printf("Preparing image: %s\n", *item.Image)
			err := ModJobImgPrepare.PrepareImages([]string{*item.Image})
			if err != nil {
				return fmt.Errorf("failed to prepare image %s: %w", *item.Image, err)
			}
		} else if item.Pyscript != nil && *item.Pyscript != "" {
			err := DeploymentPreparePyscript(filepath.Join(ConfigLoad().ProjectDir, project), &item)
			if err != nil {
				return fmt.Errorf("failed to prepare pyscript %s: %w", *item.Pyscript, err)
			}
		} else if *item.URL != "" {
			err := DeploymentPrepareUrl(filepath.Join(ConfigLoad().ProjectDir, project), &item)
			if err != nil {
				return fmt.Errorf("failed to prepare url %s: %w", *item.URL, err)
			}
		} else if item.FileMap != nil {
			err := item.FileMap.WriteToFile()
			if err != nil {
				fmt.Println(color.RedString("filemap write failed %v", err))
			}
		} else if *item.Git != "" {
			err := DeploymentPrepareGit(filepath.Join(ConfigLoad().ProjectDir, project), &item)
			if err != nil {
				return fmt.Errorf("failed to prepare git %s: %w", *item.Git, err)
			}
		} else {
			fmt.Println(color.YellowString("invalid prepare item found at idx:%d", idx))
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
	prjcachedir := filepath.Join(prjdir, PrepareCacheDir)
	// mkdir
	err := os.MkdirAll(prjcachedir, 0755)
	if err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// remove old pyscript
	prepare_script_prefix := "prepare_script_"
	{
		// Use Go's built-in file operations for cross-platform compatibility
		pattern := filepath.Join(prjcachedir, prepare_script_prefix+"*")
		matches, err := filepath.Glob(pattern)
		if err != nil {
			return fmt.Errorf("failed to glob pattern %s: %w", pattern, err)
		}
		for _, match := range matches {
			err := os.RemoveAll(match)
			if err != nil {
				return fmt.Errorf("failed to remove %s: %w", match, err)
			}
		}
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
	err = DeploymentPrepareHandleCached(item, prjdir, prjcachedir, "prepare_py_script has no src file")
	if err != nil {
		return fmt.Errorf("failed to handle cached: %w", err)
	}
	os.RemoveAll(temp.Name())
	return nil
}

func DeploymentPrepareUrl(prjdir string, item *DeploymentPrepareItem) error {
	util.PrintStep("prepare url", fmt.Sprintf("Downloading file from URL: %s", *item.URL))

	// download to download_cache
	// downloadCachePath := filepath.Join(PrepareCacheDir, filepath.Base(*item.URL))
	prjcachedir := filepath.Join(prjdir, PrepareCacheDir)
	err := os.MkdirAll(prjcachedir, 0755)
	if err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}
	downloadCachePath := filepath.Join(prjcachedir, filepath.Base(*item.URL))
	if _, err := os.Stat(downloadCachePath); err != nil {
		err := util.DownloadFile(*item.URL, downloadCachePath)
		if err != nil {
			// fmt.Println(color.RedString("Failed to download file from %s\n   err: %v", *item.URL, err))
			return fmt.Errorf("failed to download file from %s: %w", *item.URL, err)
		}
	}
	return DeploymentPrepareHandleCached(item, prjdir, prjcachedir, filepath.Base(*item.URL))
}

func DeploymentPrepareGit(prjdir string, item *DeploymentPrepareItem) error {
	// maybe split with :
	heads := []string{"http://", "https://", "git://", "git@"}
	giturl := *item.Git
	giturlWithoutHead := *item.Git
	for _, head := range heads {
		// replace head with ""
		giturlWithoutHead = strings.Replace(giturlWithoutHead, head, "", 1)
	}
	// maybe split branch with :
	branch := ""
	if strings.Contains(giturlWithoutHead, ":") {
		// giturl = strings.Split(giturlWithoutHead, ":")[0]
		branch = strings.Split(giturlWithoutHead, ":")[1]
		giturl = strings.ReplaceAll(*item.Git, ":"+branch, "")
	}

	util.PrintStep("prepare git", fmt.Sprintf("Downloading git prj from URL: %s, branch: %s", giturl, branch))
	cloneAtDir := filepath.Join(prjdir, PrepareCacheDir)
	// mk dir all
	err := os.MkdirAll(cloneAtDir, 0755)
	if err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	cloneTargetName := filepath.Base(giturl)
	// remove end .git
	cloneTargetName = strings.TrimSuffix(cloneTargetName, ".git")

	// check exsit
	if _, err := os.Stat(filepath.Join(cloneAtDir, cloneTargetName)); err != nil {
		// clone
		_, err = util.ModRunCmd.NewBuilder("git", "clone", giturl, cloneTargetName).
			SetDir(cloneAtDir).ShowProgress().BlockRun()
		if err != nil {
			return fmt.Errorf("failed to clone git prj, giturl: %s, err: %w", giturl, err)
		}
	}

	// checkout branch
	if branch != "" {
		_, err = util.ModRunCmd.NewBuilder("git", "checkout", branch).
			SetDir(filepath.Join(cloneAtDir, cloneTargetName)).ShowProgress().BlockRun()
		if err != nil {
			return fmt.Errorf("failed to checkout branch: %w", err)
		}
	}

	// after clone, archive to prjdir/teledeploy/git_prj_name.tar.gz
	tarPath := filepath.Join(prjdir, "teledeploy", fmt.Sprintf("%s.tar.gz", cloneTargetName))
	// remove old tar
	os.RemoveAll(tarPath)
	err = archiver.Archive([]string{filepath.Join(cloneAtDir, cloneTargetName)}, tarPath)
	if err != nil {
		return fmt.Errorf("failed to archive git repo: %w", err)
	}

	util.PrintStep("prepare git", fmt.Sprintf("Archive git repo to %s, fetch it by fileserver/%s/%s.tar.gz",
		tarPath, filepath.Base(prjdir), cloneTargetName))

	return nil
}

func DeploymentPrepareHandleCached(item *DeploymentPrepareItem, prjdir string, downloadCacheDir string, srcFileName string) error {
	_, err := os.Stat(downloadCacheDir)
	if err == nil {
		// Execute "trans" command if provided
		if len(item.Trans) > 0 {
			defer os.RemoveAll("extract")
			extractAppeared := false
			for _, step := range item.Trans {
				switch step := step.(type) {
				case DeploymentTransformExtract:
					fmt.Println(color.BlueString("Extracting file: %s", srcFileName))
					toExtract := filepath.Join(downloadCacheDir, srcFileName)
					extractAppeared = true
					os.RemoveAll("extract")
					fmt.Println(color.BlueString("Extracting file: %s", toExtract))
					err := os.MkdirAll("extract", 0755)
					if err != nil {
						return fmt.Errorf("failed to create directory: %w", err)
					}
					err = archiver.Unarchive(toExtract, "extract")
					if err != nil {
						return fmt.Errorf("failed to extract file: %w", err)
					}
				case DeploymentTransformCopy:
					for _, copyStep := range step.Copy {
						src := filepath.Join("extract", *copyStep.from)

						// no extect before copy, we just copy the file from downloadCacheDir
						if !extractAppeared {
							src = filepath.Join(downloadCacheDir, *copyStep.from)
						}

						dest := filepath.Join(prjdir, *copyStep.to)

						fmt.Println(color.BlueString("Copying %s to %s", src, dest))

						destDir := filepath.Dir(dest)
						err := os.MkdirAll(destDir, 0755)
						if err != nil {
							fmt.Println(color.RedString("failed to create directory: %w", err))
							return fmt.Errorf("failed to create directory: %w", err)
						}

						err = os.Rename(src, dest)
						if err != nil {
							fmt.Println(color.RedString("failed to rename %s to %s: %w", src, dest, err))
							return fmt.Errorf("failed to rename %s to %s: %w", src, dest, err)
						}

						fmt.Println(color.BlueString("Copied %s to %s", src, dest))
						// newSrc := filepath.Join(path.Dir(src), filepath.Base(dest))
						// newSrcDir := filepath.Dir(newSrc)
						// err := os.MkdirAll(newSrcDir, 0755)
						// if err != nil {
						// 	return fmt.Errorf("failed to create directory: %w", err)
						// }
						// err = os.Rename(src, newSrc)
						// if err != nil {
						// 	filetree, err := util.GetFileTree(10, "extract")
						// 	filetreeStr := ""
						// 	if err != nil {
						// 		filetreeStr = fmt.Sprintf("get filetree failed %v", err)
						// 	} else {
						// 		filetreeStr = filetree.GetDebugStr()
						// 	}
						// 	return fmt.Errorf("failed to rename %s to %s: %w, filetree in extract: \n%s", src, dest, err, filetreeStr)
						// }

						// util.ModRunCmd.CopyDirContentOrFileTo(newSrc, path.Dir(dest))
					}
				}

			}

			// fmt.Println(color.BlueString("Executing user defined transformation: %s at %s\n", item.Trans, CurDir()))
			// err := ModRunCmd.RunCommandShowProgress(item.Trans)
			// if err != nil {
			// 	return fmt.Errorf("failed to execute transformation %s: %w", item.Trans, err)
			// }
		} else if *item.As != "" {
			copySrc := filepath.Join(downloadCacheDir, srcFileName)
			as := filepath.Join(prjdir, *item.As)
			asDir := filepath.Dir(as)
			err := os.MkdirAll(asDir, 0755)
			if err != nil {
				return fmt.Errorf("failed to create directory: %w", err)
			}
			fmt.Println(color.BlueString("Copying %s to %s", copySrc, as))
			err = os.Rename(copySrc, as)
			if err != nil {
				return fmt.Errorf("failed to rename %s to %s: %w", copySrc, as, err)
			}
			// util.ModRunCmd.CopyDirContentOrFileTo(copySrc, path.Dir(*item.As))
			// src := filepath.Join(path.Dir(*item.As), path.Base(copySrc))
			// dest := *item.As
			// err := os.Rename(src, dest)
			// if err != nil {
			// 	return fmt.Errorf("failed to rename %s to %s: %w", src, dest, err)
			// }
		}
	}
	return nil
}
