package service

import (
	"time"

	"github.com/linkingthing/clxone-utils/excel"
)

func exportFile(ctx *MetricContext, contents [][]string) (string, error) {
	return excel.WriteExcelFile(ctx.NodeIP+"-"+string(ctx.MetricName)+"-"+string(ctx.Version)+
		"-"+time.Now().Format(excel.TimeFormat), ctx.TableHeader, contents)
}
