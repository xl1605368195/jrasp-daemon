package utils

import (
	"crypto/md5"
	"encoding/hex"
	"io"
	"os"
)

func PathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func CalcFileHash(pluginPath string) (string, error) {
	file, err := os.Open(pluginPath)
	if err != nil {
		return "", err
	}
	defer func() {
		ferr := file.Close()
		if ferr != nil {
			err = ferr
		}
	}()
	h := md5.New()
	_, err = io.Copy(h, file)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
