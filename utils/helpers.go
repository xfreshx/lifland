package utils

import (
	"errors"
	"net/url"
	"strconv"
)

func GetStringURLParam(params url.Values, name string) (string, error) {
	ret := params.Get(name)
	if ret == "" {
		return "", errors.New("no such param")
	}

	return ret, nil
}

func GetUintURLParam(params url.Values, name string) (uint64, error) {
	_ret := params.Get(name)
	if _ret == "" {
		return 0, errors.New("no such param")
	}

	ret, err := strconv.ParseUint(_ret, 10, 64)
	if err != nil {
		return 0, err
	}

	return ret, nil
}
