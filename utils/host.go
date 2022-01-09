package utils

import (
	"os"
	"strings"
)

func GetHostname() string {
	var host_re = "unknown"
	hostname, err := os.Hostname()
	if err != nil {
		return host_re
	}
	hostname = strings.ToLower(hostname)
	return hostname
}
