package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
)

func inSlice(str string, l []string) bool {
	for i := range l {
		if l[i] == str {
			return true
		}
	}
	return false
}

// FormatBytes Convert bytes to human readable string. Like a 2 MB, 64.2 KB, 52 B
func FormatBytes(i int64) (result string) {
	switch {
	case i > (1024 * 1024 * 1024 * 1024):
		result = fmt.Sprintf("%#.02f TB", float64(i)/1024/1024/1024/1024)
	case i > (1024 * 1024 * 1024):
		result = fmt.Sprintf("%#.02f GB", float64(i)/1024/1024/1024)
	case i > (1024 * 1024):
		result = fmt.Sprintf("%#.02f MB", float64(i)/1024/1024)
	case i > 1024:
		result = fmt.Sprintf("%#.02f KB", float64(i)/1024)
	default:
		result = fmt.Sprintf("%d B", i)
	}
	result = strings.Trim(result, " ")
	return
}

const EnvPrefix = "GURL_"

func flagEnv(name, value, usage string) *string {
	if value == "" {
		value = os.Getenv(EnvPrefix + strings.ToUpper(name))
	}
	return flag.String(name, value, usage)
}

func flagEnvVar(p *string, name, value, usage string) {
	if value == "" {
		value = os.Getenv(EnvPrefix + strings.ToUpper(name))
	}
	flag.StringVar(p, name, value, usage)
}
