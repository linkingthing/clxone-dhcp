package util

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
)

func ReadFile(fileName string) ([]byte, error) {
	file, err := os.Open(path.Join(FileRootPath, fileName))
	if err != nil {
		return nil, err
	}

	defer file.Close()
	return ioutil.ReadAll(file)
}

func RemoveFile(fileName string) error {
	if _, err := os.Stat(path.Join(FileRootPath, fileName)); !os.IsNotExist(err) {
		if err := os.Remove(path.Join(FileRootPath, fileName)); err != nil {
			return fmt.Errorf("remove %s failed:%s ", fileName, err.Error())
		}
	}

	return nil
}

func RemoveFolder(folderName string) error {
	if dir, _ := path.Split(folderName); dir != "" {
		if _, err := os.Stat(path.Join(FileRootPath, dir)); !os.IsNotExist(err) {
			if err := os.RemoveAll(path.Join(FileRootPath, dir)); err != nil {
				return fmt.Errorf("remove %s failed:%s ", folderName, err.Error())
			}
		}
	}

	return nil
}

func CreateFolder(folderName string) error {
	if _, err := os.Stat(path.Join(FileRootPath, folderName)); os.IsNotExist(err) {
		if err := os.Mkdir(path.Join(FileRootPath, folderName), 0777); err != nil {
			return fmt.Errorf("createFolder %s failed:%s ", folderName, err.Error())
		}
	}

	return nil
}
