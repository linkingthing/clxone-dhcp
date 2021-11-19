package resource

import (
	restresource "github.com/linkingthing/gorest/resource"
)

const ActionNameExportCSV = "exportcsv"

type ExportFilter struct {
	From string `json:"from"`
	To   string `json:"to"`
}

type FileInfo struct {
	Path string `json:"path"`
}

var exportActions = []restresource.Action{
	restresource.Action{
		Name:   ActionNameExportCSV,
		Input:  &ExportFilter{},
		Output: &FileInfo{},
	},
}
