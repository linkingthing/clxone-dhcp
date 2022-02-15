package service

import (
	"time"

	csvutil "github.com/linkingthing/clxone-utils/csv"
)

func exportFile(ctx *MetricContext, contents [][]string) (string, error) {
	return csvutil.WriteCSVFile(ctx.NodeIP+"-"+string(ctx.MetricName)+"-"+string(ctx.Version)+
		"-"+time.Now().Format(csvutil.TimeFormat), ctx.TableHeader, contents)
}
