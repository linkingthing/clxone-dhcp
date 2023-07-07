package service

import (
	"github.com/linkingthing/clxone-dhcp/pkg/errorno"
	"time"

	"github.com/linkingthing/clxone-utils/excel"
)

func exportFile(ctx *MetricContext, contents [][]string) (string, error) {
	fpath, err := excel.WriteExcelFile(ctx.NodeIP+"-"+string(ctx.MetricName)+"-"+string(ctx.Version)+
		"-"+time.Now().Format(excel.TimeFormat), ctx.TableHeader, contents)
	if err != nil {
		return "", errorno.ErrExport(errorno.ErrName(ctx.MetricName), err.Error())
	}
	return fpath, nil
}
