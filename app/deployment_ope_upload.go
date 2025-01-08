package app

import (
	"fmt"
	"os"
	"path"
	"strings"
	"telego/util"
	"telego/util/yamlext"

	"github.com/fatih/color"
	"github.com/thoas/go-funk"
)

func DeploymentUpload(project string, deploymentConf *Deployment) error {
	if deploymentConf == nil {
		util.Logger.Warnf("deploymentConf is nil")
		return fmt.Errorf("deploymentConf is nil")
	}
	curDir0 := util.CurDir()
	defer os.Chdir(curDir0)

	// cd into projectDir
	projectDir := path.Join(ConfigLoad().ProjectDir, project)
	fmt.Println(color.BlueString("Uploading project %s contents in %s", project, projectDir))
	os.Chdir(projectDir)

	DeploymentOpePretreatment(project, deploymentConf)

	needUploadPublic := false
	needUploadSecret := false
	needUploadImages := false
	needUploadYaml := false

	if strings.HasPrefix(project, "bin_") {
		needUploadPublic = true
		needUploadSecret = true
		needUploadYaml = true
	} else if strings.HasPrefix(project, "k8s_") {
		needUploadPublic = true
		needUploadSecret = true
		needUploadImages = true
		needUploadYaml = true
	} else if strings.HasPrefix(project, "dist_") {
		needUploadPublic = true
		needUploadSecret = true
		needUploadImages = true
	} else {
		util.Logger.Warnf("unknown project type %s", project)
		return fmt.Errorf("unknown project type %s", project)
	}

	if needUploadPublic {
		// teledeploy mapped to /teledeploy on main node, which is a public file server on port 8003
		util.PrintStep("DeploymentUpload", "uploading public content (teledeploy) to main_node.fileserver")
		_, err := os.Stat("teledeploy")
		if err != nil {
			fmt.Println(color.YellowString("teledeploy is not exist, \n  please make sure if there are things to upload to public fileserver and prepare is executed"))
		} else {
			util.UploadToMainNode("teledeploy", path.Join("/teledeploy", project))
		}
	}

	if needUploadSecret {
		// teledeploy_secret mapped to /teledeploy_secret on main node, which is a private position, only can be accessed by rclone
		util.PrintStep("DeploymentUpload", "uploading secret content (teledeploy_secret) to main_node")
		_, err := os.Stat("teledeploy_secret")
		if err != nil {
			fmt.Println(color.YellowString("teledeploy_secret is not exist, \n  please make sure if there are things to upload to secret fileserver and prepare is executed"))
		} else {
			util.UploadToMainNode("teledeploy_secret", path.Join("/teledeploy_secret", project))
		}
	}

	if needUploadYaml {
		util.PrintStep("DeploymentUpload", "uploading yaml content (deployment.yml) to main_node")
		// upload k8s things
		// - deployment.yml
		util.UploadToMainNode("deployment.yml", path.Join("/teledeploy", project))

	}

	if needUploadImages {
		imageNames := funk.Map(
			funk.Filter(deploymentConf.Prepare, func(item DeploymentPrepareItem) bool {
				return item.Image != nil && *item.Image != ""
			}).([]DeploymentPrepareItem),
			func(item DeploymentPrepareItem) string {
				// fetch image name between last / and :
				splitTail := strings.Split(*item.Image, "/")
				return strings.ReplaceAll(splitTail[len(splitTail)-1], ":", "_")
			},
		).([]string)
		_, err := util.MainNodeConfReader{}.ReadPubConf(util.PubConfTypeImgUploaderUrl{})
		if err == nil {
			util.PrintStep("ImgUploader", "upload images by img uploader")
			// is img uploader enabled
			for _, img := range imageNames {
				fmt.Println(color.BlueString("Uploading img %s", img))
				cmds := ModJobImgUploader.NewCmd(ImgUploaderModeClient{
					ImagePath: path.Join(ConfigLoad().ProjectDir, "container_image", fmt.Sprintf("image_%v", img))})
				_, err := util.ModRunCmd.ShowProgress(cmds[0], cmds[1:]...).ShowProgress().BlockRun()
				if err != nil {
					fmt.Println(color.RedString("ImgUploader upload failed: %s", err))
					os.Exit(1)
				}
				fmt.Println(color.GreenString("ImgUploader upload success: %s", img))
				// ModJobImgUploader.ImgUploaderLocal(ImgUploaderModeClient{
				// 	ImagePath: path.Join(ConfigLoad().ProjectDir, "container_image", fmt.Sprintf("image_%v", img))})
			}
		} else {
			util.PrintStep("ImgUploader", "upload images by rclone with admin permission")
			if len(imageNames) > 0 {
				for _, imageName := range imageNames {
					localPath := path.Join(ConfigLoad().ProjectDir, "container_image", fmt.Sprintf("image_%s", imageName))
					remotePath := fmt.Sprintf("/teledeploy_secret/container_image/image_%s", imageName)
					fmt.Println(color.BlueString("Uploading img %s to %s", localPath, remotePath))
					util.UploadToMainNode(localPath, remotePath)
				}
				imageNamesWithQuotes := []string{}
				for _, imageName := range imageNames {
					imageNamesWithQuotes = append(imageNamesWithQuotes, fmt.Sprintf("\"%s\"", imageName))
				}

				imgRepoYaml, err := util.MainNodeConfReader{}.ReadSecretConf(util.SecretConfTypeImgRepo{})
				if err != nil {
					fmt.Println(color.RedString("ImgUploader get img repo secret failed: %s", err))
					os.Exit(1)
				}

				imgRepo := util.ContainerRegistryConf{}
				err = yamlext.UnmarshalAndValidate([]byte(imgRepoYaml), &imgRepo)
				if err != nil {
					fmt.Println(color.RedString("ImgUploader get img repo secret failed: %s", err))
					os.Exit(1)
				}

				uploadScript := fmt.Sprintf(`
import os, subprocess

images=[%s]
REPO_HOST="%s"
REPO_NAMESPACE="teleinfra"
REPO_USER="%s"
REPOPW=f"%s"

os.chdir("/teledeploy_secret/container_image")
def run_command(command,allow_fail=False):
	"""
	Run a shell command and ensure it completes successfully.
	"""
	result = subprocess.run(command, shell=True, stdout=subprocess.PIPE, stderr=subprocess.PIPE)
	if result.returncode != 0:
		print(f"Error running command: {command}")
		print(result.stderr)
		if not allow_fail:
			raise Exception(f"Command failed: {command}")
	else:
		print(result.stdout)
		return result.stdout.decode("utf-8")

sudo_prefix=""
if os.getuid() != 0:
	sudo_prefix="sudo "

run_command(f"{sudo_prefix}docker login -u {REPO_USER} -p {REPOPW} {REPO_HOST}")
for image in images:
	os.chdir(f"image_{image}")
	list_dir=os.listdir()
	arm64=[f for f in list_dir if f.find("_arm64_")!=-1]
	amd64=[f for f in list_dir if f.find("_amd64_")!=-1]

	def upload_arch(arch,arch_tar_pkg):
		img_name=run_command(f"{sudo_prefix}docker load -i {arch_tar_pkg}").split("Loaded image: ")[-1].strip()
		img_key_no_prefix=img_name.split("/")[-1]
		id=run_command(f"{sudo_prefix}docker image list -q {img_name}").strip()
		run_command(f"{sudo_prefix}docker tag {id} {REPO_HOST}/{REPO_NAMESPACE}/{img_key_no_prefix}_{arch}")
		run_command(f"{sudo_prefix}docker push {REPO_HOST}/{REPO_NAMESPACE}/{img_key_no_prefix}_{arch} -q")
		return img_key_no_prefix
	img_key_no_prefix=""
	mani_create_arch=""
	if len(arm64)>0:
		img_key_no_prefix=upload_arch("arm64",arm64[0])
		mani_create_arch+=f"{REPO_HOST}/{REPO_NAMESPACE}/{img_key_no_prefix}_amd64 "
	if len(amd64)>0:
		img_key_no_prefix=upload_arch("amd64",amd64[0])
		mani_create_arch+=f"{REPO_HOST}/{REPO_NAMESPACE}/{img_key_no_prefix}_arm64 "
	run_command(f"{sudo_prefix}docker manifest create {REPO_HOST}/{REPO_NAMESPACE}/{img_key_no_prefix} {mani_create_arch} --insecure", allow_fail=True)
	if len(arm64)>0:
		run_command(f"{sudo_prefix}docker manifest annotate {REPO_HOST}/{REPO_NAMESPACE}/{img_key_no_prefix} "
			f"--arch arm64 {REPO_HOST}/{REPO_NAMESPACE}/{img_key_no_prefix}_arm64")
	if len(amd64)>0:
		run_command(f"{sudo_prefix}docker manifest annotate {REPO_HOST}/{REPO_NAMESPACE}/{img_key_no_prefix} "
			f"--arch amd64 {REPO_HOST}/{REPO_NAMESPACE}/{img_key_no_prefix}_amd64")
	run_command(f"{sudo_prefix}docker manifest push {REPO_HOST}/{REPO_NAMESPACE}/{img_key_no_prefix} --insecure")
	os.chdir("..")
		`, strings.Join(imageNamesWithQuotes, ","), util.ImgRepoAddressNoPrefix, imgRepo.User, imgRepo.Password)

				util.StartRemoteCmds([]string{
					fmt.Sprintf("%s@%s", util.MainNodeUser, util.MainNodeIp),
				}, "sudo "+util.EncodeRemoteRunPy(uploadScript), "")
			}
		}

	}

	return nil
}
