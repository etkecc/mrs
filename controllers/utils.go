package controllers

import (
	"strconv"
	"strings"
)

func string2int(value string, defaultValue int) int {
	vInt, err := strconv.Atoi(value)
	if err != nil {
		return defaultValue
	}

	return vInt
}

func string2slice(value string, defaultValue string) []string {
	value = strings.TrimSpace(value)
	if idx := strings.Index(value, ","); idx == -1 {
		value = defaultValue
	}
	return strings.Split(value, ",")
}
