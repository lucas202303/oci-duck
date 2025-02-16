package utils

import (
	"regexp"
	"strings"
)

func WildCardToRegexp(pattern string) string {
	var result strings.Builder
	for i, literal := range strings.Split(pattern, "*") {
		if i > 0 {
			result.WriteString(".*")
		}
		result.WriteString(regexp.QuoteMeta(literal))
	}
	return result.String()
}

func Match(pattern string, value string) (re string) {
	result, _ := regexp.MatchString(WildCardToRegexp(pattern), value)
	var resu string
	if !result {
		resu = "false"
	} else {
		resu = "true"
	}
	return resu
}
