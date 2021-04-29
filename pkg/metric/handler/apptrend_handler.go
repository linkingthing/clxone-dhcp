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

const appTrendQuery = `
{
    "size": 0,
    "query": {
        "bool": {
            "filter": [
                {
                    "range": {
                        "@timestamp": {
                            "gte": "%s",
                            "lte": "%s",
                            "format": "yyyy-MM-dd HH:mm"
                        }
                    }
                }
            ]
	    %s
        }
    },
    "aggs": {
        "domain_bucket": {
            "terms": {
                "field": "domain.keyword",
                "size": 5
            },
            "aggs": {
                "time_bucket": {
                    "date_histogram": {
                        "field": "@timestamp",
                        "calendar_interval": "1d",
                        "format": "yyyy-MM-dd"
                    }
                }
            }
        }
    }
}
`

const appTrendDomainFilter = `
,"must": [
    {
        "term": {
            "domain.keyword": "%s"
        }
    }
]
`

type AppTrendHandler struct{}

func NewAppTrendHandler() *AppTrendHandler {
	return &AppTrendHandler{}
}

func (h *AppTrendHandler) List(ctx *resource.Context) (interface{}, *goresterr.APIError) {
	domain, _ := util.GetFilterValueWithEqModifierFromFilters("domain", ctx.GetFilters())

	result, err := queryAppTrendFromES(domain)
	if err != nil {
		return nil, goresterr.NewAPIError(goresterr.ServerError, err.Error())
	}
	return result, nil
}

type msi = map[string]interface{}

func queryAppTrendFromES(domain string) (result []*metricresource.AppTrend, err error) {

	now := time.Now()
	begin := util.GetBeginTimeOfDate(now.AddDate(0, 0, -29), time.UTC)
	end := util.GetEndTimeOfDate(now, time.UTC)

	domainFilter := ""
	if domain != "" {
		domainFilter = fmt.Sprintf(appTrendDomainFilter, domain)
	}
	query := fmt.Sprintf(appTrendQuery,
		begin.Format(util.TimeFormatYMDHM),
		end.Format(util.TimeFormatYMDHM),
		domainFilter)

	dnsIndex, err := metricresource.GenDnsIndex()
	if err != nil {
		log.Errorf("get dns index: %v", err)
		return nil, err
	}

	esIndex := strings.Split(dnsIndex, ",")

	for _, idx := range esIndex {
		r, err := doQueryAppTrendFromES(idx, query)
		if err != nil {
			log.Errorf("query es index: %v ", err)
			continue
		}

		fillMissingDate(begin, r)
		result = mergeAppTrendResult(result, r)
	}

	sort.Slice(result, func(i, j int) bool {
		if result[i].Hits > result[j].Hits {
			return true
		} else {
			return false
		}
	})

	if len(result) > 5 {
		result = result[0:5]
	}

	return result, nil
}

func doQueryAppTrendFromES(dnsIndex string, query string) (result []*metricresource.AppTrend, err error) {
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

	for _, bucket := range r["aggregations"].(msi)["domain_bucket"].(msi)["buckets"].([]interface{}) {
		data := &metricresource.AppTrend{
			Domain: bucket.(msi)["key"].(string),
			Hits:   uint64(bucket.(msi)["doc_count"].(float64)),
		}

		for _, v := range bucket.(msi)["time_bucket"].(msi)["buckets"].([]interface{}) {
			data.Trend = append(data.Trend, &metricresource.AppTrendCell{
				Date: v.(msi)["key_as_string"].(string),
				Hits: uint64(v.(msi)["doc_count"].(float64)),
			})
		}
		result = append(result, data)
	}
	return
}

func fillMissingDate(begin time.Time, result []*metricresource.AppTrend) {
	for _, r := range result {
		trand := make([]*metricresource.AppTrendCell, 30)
		for _, t := range r.Trend {
			date, _ := time.ParseInLocation("2006-01-02", t.Date, time.UTC)
			i := int(date.Sub(begin).Hours()/24) % 30
			trand[i] = t
		}

		for i, _ := range trand {
			if trand[i] == nil {
				trand[i] = &metricresource.AppTrendCell{
					Date: begin.AddDate(0, 0, i).Format("2006-01-02"),
					Hits: 0,
				}
			}
		}
		r.Trend = trand
	}
}

func mergeAppTrendResult(r1 []*metricresource.AppTrend,
	r2 []*metricresource.AppTrend) (out []*metricresource.AppTrend) {

	for _, v1 := range r1 {
		out = append(out, v1)
	}
loop:
	for _, v2 := range r2 {
		for _, o := range out {
			if o.Domain == v2.Domain {
				o.Hits = o.Hits + v2.Hits
				o.Trend = mergeAppTrendCell(o.Trend, v2.Trend)
				continue loop
			}
		}
		out = append(out, v2)
	}
	return out
}

func mergeAppTrendCell(r1 []*metricresource.AppTrendCell,
	r2 []*metricresource.AppTrendCell) (out []*metricresource.AppTrendCell) {

	for i, _ := range r1 {
		out = append(out, &metricresource.AppTrendCell{
			Hits: r1[i].Hits + r2[i].Hits,
			Date: r1[i].Date,
		})
	}
	return out
}
