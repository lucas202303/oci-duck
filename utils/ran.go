package utils

import (
	rand2 "crypto/rand"

	"math/big"

	"strings"
)

func RandomkeyGenerate(from string, length int, duplicate string) (result string) {

	var Key string
	if duplicate == "true" {
		b := make([]byte, length)
		for i := range b {
			c, err := rand2.Int(rand2.Reader, big.NewInt(int64(len(from))))
			if err != nil {
				panic(err)
			}
			b[i] = from[c.Int64()]
		}
		return string(b)
	} else if duplicate == "false" {
		for i := 1; i < length+1; i++ {
			// 先生成数据
			b := make([]byte, 1)
			for i := range b {
				c, err := rand2.Int(rand2.Reader, big.NewInt(int64(len(from))))
				if err != nil {
					panic(err)
				}
				b[i] = from[c.Int64()]
			}
			// 再from删除生成的数据
			from = strings.Replace(from, string(b), "", -1)
			// 再添加数据
			Key = Key + string(b)
		}
		return Key
	}
	return
}
