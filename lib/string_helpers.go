package lib

import (
	"fmt"
	"net/url"
)

func urlPathLastN(us string, n int) string {
	u, err := url.Parse(us)
	if err != nil {
		return stringLastN(us, n)
	}
	return stringLastN(u.Path, n)
}
func stringLastN(str string, n int) string {
	if len(str) <= n {
		return str
	}
	return fmt.Sprintf("...%s", string([]rune(str)[len(str)-n:]))
}
