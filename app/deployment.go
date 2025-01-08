package app

import (
	"fmt"
	"io/fs"
	"os"
	"path"
	"reflect"
	"strconv"
	"strings"
	"telego/util"
	"telego/util/yamlext"

	"github.com/mitchellh/mapstructure"
	"github.com/thoas/go-funk"
)

func StrPtr(s string) *string {
	return &s
}

type LocalValue interface {
	LocalValueInterfaceDummy()
}

type LocalValueStr struct {
	Value string
}

func (s LocalValueStr) LocalValueInterfaceDummy() {}

// replace:
//
//	match: "\n"
//	with: "    \n"
type LocalValueReadFileReplace struct {
	Match string `mapstructure:"match"`
	With  string `mapstructure:"with"`
}

// func (l *LocalValueReadFileReplace) TransUnquote() {

// 	new := util.Unquote(l.Match)
// 	util.Logger.Debugf("replace: %s -> %s , %v, %v", l.Match, new, strings.Contains(l.Match, "\n"), strings.Contains(new, "\n"))
// 	l.Match = new

// 	new = util.Unquote(l.With)
// 	util.Logger.Debugf("replace: %s -> %s", l.With, new)
// 	l.With = new
// }

type LocalValueReadFile struct {
	ReadFromFile string                     `mapstructure:"read_from_file"`
	Replace      *LocalValueReadFileReplace `mapstructure:"replace,omitempty"`
}

func (l LocalValueReadFile) LocalValueInterfaceDummy() {}

// DeploymentYaml represents the structure of yml.
type DeploymentYaml struct {
	Comment     string                            `yaml:"comment"`
	LocalValues map[string]interface{}            `yaml:"local_values"`
	Secrets     []string                          `yaml:"secret,omitempty"`
	Prepare     []DeploymentPrepareItemYaml       `yaml:"prepare"`
	Helms       map[string]DeploymentHelm         `yaml:"helms,omitempty"`
	K8s         map[string]DeploymentK8s          `yaml:"k8s,omitempty"`
	Bin         map[string]DeploymentBinDetails   `yaml:"bin,omitempty"`
	Dist        map[string]DeploymentDistConfYaml `yaml:"dist,omitempty"`
}

type Deployment struct {
	Comment     string
	LocalValues map[string]LocalValue
	Secrets     []string
	Prepare     []DeploymentPrepareItem
	Helms       map[string]DeploymentHelm
	K8s         map[string]DeploymentK8s
	Bin         map[string]DeploymentBinDetails
	Dist        map[string]DeploymentDistConfYaml `yaml:"dist,omitempty"`
}

func (d *DeploymentYaml) Verify(prjName string, prjDir string, yml []byte, skipReadFromFile bool) (*Deployment, error) {

	dply := &Deployment{
		Comment: d.Comment,
		// default values
		LocalValues: map[string]LocalValue{
			"MAIN_NODE_IP": LocalValueStr{Value: util.MainNodeIp},
			"IMG_REPO":     LocalValueStr{Value: util.ImgRepoAddressNoPrefix},
			"BIN_PRJ":      LocalValueStr{Value: prjName},
		},
		Secrets: d.Secrets,
		Prepare: []DeploymentPrepareItem{},
		Helms:   d.Helms,
		K8s:     d.K8s,
		Bin:     d.Bin,
		Dist:    d.Dist,
	}
	// trans to specific interface
	for idx, v := range d.LocalValues {
		_, ok := v.(string)
		if ok {
			dply.LocalValues[idx] = LocalValueStr{Value: v.(string)}
		} else {
			target := LocalValueReadFile{}
			err := mapstructure.Decode(v, &target)
			if err != nil {
				return nil, fmt.Errorf("invalid local_values item: only string or read_file is allowed")
			}
			dply.LocalValues[idx] = target
		}
	}

	// resolve local values self
	foreachLocalValue := func(cb func(k string, v LocalValue)) {
		for k, v := range dply.LocalValues {
			cb(k, v)
		}
	}
	// first iter replace ${key} in value or file path
	for k, _ := range dply.LocalValues {
		switch dply.LocalValues[k].(type) {
		case LocalValueStr:
			foreachLocalValue(func(otherK string, otherV LocalValue) {
				otherV_, ok := otherV.(LocalValueStr)
				if ok {
					switch dply.LocalValues[k].(type) {
					case LocalValueStr:
						v_ := dply.LocalValues[k].(LocalValueStr)
						v_.Value = strings.ReplaceAll(v_.Value, fmt.Sprintf("${%s}", otherK), otherV_.Value)
						dply.LocalValues[k] = v_
					case LocalValueReadFile:
						v_ := dply.LocalValues[k].(LocalValueReadFile)
						v_.ReadFromFile = strings.ReplaceAll(v_.ReadFromFile, fmt.Sprintf("${%s}", otherK), otherV_.Value)
						dply.LocalValues[k] = v_
					}
				}
			})
		}
	}

	if !skipReadFromFile {
		// second iter read files
		for k, v := range dply.LocalValues {
			switch v := v.(type) {
			case LocalValueReadFile:
				fileContent, err := os.ReadFile(path.Join(prjDir, v.ReadFromFile))
				if err != nil {
					return nil, fmt.Errorf("invalid local_values item: read_from_file %s not found", path.Join(prjDir, v.ReadFromFile))
				}
				v.ReadFromFile = string(fileContent)
				dply.LocalValues[k] = v
			}
		}

		// third iter again replace all strs
		for k, _ := range dply.LocalValues {
			// switch v.(type) {
			// case LocalValueStr:
			foreachLocalValue(func(otherK string, otherV LocalValue) {
				// fmt.Printf("%s,%s:%s,%s\n", k, v, otherK, otherV)
				otherStr := ""
				switch otherV.(type) {
				case LocalValueStr:
					otherStr = otherV.(LocalValueStr).Value
				case LocalValueReadFile:
					otherStr = otherV.(LocalValueReadFile).ReadFromFile
				}
				switch dply.LocalValues[k].(type) {
				case LocalValueStr:
					v_ := dply.LocalValues[k].(LocalValueStr)
					newValue := strings.ReplaceAll(v_.Value, fmt.Sprintf("${%s}", otherK), otherStr)
					v_.Value = newValue
					dply.LocalValues[k] = v_
				case LocalValueReadFile:
					v_ := dply.LocalValues[k].(LocalValueReadFile)
					// fmt.Printf("replace %s in %s with %s\n", fmt.Sprintf("${%s}", otherK), v_.ReadFromFile, otherStr)
					newValue := strings.ReplaceAll(v_.ReadFromFile, fmt.Sprintf("${%s}", otherK), otherStr)
					// if newValue != v_.ReadFromFile {
					// 	fmt.Printf("%s -> %s\n", v_.ReadFromFile, newValue)
					// }
					v_.ReadFromFile = newValue
					replace := v_.Replace
					if replace != nil {
						v_.Replace = nil // consume it, so that it will not be replaced again
						// replace.TransUnquote()
						v_.ReadFromFile = strings.ReplaceAll(
							v_.ReadFromFile,
							replace.Match, replace.With)
						// util.Logger.Debugf("replace in value %s, (%s->%s)", k, replace.Match, replace.With)
					}
					dply.LocalValues[k] = v_
				}
			})
			// }
		}
	}

	// forth iter checks ${...} remains
	type ErrorOwnerError struct {
		err error
	}
	errOwner := &ErrorOwnerError{}
	foreachLocalValue(func(k string, v LocalValue) {
		switch v.(type) {
		case LocalValueStr:
			v_ := v.(LocalValueStr)
			if strings.Contains(v_.Value, "${") {
				errOwner.err = fmt.Errorf("invalid local_values item: ${...} not replaced: %s\n  values: %v", v_.Value, dply.LocalValues)
			}
		case LocalValueReadFile:
			v_ := v.(LocalValueReadFile)
			if strings.Contains(v_.ReadFromFile, "${") {
				errOwner.err = fmt.Errorf("invalid local_values item: ${...} not replaced: %s\n  values: %v", v_.ReadFromFile, dply.LocalValues)
			}
		}
	})
	if errOwner.err != nil {
		return nil, errOwner.err
	}

	// transform other field
	for _, item := range d.Prepare {
		itemTyped, err := item.To(util.Empty{})
		if err != nil {
			return nil, fmt.Errorf("invalid prepare item: %w", err)
		}
		dply.Prepare = append(dply.Prepare, *itemTyped)
	}

	//replace all string with value
	replaceWithValue := func(s *string) {
		if s == nil {
			return
		}
		foreachLocalValue(func(k string, v LocalValue) {
			vStr := ""
			switch v := v.(type) {
			case LocalValueStr:
				vStr = v.Value
			case LocalValueReadFile:
				vStr = v.ReadFromFile
			}

			*s = strings.ReplaceAll(*s, fmt.Sprintf("${%s}", k), vStr)
		})
	}
	for _, b := range dply.Bin {
		replaceWithValue(b.PyInstaller0)
	}
	for _, s := range dply.Helms {
		replaceWithValue(s.HelmDir)
		replaceWithValue(s.Namespace)
	}
	for _, s := range dply.Prepare {
		replaceWithValue(s.As)
		replaceWithValue(s.Image)
		replaceWithValue(s.URL)
		replaceWithValue(s.Pyscript)
		if s.FileMap != nil {
			replaceWithValue(s.FileMap.Content)
			replaceWithValue(s.FileMap.Path)
		}
		for _, t := range s.Trans {
			switch t.(type) {
			case DeploymentTransformCopy:
				for _, v := range t.(DeploymentTransformCopy).Copy {
					replaceWithValue(v.from)
					replaceWithValue(v.to)
				}
			case DeploymentTransformExtract:
			}
		}
	}

	return dply, nil
}

// DeploymentPrepareItemYaml represents an item in the "prepare" list.
type DeploymentPrepareItemYaml struct {
	Image    string             `yaml:"image,omitempty"`
	URL      string             `yaml:"url,omitempty"`
	As       string             `yaml:"as,omitempty"`
	FileMap  *DeploymentFileMap `yaml:"filemap,omitempty"`
	Trans    []interface{}      `yaml:"trans,omitempty"`
	Pyscript string             `yaml:"pyscript,omitempty"`
}

type DeploymentPrepareItem struct {
	Image    *string
	URL      *string
	As       *string
	Pyscript *string
	FileMap  *DeploymentFileMap
	Trans    []DeploymentTransform
}

var _ util.Conv[util.Empty, *DeploymentPrepareItem] = &DeploymentPrepareItemYaml{}

func (i *DeploymentPrepareItemYaml) To(util.Empty) (*DeploymentPrepareItem, error) {
	count := 0
	if i.Image != "" {
		count++
	}
	if i.URL != "" {
		count++
	}
	if i.FileMap != nil {
		count++
	}
	if i.Pyscript != "" {
		count++
	}

	if count != 1 {
		return nil, fmt.Errorf("only one of image/url/filemap/pyscript can be specified")
	}
	item := &DeploymentPrepareItem{
		Image:    StrPtr(i.Image),
		URL:      StrPtr(i.URL),
		As:       StrPtr(i.As),
		FileMap:  i.FileMap,
		Trans:    []DeploymentTransform{},
		Pyscript: StrPtr(i.Pyscript),
	}
	caseStrInterface := func(trans map[string]interface{}) error {
		transMap := trans
		target := map[string][]map[string]string{}
		err := mapstructure.Decode(transMap, &target)
		copy, exist := target["copy"]
		if !exist {
			err = fmt.Errorf("could not find copy in one trans")
		}
		for _, oneCopy := range copy {
			if len(oneCopy) != 1 {
				err = fmt.Errorf("one copy should be single kv as {src: dest}")
				break
			}
		}
		if err != nil {
			err := fmt.Errorf("invalid prepare/trans item: only 'extract' or {'copy':[{'from':'to'}]} is allowed, \n  err: %v, \n  current is %v %v", err, trans, reflect.TypeOf(trans))
			return err
		}
		// i.Trans[idx] = DeploymentTransformCopy{
		// 	Copy: copy,
		// }
		item.Trans = append(item.Trans, DeploymentTransformCopy{
			Copy: funk.Map(copy, func(v map[string]string) DeploymentTransformCopyOne {
				firstK := ""
				firstV := ""
				for k, v := range v {
					firstK = k
					firstV = v
				}
				return DeploymentTransformCopyOne{
					from: &firstK,
					to:   &firstV,
				}
			}).([]DeploymentTransformCopyOne),
		})
		return nil
	}
	for _, trans := range i.Trans {
		switch trans := trans.(type) {
		case string:
			if trans == "extract" {
				item.Trans = append(item.Trans, DeploymentTransformExtract{})
				continue
			} else {
				return nil, fmt.Errorf("invalid prepare/trans item: only 'extract' is allowed")
			}
		case map[string]interface{}:
			if err := caseStrInterface(trans); err != nil {
				return nil, err
			}
		case map[interface{}]interface{}:
			var transStrInterface map[string]interface{}
			err := mapstructure.Decode(trans, &transStrInterface)
			if err != nil {
				return nil, fmt.Errorf("invalid prepare/trans item: only 'extract' or {'copy':[{'from':'to'}]} is allowed, \n  err: %v, \n  current is %v %v", err, trans, reflect.TypeOf(trans))
			}
			if err := caseStrInterface(transStrInterface); err != nil {
				return nil, err
			}
		default:
			err := fmt.Errorf("invalid prepare/trans item: only 'extract' or {'copy':[{'from':'to'}]} is allowed, \n  current is %v %v", trans, reflect.TypeOf(trans))
			return nil, err
		}
	}
	return item, nil
}

type DeploymentTransform interface {
	DeploymentTransformDummy()
}

type DeploymentTransformCopyOne struct {
	from *string
	to   *string
}

// Example
//
//	trans:
//	- copy:
//	  - rclone-v1.68.0-windows-amd64/rclone.exe: rclone_windows.exe
type DeploymentTransformCopy struct {
	Copy []DeploymentTransformCopyOne
}

func (c DeploymentTransformCopy) DeploymentTransformDummy() {}

type DeploymentTransformExtract struct{}

func (e DeploymentTransformExtract) DeploymentTransformDummy() {}

// DeploymentFileMap represents the file mapping details.
type DeploymentFileMap struct {
	Content *string `yaml:"content"`
	Path    *string `yaml:"path"`
	Mode    *string `yaml:"mode"`
}

// WriteToFile 打开文件并写入内容
func (dfm *DeploymentFileMap) WriteToFile() error {
	// 检查 Path 和 Content 是否有效
	if dfm.Path == nil || dfm.Content == nil {
		return fmt.Errorf("path or content is nil")
	}

	if len(*dfm.Mode) == 3 {
		*dfm.Mode = "0" + *dfm.Mode
	}

	mode, err := strconv.ParseInt(*dfm.Mode, 8, 64)
	if err != nil {
		return fmt.Errorf("Error parsing octal string: %w", err)
	}

	dir := path.Dir(*dfm.Path)
	err = os.MkdirAll(dir, 0755)
	if err != nil {
		return fmt.Errorf("Error creating directory: %w", err)
	}

	// 创建或打开文件
	file, err := os.OpenFile(*dfm.Path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, fs.FileMode(mode))
	if err != nil {
		return fmt.Errorf("failed to open file %s: %w", *dfm.Path, err)
	}
	defer file.Close()

	// 写入内容
	_, err = file.WriteString(*dfm.Content)
	if err != nil {
		return fmt.Errorf("failed to write to file %s: %w", *dfm.Path, err)
	}

	return nil
}

// DeploymentHelm represents a helm chart configuration.
type DeploymentHelm struct {
	HelmDir   *string `yaml:"helm-dir"`
	Namespace *string `yaml:"namespace"`
}

// DeploymentK8s represents a Kubernetes configuration.
type DeploymentK8s struct {
	K8sDir    *string `yaml:"k8s-dir"`
	Namespace *string `yaml:"namespace"`
}

func LoadDeploymentYml(prjName string, subPrjDir string) (*Deployment, error) {
	ymlFile := path.Join(subPrjDir, "deployment.yml")
	if path.Base(ymlFile) != "deployment.yml" {
		util.Logger.Fatal("yml not found")
		os.Exit(1)
	}
	if !util.PathIsAbsolute(ymlFile) {
		util.Logger.Fatalf("path should be absolute %s", ymlFile)
		os.Exit(1)
	}

	data, err := os.ReadFile(ymlFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read YAML file: %w", err)
	}

	return LoadDeploymentYmlByContent(prjName, subPrjDir, data)
}

func LoadDeploymentYmlByContent(prjName string, projectDir string, data []byte) (*Deployment, error) {
	var deployment_ DeploymentYaml

	// fmt.Printf("load deployment yml str: %s", string(data))

	err := yamlext.UnmarshalAndValidate(data, &deployment_)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal YAML file at projectDir , %w", err)
	}

	// fmt.Printf("load deployment yml obj: %+v", deployment_)

	skipReadFile := false
	if projectDir == "" {
		skipReadFile = true
	}
	d, err := deployment_.Verify(prjName, projectDir, data, skipReadFile)
	if err != nil {
		return nil, fmt.Errorf("invalid yml format: %w", err)
	}
	return d, nil
}

func DeploymentOpePretreatment(project string, deployment *Deployment) error {
	if strings.HasPrefix(project, "dist_") {
		if deployment.Prepare == nil {
			deployment.Prepare = []DeploymentPrepareItem{}
		}
		// add prepare: image: alpine:3.14
		deployment.Prepare = append(deployment.Prepare, DeploymentPrepareItem{
			Image: StrPtr("python:3.12.5"),
		}, DeploymentPrepareItem{
			Image: StrPtr("alpine/openssh:9.1"),
		})
	}
	return nil
}
