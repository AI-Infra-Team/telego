package util

import (
	"crypto/rand"
	"math/big"
)

func GenerateRandomStringFromInput(input string, length int) (string, error) {
	var randomStr []byte
	for i := 0; i < length; i++ {
		// 使用 crypto/rand 来选择字符
		idx, err := rand.Int(rand.Reader, big.NewInt(int64(len(input))))
		if err != nil {
			return "", err
		}
		randomStr = append(randomStr, input[idx.Int64()])
	}
	return string(randomStr), nil
}
