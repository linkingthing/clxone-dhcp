package handler

import (
	"time"

	"github.com/linkingthing/clxone-dhcp/pkg/util"
)

func exportFile(ctx *MetricContext, contents [][]string) (string, error) {
	return util.WriteCSVFile(ctx.NodeIP+"-"+ctx.MetricName+"-"+time.Now().Format(util.TimeFormat),
		ctx.TableHeader, contents)
}
