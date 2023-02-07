package service

import (
	"time"

	csvutil "github.com/linkingthing/clxone-utils/csv"

	"github.com/linkingthing/clxone-dhcp/pkg/errorno"
)

func exportFile(ctx *MetricContext, contents [][]string) (string, error) {
	name, err := csvutil.WriteCSVFile(ctx.NodeIP+"-"+string(ctx.MetricName)+"-"+string(ctx.Version)+
		"-"+time.Now().Format(csvutil.TimeFormat), ctx.TableHeader, contents)
	if err != nil {
		err = errorno.ErrExport(errorno.ErrNameMetric, err.Error())
	}
	return name, err
}
