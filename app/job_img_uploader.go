package app

import (
	"bufio"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"telego/util"
	"telego/util/strext"
	"telego/util/yamlext"

	"github.com/barweiss/go-tuple"
	"github.com/fatih/color"
	"github.com/gin-gonic/gin"
	"github.com/mitchellh/mapstructure"
	"github.com/spf13/cobra"
	"github.com/thoas/go-funk"
)

type ImgUploaderJob struct {
	WorkDir   string
	ImagePath string
}

type remoteStoreUser struct {
	UserName string
	Password string
	Dir      string
}

type ModJobImgUploaderStruct struct {
	imageLocks         sync.Map
	remoteStoreUserMap sync.Map
}

var ModJobImgUploader *ModJobImgUploaderStruct = &ModJobImgUploaderStruct{}

func (m *ModJobImgUploaderStruct) JobCmdName() string {
	return "img-uploader"
}

func (m *ModJobImgUploaderStruct) uploadImageV2() {
	err := NewBinManager(BinManagerWinfsp{}).MakeSureWith()
	if err != nil {
		fmt.Println(color.RedString("winfsp is not installed: %v", err))
		os.Exit(1)
	}

	// get image uploader url
	util.PrintStep("ImgUploader", "fetching image uploader url")
	img_upload_server := func() string {
		img_upload_server, err := util.MainNodeConfReader{}.ReadPubConf(util.PubConfTypeImgUploaderUrl{})
		if err != nil {
			fmt.Println(color.RedString("Failed to read image uploader url: %v", err))
			os.Exit(1)
		}
		img_upload_server = strings.TrimSpace(img_upload_server)
		return img_upload_server
	}()

	util.PrintStep("ImgUploader", "creating new template store")
	tempStoreInfo, tempStorePw := func() (ImgUploadNewRemoteStoreResponse, string) {
		tempStoreInfo := ImgUploadNewRemoteStoreResponse{}
		// 创建用户
		resJson, err := util.HttpOneshot(util.UrlJoin(img_upload_server, "/uploadv2"), nil)
		if err != nil {
			fmt.Println(color.RedString("Failed to new temp store: %v", err))
			os.Exit(1)
		}
		err = json.Unmarshal(resJson, &tempStoreInfo)
		if err != nil {
			fmt.Println(color.RedString("Failed to new temp store: %v", err))
			os.Exit(1)
		}
		tempStorePw_, err := (base64.StdEncoding.DecodeString(tempStoreInfo.Pwb64))
		if err != nil {
			fmt.Println(color.RedString("Failed to new temp store: %v", err))
			os.Exit(1)
		}
		tempStorePw := string(tempStorePw_)
		return tempStoreInfo, tempStorePw
	}()

	mountPath := "D:/img-uploader-temp"
	os.RemoveAll(mountPath)
	err = os.MkdirAll(filepath.Join(mountPath, "使用传输工具在此新建任意文件夹后，等待刷新和下一步提示"), 0755)
	if err != nil {
		fmt.Println(color.RedString("Failed to create dir: %v", err))
		os.Exit(1)
	}

	util.PrintStep("ImgUploader", fmt.Sprintf("请在文件传输助手中进入临时目录 %s, 并新建任意文件夹", mountPath))
	for {
		// if dir list >0 then break
		list, err := os.ReadDir(mountPath)
		if err != nil {
			fmt.Println(color.RedString("read dir failed %s", err))
			os.Exit(1)
		}
		if len(list) > 1 {
			break
		}
	}

	util.PrintStep("ImgUploader", "请不要切换传输助手路径，远程目录挂载中")
	for {
		err = os.RemoveAll(mountPath)
		if err != nil {
			fmt.Println(color.RedString("Failed to remove dir: %v", err))
			time.Sleep(200 * time.Millisecond)
		} else {
			break
		}
	}

	mountPath, mountHandle := func() (string, *util.CmdBuilder) {
		if util.IsWindows() {
			// use rclone to mount remote
			rcloneTempRemoteName := "img-uploader-temp-" + tempStoreInfo.User
			err := util.NewRcloneConfiger(util.RcloneConfigTypeSsh{}, rcloneTempRemoteName, tempStoreInfo.ServerHost).
				WithUser(tempStoreInfo.User, tempStorePw).DoConfig()
			if err != nil {
				fmt.Println(color.RedString("Failed to config rclone: %v", err))
			}

			cmds := ModJobRclone.mountCmd(&rcloneMountArgv{
				remotePath: rcloneTempRemoteName + ":/",
				localPath:  mountPath,
			})
			// .NewCmd(JobRcloneTypeMount{
			// 	rcloneMountArgv: rcloneMountArgv{
			// 		remotePath: rcloneTempRemoteName + ":/",
			// 		localPath:  mountPath,
			// 	},
			// })
			mountHandle := util.ModRunCmd.NewBuilder(cmds[0], cmds[1:]...)
			_, err = mountHandle.AsyncRun()
			if err != nil {
				fmt.Println(color.RedString("Failed to mount rclone: %v", err))
			}
			return mountPath, mountHandle
		} else {
			fmt.Println(color.RedString("挂载式镜像上传暂未支持 linux"))
			os.Exit(1)
			return "", nil
		}
	}()
	defer func() {
		if mountHandle != nil {
			mountHandle.Cmd.Process.Kill()
		}
	}()
	for {
		time.Sleep(1 * time.Second)
		_, err := os.Stat(mountPath)
		if err == nil {
			break
		}
	}

	util.PrintStep("ImgUploader", "已挂载到 "+mountPath+", 请用文件传输助手上传镜像文件到对应目录，并在终端输入 y 后回车")
	tempfile := "现在开始往当前文件夹上传镜像，上传完后回到终端确认"
	file, err := os.Create(filepath.Join(mountPath, tempfile))
	if err != nil {
		fmt.Println(color.RedString("创建临时文件失败"))
		os.Exit(1)
	}
	file.Close()

	imglist := []string{}
	remotelist := []string{}
	{
		reader := bufio.NewReader(os.Stdin)
		for {
			fmt.Print("直接回车校验列表，输入 y 回车开始上传: ")
			input, err := reader.ReadString('\n')
			if err != nil {
				fmt.Println("\n输入读取失败，请重试")
				continue
			}

			// 去除多余的空格和换行符
			input = strings.TrimSpace(input)

			if len(imglist) > 0 && funk.Equal(imglist, remotelist) && input == "y" {
				fmt.Println("开始上传...")
				break
			} else {
				fmt.Println("开始校验列表...")
				imglist = func() []string {
					// list dir
					list, err := os.ReadDir(mountPath)
					if err != nil {
						fmt.Println(color.RedString("读取目录失败，请重试，或寻找管理员寻求帮助, err: %s", err))
						os.Exit(1)
					}

					tarfiles := []string{}
					for _, file := range list {
						if file.IsDir() {
							continue
						}
						if !strings.HasSuffix(file.Name(), ".tar") {
							continue
						}
						tarfiles = append(tarfiles, file.Name())
					}
					return tarfiles
				}()

				resJson, err := util.HttpOneshot(util.UrlJoin(img_upload_server, "/uploadv2"), ImgUploadUploadedReq{
					Username:    tempStoreInfo.User,
					PasswordB64: tempStoreInfo.Pwb64,
					CheckDir:    true,
				})
				if err != nil {
					fmt.Println(color.RedString("请求远程镜像列表失败, err: %s", err))
					return
				}
				res := map[string]interface{}{}
				if err := json.Unmarshal(resJson, &res); err != nil {
					fmt.Println(color.RedString("解析返回列表 json: %v, err: %s", string(resJson), err))
					return
				}
				var ok bool
				remotelist_, ok := res["success"]
				if !ok {
					fmt.Println(color.RedString("解析返回列表 json: %v, map: %v, err: %v", string(resJson), res, ok))
					return
				}
				err = mapstructure.Decode(remotelist_, &remotelist)
				if err != nil {
					fmt.Println(color.RedString("解析lisst to string err %v", err))
					return
				}
				remotelist = funk.Map(remotelist, func(s string) string {
					return filepath.Base(s)
				}).([]string)

				sort.Strings(remotelist)
				sort.Strings(imglist)
				if funk.Equal(remotelist, imglist) {
					fmt.Println(color.GreenString("镜像列表已同步"))
					for _, v := range remotelist {
						fmt.Println(color.GreenString("- %s", v))
					}
					fmt.Println()
				} else {
					fmt.Println(color.YellowString("列表未完全同步，请稍后回车校验"))
				}
			}
		}
	}

	util.PrintStep("ImgUploader", fmt.Sprintf("开始上传镜像 %v", imglist))
	resJson, err := util.HttpOneshot(util.UrlJoin(img_upload_server, "/uploadv2"), ImgUploadUploadedReq{
		Username:    tempStoreInfo.User,
		PasswordB64: tempStoreInfo.Pwb64,
	})
	if err != nil {
		fmt.Println(color.RedString("上传失败，请重试，或寻找管理员寻求帮助, err: %v", err))
		return
	}
	fmt.Println(color.GreenString("上传成功, result: %s", resJson))
}

func (m *ModJobImgUploaderStruct) uploadImage(imagePath string) {
	util.PrintStep("ImgUploader", fmt.Sprintf("开始上传 %v 目录下的镜像", imagePath))
	if !filepath.IsAbs(imagePath) {
		imagePath = filepath.Join(util.GetEntryDir(), imagePath)
	}

	// scan files under imagePath
	files, err := filepath.Glob(filepath.Join(imagePath, "*.tar"))
	if err != nil {
		fmt.Println(color.RedString("Failed to scan files under %s: %v", imagePath, err))
		os.Exit(1)
	}

	// get image uploader url
	img_upload_server, err := util.MainNodeConfReader{}.ReadPubConf(util.PubConfTypeImgUploaderUrl{})
	if err != nil {
		fmt.Println(color.RedString("Failed to read image uploader url: %v", err))
		os.Exit(1)
	}
	img_upload_server = util.UrlJoin(strings.TrimSpace(img_upload_server), "/upload")

	util.PrintStep("ImgUploader", fmt.Sprintf("开始上传镜像 %v", files))
	res, err := util.UploadMultipleFilesInOneConnection(files, img_upload_server)
	if err != nil {
		fmt.Println(color.RedString("Failed to upload files: %v", err))
		os.Exit(1)
	} else {
		fmt.Println(color.GreenString("Uploaded files successfully:"))
		fmt.Println(res)
	}
}

func (m *ModJobImgUploaderStruct) startServer(workdir string) {
	util.PrintStep("ImgUploader", "starting server... at "+workdir)

	_, err := os.Stat(workdir)
	if err != nil {
		if !os.IsNotExist(err) {
			fmt.Println(color.RedString("workdir %s access error: %v", workdir, err))
			return
		}
	}

	// 文件上传处理函数

	r := gin.Default()
	r.POST("/upload", ImgUploaderUploadHandlerV1)
	r.POST("/uploadv2", func(c *gin.Context) {
		imgUploaderUploadHandlerV2(c, workdir)
	})
	// http.HandleFunc("/upload", uploadHandler) // 注册文件上传的处理函数

	// 启动 HTTP 服务器
	port := "8088"
	fmt.Printf("Starting server on :%s\n", port)
	r.Run(":" + port)
	// if err := http.ListenAndServe(":"+port, nil); err != nil {
	// 	fmt.Printf("Error starting server: %s\n", err)
	// }
}

func (m *ModJobImgUploaderStruct) ParseJob(ImgUploaderCmd *cobra.Command) *cobra.Command {
	job := &ImgUploaderJob{}

	// 绑定命令行标志到结构体字段
	ImgUploaderCmd.Flags().StringVar(&job.WorkDir, "workdir", "", "workdir for image uploader server")
	ImgUploaderCmd.Flags().StringVar(&job.ImagePath, "image", "", "image path for image uploader client")
	ImgUploaderCmd.Flags()
	ImgUploaderCmd.Run = func(_ *cobra.Command, _ []string) {
		if job.WorkDir == "" && job.ImagePath == "" {
			m.uploadImageV2()
		} else if job.WorkDir != "" {
			m.startServer(job.WorkDir)
		} else if job.ImagePath != "" {
			m.uploadImage(job.ImagePath)
		}
	}
	// err := ImgUploaderCmd.Execute()
	// if err != nil {
	// 	return nil,nil
	// }

	return ImgUploaderCmd
}

type ImgUploaderMode interface {
	ImgUploaderModeDummyInterface()
}

type ImgUploaderModeServer struct {
	WorkDir string
}

type ImgUploaderModeClient struct {
	ImagePath string
}

func (m ImgUploaderModeServer) ImgUploaderModeDummyInterface() {}

func (m ImgUploaderModeClient) ImgUploaderModeDummyInterface() {}

func (m *ModJobImgUploaderStruct) NewCmd(
	mode ImgUploaderMode,
) []string {
	switch mode.(type) {
	case ImgUploaderModeServer:
		mode := mode.(ImgUploaderModeServer)
		return []string{"telego", "img-uploader", "--workdir", mode.WorkDir}
	case ImgUploaderModeClient:
		mode := mode.(ImgUploaderModeClient)
		return []string{"telego", "img-uploader", "--image", mode.ImagePath}
	default:
		panic(color.RedString("unknown mode %v", mode))
	}
}

// // in app entry
// func (m *ModJobImgUploaderStruct) ImgUploaderLocal(mode ImgUploaderMode) {
// 	// cmds := []string{}
// 	cmds := ModJobImgUploader.NewCmd(mode)
// 	util.ModRunCmd.ShowProgress(cmds[0], cmds[1:]...).BlockRun()
// }

type ImgUploadUploadedReq struct {
	Username    string `json:"user"`
	PasswordB64 string `json:"pwb64"`
	CheckDir    bool   `json:"check_dir,omitempty"`
}

type ImgUploadNewRemoteStoreResponse struct {
	ServerHost string `json:"server_host"` // with ip and port
	User       string `json:"user"`
	Pwb64      string `json:"pwb64"`
}

func imgUploaderUploadHandlerV2(c *gin.Context, workdir string) {
	handler := (&ImgUploaderUploadHandlerV2{})

	// 尝试解析为 User 类型
	uploaded := ImgUploadUploadedReq{}
	if err := c.ShouldBindJSON(&uploaded); err == nil {
		pw_, err := base64.StdEncoding.DecodeString(uploaded.PasswordB64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid password"})
			return
		}
		pw := string(pw_)
		imgdir, err := handler.findPrevTempUsr(uploaded.Username, pw)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("prev temp user not found: %v", err)})
			return
		}

		if uploaded.CheckDir {
			tarlist_, err := os.ReadDir(filepath.Join(workdir, imgdir))
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("read dir failed: %v", err)})
				return
			}
			tarlist := make([]string, 0)
			for _, tar := range tarlist_ {
				if tar.IsDir() {
					continue
				}
				if !strings.HasSuffix(tar.Name(), ".tar") {
					continue
				}
				tarlist = append(tarlist, tar.Name())
			}
			c.JSON(http.StatusOK, gin.H{"success": tarlist})
			return
		}

		succress, err := handler.checkAndUploadImgs(filepath.Join(workdir, imgdir))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("check and upload imgs failed: %v", err)})
			return
		}
		c.JSON(http.StatusOK, gin.H{"success": succress})
	} else {
		printtag := "ImgUploaderCreatingTemp"
		registryConfYaml, err := util.MainNodeConfReader{}.ReadSecretConf(util.SecretConfTypeImgRepo{})
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("read img repo conf failed: %v", err)})
			return
		}
		registryConf := util.ContainerRegistryConf{}
		err = yamlext.UnmarshalAndValidate([]byte(registryConfYaml), &registryConf)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("unmarshal img repo conf failed: %v", err)})
			return
		}

		user_pw_dir, err := handler.generateTempUser()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("generate temp user failed: %v", err)})
			return
		}

		util.PrintStep(printtag, fmt.Sprintf("creating temp user_pw_dir: %v", user_pw_dir))

		if !func() bool {
			// return if empty or not exist
			_, err := os.Stat(filepath.Join(workdir, user_pw_dir.V3))
			if err != nil {
				// not exist
				return true
			}
			// listdir
			files, err := os.ReadDir(filepath.Join(workdir, user_pw_dir.V3))
			if err != nil {
				return false
			}
			if len(files) == 0 {
				// empty
				return true
			}

			return false
		}() {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "temp dir not empty"})
			return
		}

		err = os.MkdirAll(filepath.Join(workdir, user_pw_dir.V3), 0755)
		if err != nil {
			err = fmt.Errorf("failed to create temp dir: %w", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to create temp dir: %v", err)})
			return
		}

		err = util.ModSftpgo.CreateTempSpace(
			registryConf.UploaderStoreAddr,
			registryConf.UploaderStoreAdmin,
			registryConf.UploaderStoreAdminPw, user_pw_dir.V1, user_pw_dir.V2, user_pw_dir.V3)
		if err != nil {
			// return fmt.Errorf("failed to create temp remote store: %w", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to create temp remote store: %v", err)})
			return
		}

		// record temp user
		handler.record(user_pw_dir.V1, user_pw_dir.V2, user_pw_dir.V3)

		c.JSON(http.StatusOK, ImgUploadNewRemoteStoreResponse{
			ServerHost: registryConf.UploaderStoreTransferAddr,
			User:       user_pw_dir.V1,
			Pwb64:      base64.StdEncoding.EncodeToString([]byte(user_pw_dir.V2)),
		})
	}
}

type RemoteStoreConfig struct {
	User     string `json:"user"`
	PwBase64 string `json:"pwb64"`
	Host     string `json:"host"`
	Type     string `json:"type"` // sftp
}

func (m *ImgUploaderUploadHandlerV2) findPrevTempUsr(username, password string) (string, error) {
	res, ok := ModJobImgUploader.remoteStoreUserMap.Load(username + ":" + password)
	if !ok {
		return "", errors.New("prev temp user not found")
	}
	return res.(string), nil
}

// imageDir is absolute path
func (m *ImgUploaderUploadHandlerV2) checkAndUploadImgs(imageDir string) ([]string, error) {
	getTarFiles := func(dir string) ([]string, error) {
		var tarFiles []string

		// 遍历目录及其子目录
		err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			// 检查是否为 .tar 文件
			if !info.IsDir() && filepath.Ext(path) == ".tar" {
				tarFiles = append(tarFiles, path)
			}
			return nil
		})

		return tarFiles, err
	}
	tarFiles, err := getTarFiles(imageDir)
	if err != nil {
		return nil, fmt.Errorf("get tar files failed %w", err)
	}

	// 镜像信息结构体
	type ImageInfo struct {
		imagename string // like xxx/xxx
		tag       string
		id        string
		arm       bool
		amd       bool
		file      string
	}

	imageBaseName := func(i ImageInfo) string {
		arr := strings.Split(i.imagename, "/")
		return arr[len(arr)-1]
	}

	imgInfoMap := make(map[string][]ImageInfo)
	processImage := func(tarPath string) error {
		// 1. 加载镜像
		// 加载镜像, 返回 imagename xxx/xxx 和 tag
		loadImage := func(tarPath string) (ImageInfo, error) {
			cmd := exec.Command("docker", "load", "-i", tarPath)
			cmd.CombinedOutput()
			cmd = exec.Command("docker", "load", "-i", tarPath)
			output, err := cmd.CombinedOutput()
			if err != nil {
				return ImageInfo{}, fmt.Errorf("failed to load image: %s", err)
			}
			// 解析镜像名称和标签
			outputStr := string(output)
			if strings.HasPrefix(outputStr, "Loaded image ID: ") {
				id := strings.TrimSpace(strings.Split(outputStr, ": ")[1])
				fmt.Printf("Loaded image from %s with id %s\n", tarPath, id)
				return ImageInfo{
					file: tarPath,
					id:   id,
				}, nil
			}

			if strings.HasPrefix(outputStr, "Loaded image: ") {
				nameTag := strings.TrimSpace(strings.Split(outputStr, ": ")[1])
				fmt.Printf("Loaded image from %s with tag %s\n", tarPath, nameTag)
				// 2. 获取id信息
				getImageID := func(imageName string) ([]string, error) {
					cmd := exec.Command("docker", "image", "list", "-q", imageName)
					output, err := cmd.CombinedOutput()
					if err != nil {
						return nil, fmt.Errorf("failed to list image ids: %s", err)
					}
					ids := strings.Fields(string(output))
					return ids, nil
				}
				ids, err := getImageID(nameTag)
				if err != nil {
					return ImageInfo{}, fmt.Errorf("image '%s' not found", nameTag)
				}
				if len(ids) == 0 {
					return ImageInfo{}, fmt.Errorf("image '%s' not found", nameTag)
				}

				return ImageInfo{
					file:      tarPath,
					imagename: strings.Split(nameTag, ":")[0],
					tag:       strings.Split(nameTag, ":")[1],
					id:        ids[0],
				}, nil
			}

			return ImageInfo{}, fmt.Errorf("unexpected output from docker load: %s", outputStr)
		}
		imageInfo, err := loadImage(tarPath)
		if err != nil {
			return fmt.Errorf("failed to load image: %v", err)
		}

		// 3. 获取架构信息
		loadImageInfoArch := func(info *ImageInfo) error {
			if info == nil {
				return fmt.Errorf("invalid ImageInfo: nil pointer")
			}

			cmd := exec.Command("docker", "inspect", info.id)
			output, err := cmd.CombinedOutput()
			if err != nil {
				return fmt.Errorf("failed to inspect image: %v", err)
			}

			// 解析 JSON 输出
			var inspectResult []map[string]interface{}
			if err := json.Unmarshal(output, &inspectResult); err != nil {
				return fmt.Errorf("failed to parse docker inspect output: %v", err)
			}

			if len(inspectResult) == 0 {
				return fmt.Errorf("no data found for image ID: %s", info.id)
			}

			// 提取架构信息
			arch, ok := inspectResult[0]["Architecture"].(string)
			if !ok {
				return fmt.Errorf("failed to find architecture info in image data")
			}
			fmt.Println("Img Architecture:", arch, " ,id:", info.id)

			// 更新 ImageInfo 结构体
			info.arm = strings.Contains(arch, "arm64")
			info.amd = strings.Contains(arch, "amd64")

			return nil
		}
		err = loadImageInfoArch(&imageInfo)
		if err != nil {
			return fmt.Errorf("failed to get image info: %v", err)
		}

		// 4. 补全镜像信息
		if imageInfo.imagename == "" {
			// use filename
			imageInfo.imagename = strings.Split(tarPath, ".tar")[0]
		}
		if imageInfo.tag == "" {
			imageInfo.tag = strext.SafeSubstring(imageInfo.id, 0, 5)
		}

		// 加入到收集map
		_, ok := imgInfoMap[imageInfo.imagename+":"+imageInfo.tag]
		if !ok {
			imgInfoMap[imageInfo.imagename+":"+imageInfo.tag] = []ImageInfo{}
		}
		imgInfoMap[imageInfo.imagename+":"+imageInfo.tag] = append(
			imgInfoMap[imageInfo.imagename+":"+imageInfo.tag],
			imageInfo,
		)
		return nil
	}
	for _, tarPath := range tarFiles {
		err := processImage(tarPath)
		if err != nil {
			return nil, fmt.Errorf("Error processing image: %v", err)
		}
	}

	if len(imgInfoMap) > 0 {
		registryConfYaml, err := util.MainNodeConfReader{}.ReadSecretConf(util.SecretConfTypeImgRepo{})
		if err != nil {
			return nil, fmt.Errorf("Error reading image repo config: %v", err)
		}
		registryConf := util.ContainerRegistryConf{}
		err = yamlext.UnmarshalAndValidate([]byte(registryConfYaml), &registryConf)
		if err != nil {
			return nil, fmt.Errorf("Error unmarshaling image repo config: %v", err)
		}

		util.ModDocker.SetUserPwd(registryConf.User, registryConf.Password)
	}

	wait := &sync.WaitGroup{}
	failress := make([]error, len(imgInfoMap))
	succress := make([]string, len(imgInfoMap))
	idx := 0
	for _, info := range imgInfoMap {
		wait.Add(1)
		info_ := info
		go func(idx_ int) {
			// 上传镜像
			uploadImage := func(imageInfos []ImageInfo) error {
				var amdInfo *ImageInfo
				var armInfo *ImageInfo
				func() {
					for _, imageInfo := range imageInfos {
						if imageInfo.amd && imageInfo.arm {
							amdInfo = &imageInfo
							armInfo = &imageInfo
						}
					}
					if amdInfo != nil && armInfo != nil {
						return
					}

					for _, imageInfo := range imageInfos {
						if imageInfo.amd {
							amdInfo = &imageInfo
						}
						if imageInfo.arm {
							armInfo = &imageInfo
						}
					}
				}()

				if amdInfo == nil && armInfo == nil {
					return fmt.Errorf("no amd or arm image found")
				}
				fmt.Printf("infos %v %v\n", amdInfo, armInfo)
				{
					tagWithArch := func(info ImageInfo, arch string) (string, error) {
						// 1. tag by info.id
						registryTag := util.UrlJoin(util.ImgRepoAddressNoPrefix(), "/teleinfra/"+imageBaseName(info)+":"+info.tag+"_"+arch)
						_, err := util.ModRunCmd.ShowProgress(
							"docker", "tag", info.id, registryTag,
						).WithRoot().BlockRun()
						if err != nil {
							return "", fmt.Errorf("docker tag failed %v", err)
						}
						return registryTag, nil
					}
					amdTag := ""
					if amdInfo != nil {
						amdTag, err = tagWithArch(*amdInfo, "amd64")
						if err != nil {
							return err
						}
					}
					armTag := ""
					if armInfo != nil {
						armTag, err = tagWithArch(*armInfo, "arm64")
						if err != nil {
							return err
						}
					}

					// 2. upload
					push := func(tag string) error {
						cmds, err := util.ModDocker.PushDockerImage(tag)
						for _, cmd := range cmds {
							outPut, err := cmd.BlockRun()
							if err != nil {
								return fmt.Errorf("docker upload %v failed %v,	Output:%s", cmd.Cmds(), err, outPut)
							}
						}
						return err
					}
					if amdTag != "" {
						err = push(amdTag)
						if err != nil {
							return err
						}
					}
					if armTag != "" {
						err = push(armTag)
						if err != nil {
							return err
						}
					}

					// 3. manifest, allow fail
					noArchTag := util.UrlJoin(util.ImgRepoAddressNoPrefix(), "/teleinfra/"+imageBaseName(*amdInfo)+":"+amdInfo.tag)
					fmt.Println("manifest noArchTag:", noArchTag)
					util.ModRunCmd.ShowProgress(
						"docker", "manifest", "create",
						noArchTag,
						amdTag, armTag, "--insecure", "--amend",
					).WithRoot().BlockRun()
					annotate := func(tag string) error {
						arch := "amd64"
						if strings.Contains(tag, "arm64") {
							arch = "arm64"
						}
						_, err := util.ModRunCmd.ShowProgress(
							"docker", "manifest", "annotate",
							noArchTag,
							"--arch", arch, tag,
						).WithRoot().BlockRun()
						if err != nil {
							return fmt.Errorf("docker manifest annotate failed %v", err)
						}
						return err
					}
					if armTag != "" {
						err := annotate(armTag)
						if err != nil {
							return err
						}
					}
					if amdTag != "" {
						err := annotate(amdTag)
						if err != nil {
							return err
						}
					}
					_, err := util.ModRunCmd.ShowProgress(
						"docker", "manifest", "push",
						noArchTag, "--insecure",
					).WithRoot().BlockRun()
					if err != nil {
						return fmt.Errorf("docker manifest push failed %v", err)
					}
				}
				res := ""
				if amdInfo != nil && armInfo != nil {
					res = fmt.Sprintf("Uploaded image %s:%s with arm in %s & amd in %s",
						armInfo.imagename, armInfo.tag, armInfo.file, amdInfo.file)
				} else if amdInfo != nil {
					res = fmt.Sprintf("Uploaded image %s:%s with amd in %s",
						amdInfo.imagename, amdInfo.tag, amdInfo.file)
				} else if armInfo != nil {
					res = fmt.Sprintf("Uploaded image %s:%s with arm in %s",
						armInfo.imagename, armInfo.tag, armInfo.file)
				}
				fmt.Println(color.GreenString(res))
				succress[idx_] = res
				return nil
			}
			err := uploadImage(info_)
			if err != nil {
				fmt.Println(color.RedString("ImgUploader upload image failed: %s", err))
			} else {
				fmt.Println(color.GreenString(succress[idx_]))
			}
			failress[idx_] = err

			defer wait.Done()
			fmt.Println("wait done")
		}(idx)
		idx += 1
	}
	wait.Wait()
	fmt.Println("wait receive done")

	// 返回成功响应
	if err := funk.Find(failress, func(err error) bool {
		return err != nil
	}); err != nil {
		fmt.Println(color.RedString("ImgUploader upload image failed: %v", err))
		// c.JSON(500, gin.H{"error": fmt.Sprintf("ImgUploader upload image failed: %v", err)})
		return nil, err.(error)
	} else {
		fmt.Println(color.GreenString("ImgUploader upload image success %v", succress))
		// c.JSON(200, gin.H{"ok": strings.Join(succress, "\n")})
		fmt.Println("/upload ok return")
		return succress, nil
	}
}

func (m *ImgUploaderUploadHandlerV2) generateTempUser() (tuple.T3[string, string, string], error) {

	template := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789_"
	// generate temp user,pw,dir string
	user, err := util.GenerateRandomStringFromInput(template, 8)
	if err != nil {
		return tuple.T3[string, string, string]{}, err
	}
	pw, err := util.GenerateRandomStringFromInput(template, 8)
	if err != nil {
		return tuple.T3[string, string, string]{}, err
	}
	dir, err := util.GenerateRandomStringFromInput(template, 8)
	if err != nil {
		return tuple.T3[string, string, string]{}, err
	}
	return tuple.New3(user, pw, dir), nil
}

type ImgUploaderUploadHandlerV2 struct{}

func (m *ImgUploaderUploadHandlerV2) record(user, pw, dir string) {
	ModJobImgUploader.remoteStoreUserMap.Store(user+":"+pw, dir)
}

func ImgUploaderUploadHandlerV1(c *gin.Context) {
	// 先返回响应头，表示开始处理
	// c.Header("X-Upload-Status", "Started")
	// c.JSON(http.StatusOK, gin.H{
	// 	"message": "Upload started",
	// })
	fmt.Println("11111")
	// 检查请求方法是否为 POST
	if c.Request.Method != "POST" {
		c.JSON(405, gin.H{"error": "Invalid request method"})
		return
	}

	fmt.Println("22222")

	// 获取所有表单数据
	form, err := c.MultipartForm()
	if err != nil {
		c.JSON(400, gin.H{"error": "Error retrieving form data"})
		return
	}
	fmt.Println("33333")

	// 获取文件部分
	files := form.File["files"] // 获取字段名为 "files" 的文件
	if len(files) == 0 {
		c.JSON(400, gin.H{"error": "No files uploaded"})
		return
	}

	fmt.Println("44444")

	// tarFiles := []string{}
	tempDir, err := os.MkdirTemp("", "uploads-*")
	defer os.RemoveAll(tempDir)

	if err != nil {
		fmt.Println("55555")
		c.JSON(500, gin.H{"error": fmt.Sprintf("Error creating temp dir: %v", err)})
		return
	}
	for _, fileHeader := range files {
		func() {
			// 打开上传的文件
			if !strings.HasSuffix(fileHeader.Filename, ".tar") {
				return
			}
			file, err := fileHeader.Open()
			if err != nil {
				fmt.Println("66666")

				c.JSON(500, gin.H{"error": fmt.Sprintf("Error opening file %s: %v", fileHeader.Filename, err)})
				return
			}
			defer file.Close()

			// 创建目标文件

			dstPath := filepath.Join(tempDir, fileHeader.Filename)
			if err := os.MkdirAll(filepath.Dir(dstPath), 0755); err != nil {
				fmt.Println("77777")
				c.JSON(500, gin.H{"error": fmt.Sprintf("Error creating directory: %v", err)})
				return
			}
			dstFile, err := os.Create(dstPath)
			if err != nil {
				fmt.Println("88888")
				c.JSON(500, gin.H{"error": fmt.Sprintf("Error saving file %s: %v", fileHeader.Filename, err)})
				return
			}
			defer dstFile.Close()

			// 将上传的文件内容写入目标文件
			if _, err := io.Copy(dstFile, file); err != nil {
				fmt.Println("99999")
				c.JSON(500, gin.H{"error": fmt.Sprintf("Error writing file %s: %v", fileHeader.Filename, err)})
				return
			}

			// tarFiles = append(tarFiles, dstPath)
			fmt.Printf("File uploaded successfully: %s\n", fileHeader.Filename)
		}()
	}

	succress, err := (&ImgUploaderUploadHandlerV2{}).checkAndUploadImgs(tempDir)
	if err != nil {
		fmt.Println("aaaaa")
		c.JSON(500, gin.H{"error": fmt.Errorf("checkAndUploadImgs failed: %v", err)})
		return
	}

	c.JSON(200, gin.H{"success": succress})
	return
}
