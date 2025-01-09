package yamlext

import (
	"bytes"
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/thoas/go-funk"
	"gopkg.in/yaml.v2"
)

// GenerateYAMLTemplate 生成 YAML 模板并为 omitempty 字段增加注释
func GenerateYAMLTemplate(v interface{}) string {
	t := reflect.TypeOf(v)
	if t.Kind() != reflect.Struct {
		panic("input must be a struct")
	}

	// 获取字段的 omitempty 信息
	omitemptyFields := map[string]bool{}
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		tag := field.Tag.Get("yaml")
		if strings.Contains(tag, "omitempty") {
			yamlKey := strings.Split(tag, ",")[0]
			if yamlKey == "" {
				yamlKey = field.Name
			}
			omitemptyFields[yamlKey] = true
		}
	}

	// 序列化 YAML
	var yamlData bytes.Buffer
	encoder := yaml.NewEncoder(&yamlData)
	if err := encoder.Encode(v); err != nil {
		panic(err)
	}
	encoder.Close()

	// 添加注释
	var buffer bytes.Buffer
	lines := strings.Split(yamlData.String(), "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "") && strings.Contains(trimmed, ":") {
			key := strings.Split(trimmed, ":")[0]
			key = strings.TrimSpace(key)
			if omitemptyFields[key] {
				line += " # 允许缺省值"
			}
		}
		buffer.WriteString(line + "\n")
	}

	return buffer.String()
}

// UnmarshalAndValidate decodes YAML data into the provided struct
// and validates that all fields have been set, considering omitempty tag.
func UnmarshalAndValidate(data []byte, v interface{}) error {
	// Step 1: Unmarshal YAML into the provided struct
	if err := yaml.Unmarshal(data, v); err != nil {
		return fmt.Errorf("failed to unmarshal YAML: %w", err)
	}

	// Step 2: Validate that all fields have been set
	val := reflect.ValueOf(v)
	if val.Kind() != reflect.Ptr || val.IsNil() {
		return errors.New("provided value must be a non-nil pointer")
	}

	// not struct don't need validate
	val = val.Elem()
	if val.Kind() != reflect.Struct {
		return nil
	}

	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		fieldType := val.Type().Field(i)
		fieldName := fieldType.Name

		// fieldType.Tag fmt
		//   yaml:"comment,omitempty"
		tags := strings.Split(fieldType.Tag.Get("yaml"), ",")
		// fmt.Println("field", fieldName, "tags", tags)
		// Skip validation for fields with the omitempty tag
		if funk.ContainsString(tags, "omitempty") {
			continue
		}

		// Check for valid and non-nil values for all other fields
		if !field.IsValid() || (field.Kind() == reflect.Ptr && field.IsNil()) {
			return fmt.Errorf("field %q is not set with %v", fieldName, v)
		}

		// Check for zero value in non-pointer fields
		if field.Kind() != reflect.Ptr && reflect.DeepEqual(field.Interface(), reflect.Zero(field.Type()).Interface()) {
			return fmt.Errorf("field %q is not set with %v", fieldName, v)
		}
	}

	return nil
}
