package app

import (
	"telego/util/yamlext"
	"testing"
)

func Test_ConfigExporterPipeline_getFinalValue(t *testing.T) {
	value := `
user: admin
paSSword: DpN*TZH3y3-F9-Ga18a
uplcader_store_addr: http://1e.127.16.8:318e8
uplcader_store_admin: teleinfra
uplcader_store_adnin_pw: Jxbnhvp4$
uplcader_store_transfer_addr: 127.16.8:32212
`
	var parsedMap map[string]interface{}
	err := yamlext.UnmarshalAndValidate([]byte(value), &parsedMap)
	if err != nil {
		t.Fatalf("failed to parse value %s as YAML: %v", value, err)
	}
}
