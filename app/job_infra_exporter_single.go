package app

import (
	"bytes"
	"fmt"
	"net/http"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"telego/util"
	"telego/util/gemini"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/cobra"
)

type ModJobInfraExporterSingleStruct struct{}

var ModJobInfraExporterSingle ModJobInfraExporterSingleStruct

var (
	// Gemini 配置
	geminiSpaceStoragePath string

	// Token
	token      string
	tokenMutex sync.Mutex

	// Gemini-storage metrics
	spaceStorageUsed = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "Gemini_space_storage_used",
			Help: "Gemini_space_storage_used_by_MB",
		},
		[]string{"spaceID"},
	)
)

func init() {
	geminiSpaceStoragePath = "/juicefs-minio-conf-nm/space_old"

	// Register Prometheus metrics
	prometheus.MustRegister(spaceStorageUsed)
}

func (m ModJobInfraExporterSingleStruct) JobCmdName() string {
	return "infra-exporter-single"
}

func (m ModJobInfraExporterSingleStruct) Run() {
	go collectMetrics()

	http.Handle("/metrics", promhttp.Handler())
	fmt.Println("Serving metrics on: 8082")
	http.ListenAndServe(":8082", nil)
}

func (m ModJobInfraExporterSingleStruct) ParseJob(infraExporterSingleCmd *cobra.Command) *cobra.Command {

	infraExporterSingleCmd.Run = func(_ *cobra.Command, _ []string) {
		m.Run()
	}

	return infraExporterSingleCmd
}

func collectMetrics() {
	// 与 Gemini API 进行交互，拿到所有的 spaceID
	gUrl, err := (util.MainNodeConfReader{}).ReadSecretConf(util.SecretConfTypeGeminiAPIUrl{})
	if err != nil {
		fmt.Println("failed to read gemini config")
	}
	gUrl = strings.TrimSpace(gUrl)
	gServer, err := gemini.NewGeminiServer(gUrl)
	if err != nil {
		fmt.Println("failed to create Gemini server ", err)
	}

	// 一次更新后，定时更新 token
	// 暂时改成阻塞
	go updateToken(gServer, 30*60)
	time.Sleep(5 * time.Second)

	// 拿取存储信息
	for {
		tokenMutex.Lock()
		currentToken := token
		tokenMutex.Unlock()

		req := &gemini.SpaceListRequest{
			Header: gemini.SpaceListRequestHeader{
				Authorization: "Bearer " + currentToken,
			},
		}
		resp, err := gServer.WebAPIFunc.SpaceList(req)
		if err != nil {
			fmt.Println("get space list error: ", err)
			time.Sleep(5 * time.Second)
			continue
		} else {
			fmt.Println(resp.Body.Data.SpaceList)
		}

		for _, thisSpace := range resp.Body.Data.SpaceList {
			thisSpaceStorageUsed, err := getStorageUsed(geminiSpaceStoragePath+"/"+thisSpace.SpaceID, "MB")
			if err != nil {
				// fmt.Println("get current storage used error: ", err)
			} else {
				fmt.Printf("%s used %v: \n", thisSpace.SpaceID, thisSpaceStorageUsed)
			}
			spaceStorageUsed.WithLabelValues(thisSpace.SpaceID).Set(thisSpaceStorageUsed)
		}
		time.Sleep(1 * time.Minute)
	}
}

func updateToken(gServer *gemini.GeminiServer, intervalBySecond int) {
	if gServer == nil {
		fmt.Println("gServer not initialized")
		return
	}

	req := &gemini.PasswdLoginRequest{
		Header: gemini.PasswdLoginRequestHeader{},
		Body: gemini.PasswdLoginRequestBody{
			UserName: "gemini",
			Password: "Gemini123",
		},
	}
	for {
		resp, err := gServer.UserAuthFunc.PasswdLogin(req)
		if err != nil {
			fmt.Println("failed to fetch token: ", err)
			time.Sleep(5 * time.Second)
			continue
		}

		if resp == nil {
			fmt.Println("response is empty")
			time.Sleep(5 * time.Second)
			continue
		}

		if resp.Body.Data.Token == "" {
			fmt.Println("token is empty")
			fmt.Println(resp)
			time.Sleep(5 * time.Second)
			continue
		}

		tokenMutex.Lock()
		token = resp.Body.Data.Token
		tokenMutex.Unlock()

		fmt.Println("token updated successfully, token: ", token)
		time.Sleep(time.Duration(intervalBySecond) * time.Second)
	}
}

func getStorageUsed(fullPath, unit string) (float64, error) {
	cmd := exec.Command("du", "-B1", "-s", fullPath)
	var out bytes.Buffer
	cmd.Stdout = &out

	err := cmd.Run()
	if err != nil {
		return 0, fmt.Errorf("getStorageUsed: failed to run du -B1 -s %s: %v", fullPath, err)
	}

	output := strings.TrimSpace(out.String())

	// 解析 du 指令的第 1 列
	fields := strings.Fields(output)
	if len(fields) < 1 {
		return -1, fmt.Errorf("getStorageUsed: unexpected output format: %s", output)
	}

	usedBytes, err := strconv.ParseFloat(fields[0], 64)
	if err != nil {
		return -1, fmt.Errorf("getStorageUsed: failed to parse size: %v", err)
	}

	// 转换单位
	switch strings.ToUpper(unit) {
	case "KB":
		return usedBytes / 1024, nil
	case "MB":
		return usedBytes / (1024 * 1024), nil
	case "TB":
		return usedBytes / (1024 * 1024 * 1024), nil
	case "PB":
		return usedBytes / (1024 * 1024 * 1024 * 1024), nil
	default:
		return -1, fmt.Errorf("getStorageUsed: invalid unit: %s (valid units: KB, MB, TB, PB)", unit)
	}
}
