package handler

import (
	"fmt"
	"sort"
	"strings"

	assetsource "github.com/linkingthing/clxone-dhcp/pkg/asset/resource"
	"github.com/linkingthing/clxone-dhcp/pkg/db"
	metricresource "github.com/linkingthing/clxone-dhcp/pkg/metric/resource"
	"github.com/linkingthing/clxone-dhcp/pkg/util"
	restdb "github.com/zdnscloud/gorest/db"
	goresterr "github.com/zdnscloud/gorest/error"
	"github.com/zdnscloud/gorest/resource"
)

const queryByProvinceSql = `
SELECT
	province region,
	device_state,
	COUNT ( * ) COUNT 
FROM
	gr_device 
WHERE
	province IN ( SELECT province region FROM gr_device WHERE province <> '' GROUP BY region ORDER BY COUNT ( * ) DESC LIMIT 5 ) 
GROUP BY
	province,
	device_state 
ORDER BY
COUNT DESC 
`

const queryByCitySql = `
SELECT
	city region,
	device_state,
	COUNT ( * ) COUNT 
FROM
	gr_device 
WHERE
	city IN ( SELECT city region FROM gr_device WHERE province = $1 AND city <> '' GROUP BY region ORDER BY COUNT ( * ) DESC LIMIT 5 ) 
GROUP BY
	city,
	device_state 
ORDER BY
COUNT DESC 
`

type AssetPortraitHandler struct{}

var TableDevice = restdb.ResourceDBType(&assetsource.Device{})

func NewAssetPortraitHandler() *AssetPortraitHandler {
	return &AssetPortraitHandler{}
}

func (h *AssetPortraitHandler) List(ctx *resource.Context) (interface{}, *goresterr.APIError) {
	province, _ := util.GetFilterValueWithEqModifierFromFilters("province", ctx.GetFilters())

	result := metricresource.AssetPortrait{}
	result.DeviceTotal = GetDeviceCount("", province)
	result.OnlineTotal = GetDeviceCount(assetsource.DeviceStateOnline, province)
	result.OfflineTotal = GetDeviceCount(assetsource.DeviceStateOffline, province)
	result.AbnormalTotal = GetDeviceCount(assetsource.DeviceStateAbnormal, province)

	if province == "" {
		data, err := GetTop5DeviceByProvince()
		if err != nil {
			return nil, goresterr.NewAPIError(goresterr.ServerError, err.Error())
		}
		result.StateStatistics = mergeAssetPortraitData(data)
	} else {
		data, err := GetTop5DeviceByCity(province)
		if err != nil {
			return nil, goresterr.NewAPIError(goresterr.ServerError, err.Error())
		}
		result.StateStatistics = mergeAssetPortraitData(data)
	}

	return []*metricresource.AssetPortrait{&result}, nil
}

func GetDeviceCount(state assetsource.DeviceState, provinceID string) (count int64) {
	cond := make(map[string]interface{})
	if state != "" {
		cond["device_state"] = state
	}
	if provinceID != "" {
		cond["province"] = provinceID
	}

	var err error
	var params []interface{}
	sql := "SELECT COUNT(*) FROM gr_device WHERE province <> ''"

	i := 1
	for k, v := range cond {
		sql += fmt.Sprintf(" AND %s=$%d", k, i)
		params = append(params, v)
		i++
	}

	err = restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		count, err = tx.CountEx(TableDevice, sql, params...)
		return err
	})
	if err != nil {
		return 0
	}
	return count
}

func GetTop5DeviceByProvince() (result []*metricresource.AssetPortraitCount, err error) {
	err = restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		err = tx.FillEx(&result, queryByProvinceSql)
		return err
	})
	return result, err
}

func GetTop5DeviceByCity(provinceID string) (result []*metricresource.AssetPortraitCount, err error) {
	err = restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		err = tx.FillEx(&result, queryByCitySql, provinceID)
		return err
	})

	return result, err
}

func mergeAssetPortraitData(data []*metricresource.AssetPortraitCount) (result []*metricresource.AssetStateStatistic) {
loop:
	for _, d := range data {
		for _, r := range result {
			if d.Region == r.Region {
				if d.DeviceState == assetsource.DeviceStateOnline {
					r.Online = d.Count
				}
				if d.DeviceState == assetsource.DeviceStateOffline {
					r.Offline = d.Count
				}
				if d.DeviceState == assetsource.DeviceStateAbnormal {
					r.Abnormal = d.Count
				}
				continue loop
			}
		}
		a := &metricresource.AssetStateStatistic{Region: d.Region}
		if d.DeviceState == assetsource.DeviceStateOnline {
			a.Online = d.Count
		}
		if d.DeviceState == assetsource.DeviceStateOffline {
			a.Offline = d.Count
		}
		if d.DeviceState == assetsource.DeviceStateAbnormal {
			a.Abnormal = d.Count
		}
		result = append(result, a)
	}

	sort.Slice(result, func(i, j int) bool {
		iTotal := result[i].Online + result[i].Offline + result[i].Abnormal
		jTotal := result[j].Online + result[j].Offline + result[j].Abnormal
		if iTotal == jTotal {
			return strings.Compare(result[i].Region, result[j].Region) > 0
		}
		return iTotal > jTotal
	})
	return result
}
