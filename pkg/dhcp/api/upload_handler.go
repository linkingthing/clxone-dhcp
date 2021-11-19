package api

import (
	"fmt"
	"net/http"
	"os"
	"path"
	"path/filepath"

	"github.com/gin-gonic/gin"

	"github.com/linkingthing/clxone-dhcp/pkg/util"
)

func UploadFiles(ctx *gin.Context) {
	form, err := ctx.MultipartForm()
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	var directory string
	if len(form.Value["directory"]) > 0 {
		directory = form.Value["directory"][0]
	}
	files := form.File["path"]
	var fileNames string
	for _, file := range files {
		if err := createFolder(directory); err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{
				"error": err.Error(),
			})
			return
		}
		filename := path.Join(directory, filepath.Base(file.Filename))
		if err := ctx.SaveUploadedFile(file,
			path.Join(util.FileRootPath, filename)); err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{
				"error": err.Error(),
			})
			return
		}
		fileNames = filename
	}

	ctx.JSON(http.StatusOK, gin.H{
		"filename": fileNames,
	})
}

func createFolder(folderName string) error {
	if _, err := os.Stat(path.Join(util.FileRootPath, folderName)); os.IsNotExist(err) {
		if err := os.Mkdir(path.Join(util.FileRootPath, folderName), 0777); err != nil {
			return fmt.Errorf("createFolder %s failed:%s ", folderName, err.Error())
		}
	}

	return nil
}
