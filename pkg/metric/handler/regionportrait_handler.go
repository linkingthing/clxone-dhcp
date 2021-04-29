package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/linkingthing/clxone-dhcp/pkg/esclient"
	metricresource "github.com/linkingthing/clxone-dhcp/pkg/metric/resource"
	"github.com/linkingthing/clxone-dhcp/pkg/util"
	"github.com/zdnscloud/cement/log"
	goresterr "github.com/zdnscloud/gorest/error"
	"github.com/zdnscloud/gorest/resource"
)

const regionPortraitQuery = `
{
    "size": 0,
    "query": {
        "bool": {
            "filter": [
                {
                    "term": {
                        "countryisocode.keyword": {
                            "value": "CN"
                        }
                    }
                },
                {
                    "range": {
                        "@timestamp": {
                            "gte":    "%s",
                            "lte":    "%s",
                            "format": "%s"
                        }
                    }
                },
                {
                    "exists": {
                        "field": "provinceisocode.keyword"
                    }
                }
            ]
        }
    },
    "aggs": {
        "src_ip_count": {
            "terms": {
                "field": "provinceisocode.keyword"
            }
        }
    }
}
`

var dic = map[string]string{
	"BJ":  "北京市",
	"TJ":  "天津市",
	"HE":  "河北省",
	"SX":  "山西省",
	"NM":  "内蒙古自治区",
	"LN":  "辽宁省",
	"JL":  "吉林省",
	"HLJ": "黑龙江省",
	"SH":  "上海市",
	"JS":  "江苏省",
	"ZJ":  "浙江省",
	"AH":  "安徽省",
	"FJ":  "福建省",
	"JX":  "江西省",
	"SD":  "山东省",
	"HA":  "河南省",
	"HB":  "湖北省",
	"HN":  "湖南省",
	"GD":  "广东省",
	"GX":  "广西壮族自治区",
	"HI":  "海南省",
	"CQ":  "重庆市",
	"SC":  "四川省",
	"GZ":  "贵州省",
	"YN":  "云南省",
	"XZ":  "西藏自治区",
	"SN":  "陕西省",
	"GS":  "甘肃省",
	"QH":  "青海省",
	"NX":  "宁夏回族自治区",
	"XJ":  "新疆维吾尔自治区",
	"TW":  "台湾省",
	"HK":  "香港特别行政区",
	"MO":  "澳门特别行政区",
}

type RegionPortraitHandler struct{}

func NewRegionPortraitHandler() *RegionPortraitHandler {
	return &RegionPortraitHandler{}
}

func (h *RegionPortraitHandler) List(ctx *resource.Context) (interface{}, *goresterr.APIError) {
	now := time.Now()
	begin := util.GetBeginTimeOfDate(now.AddDate(0, 0, -29), time.UTC)
	end := util.GetEndTimeOfDate(now, time.UTC)
	query := fmt.Sprintf(regionPortraitQuery,
		begin.Format(util.TimeFormatYMDHM),
		end.Format(util.TimeFormatYMDHM),
		esclient.TimestampFormat)
	dnsIndex, err := metricresource.GenDnsIndex()
	if err != nil {
		log.Errorf("get dns index: %v ", err)
		return nil, goresterr.NewAPIError(goresterr.ServerError, err.Error())
	}

	esIndex := strings.Split(dnsIndex, ",")

	result, err := queryRegionPortraitFromES(esIndex, query)
	if err != nil {
		return nil, goresterr.NewAPIError(goresterr.ServerError, err.Error())
	}
	return result, nil
}

func queryRegionPortraitFromES(esIndex []string, query string) (result []*metricresource.RegionPortrait, err error) {
	for _, idx := range esIndex {
		r, err := doQueryRegionPortraitFromES(idx, query)
		if err != nil {
			log.Errorf("query es index: %v ", err)
			continue
		}
		result = mergeRegionPortraitResult(result, r)
	}

	sort.Slice(result, func(i, j int) bool {
		if result[i].Hits > result[j].Hits {
			return true
		} else {
			return false
		}
	})

	if len(result) > 10 {
		result = result[0:10]
	}

	return provinceCodeToNmae(result), nil
}

func doQueryRegionPortraitFromES(dnsIndex string, query string) (result []*metricresource.RegionPortrait, err error) {
	es := esclient.GetNewESClient()
	res, err := es.Search(
		es.Search.WithContext(context.Background()),
		es.Search.WithIndex(dnsIndex),
		es.Search.WithBody(bytes.NewBufferString(query)),
		es.Search.WithTrackTotalHits(true),
		es.Search.WithPretty(),
	)
	if err != nil {
		log.Errorf("es search: %v", err)
		return nil, err
	}
	defer res.Body.Close()

	if res.IsError() {
		var e map[string]interface{}
		if err := json.NewDecoder(res.Body).Decode(&e); err != nil {
			return nil, err
		} else {
			log.Errorf("[%s] %s: %s",
				res.Status(),
				e["error"].(msi)["type"],
				e["error"].(msi)["reason"],
			)
			return nil, err
		}
	}

	var r map[string]interface{}
	if err = json.NewDecoder(res.Body).Decode(&r); err != nil {
		return nil, err
	}

	for _, bucket := range r["aggregations"].(msi)["src_ip_count"].(msi)["buckets"].([]interface{}) {
		result = append(result, &metricresource.RegionPortrait{
			Province: bucket.(msi)["key"].(string),
			Hits:     uint64(bucket.(msi)["doc_count"].(float64)),
		})
	}
	return
}

func provinceCodeToNmae(result []*metricresource.RegionPortrait) []*metricresource.RegionPortrait {
	for i := range result {
		if v, ok := dic[result[i].Province]; ok {
			result[i].Province = v
		}
	}
	return result
}

func mergeRegionPortraitResult(r1 []*metricresource.RegionPortrait,
	r2 []*metricresource.RegionPortrait) (out []*metricresource.RegionPortrait) {
	for _, v1 := range r1 {
		out = append(out, v1)
	}

loop:
	for _, v2 := range r2 {
		for _, o := range out {
			if o.Province == v2.Province {
				o.Hits = o.Hits + v2.Hits
				continue loop
			}
		}
		out = append(out, v2)
	}
	return out
}
