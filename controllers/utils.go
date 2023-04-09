package controllers

import "strconv"

func string2int(value string, defaultValue int) int {
	vInt, err := strconv.Atoi(value)
	if err != nil {
		return defaultValue
	}

	return vInt
}
