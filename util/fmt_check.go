package util

import "regexp"

type FmtCheckStruct struct {
}

var FmtCheck FmtCheckStruct

func (f FmtCheckStruct) CheckImagePath(s string) bool {
	pattern := `^[a-zA-Z0-9_-]+(?:/[a-zA-Z0-9_-]+)*(:[a-zA-Z0-9_.-]+)?$`
	re := regexp.MustCompile(pattern)
	return re.MatchString(s)
}
