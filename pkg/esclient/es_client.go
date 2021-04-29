package esclient

import (
	"fmt"
	"sync/atomic"

	"github.com/elastic/go-elasticsearch/v7"
	"github.com/linkingthing/clxone-dhcp/config"
	"github.com/linkingthing/clxone-dhcp/pkg/util/httpclient"
)

const (
	ESQueryUrl      = "http://%s/%s/_search"
	SrcIpKeyWord    = "src_ip.keyword"
	DomainKeyWord   = "domain.keyword"
	DestIpKeyWord   = "dest_ip.keyword"
	TimestampFormat = "yyyy-MM-dd HH:mm"
	DateFormat      = "yyyy-MM-dd"
)

var AggsTermOrder = map[string]string{"_count": "desc"}

var (
	gESClient *ESClient
	esClient  *elasticsearch.Client
)

type ESClient struct {
	addrs       []string
	activeIndex uint32
}

func Init(conf *config.DDIControllerConfig) {
	gESClient = &ESClient{
		addrs: conf.Elasticsearch.Addr,
	}

	newConfig := make([]string, len(conf.Elasticsearch.Addr))
	for i, addr := range conf.Elasticsearch.Addr {
		newConfig[i] = "http://" + addr
	}

	es, err := elasticsearch.NewClient(elasticsearch.Config{
		Addresses: newConfig,
	})
	if err != nil {
		panic(err)
	}
	esClient = es
}

func GetESClient() *ESClient {
	return gESClient
}

func GetNewESClient() *elasticsearch.Client {
	return esClient
}

func GetESClient2(conf *config.DDIControllerConfig) {
}

type ElasticsearchRequest struct {
	Size  uint32                       `json:"size"`
	Sort  []ElasticsearchSort          `json:"sort,omitempty"`
	Query ElasticsearchQuery           `json:"query,omitempty"`
	Aggs  map[string]ElasticsearchAggs `json:"aggs,omitempty"`
}

type ElasticsearchSort struct {
	Timestamp ElasticsearchSortTimestamp `json:"@timestamp,omitempty"`
}

type ElasticsearchSortTimestamp struct {
	Order string `json:"order,omitempty"`
}

type ElasticsearchQuery struct {
	Bool ElasticsearchBool `json:"bool,omitempty"`
}

type ElasticsearchBool struct {
	Must []interface{} `json:"must,omitempty"`
}

type ElasticsearchMustMatch struct {
	Match map[string]string `json:"match,omitempty"`
}

type ElasticsearchMustRange struct {
	Range ElasticsearchRange `json:"range,omitempty"`
}

type ElasticsearchRange struct {
	Timestamp ElasticsearchRangeTimestamp `json:"@timestamp,omitempty"`
}

type ElasticsearchRangeTimestamp struct {
	From   string `json:"from,omitempty"`
	GTE    string `json:"gte,omitempty"`
	LTE    string `json:"lte,omitempty"`
	Format string `json:"format,omitempty"`
}

type ElasticsearchAggs struct {
	Term ElasticsearchAggsTerm `json:"terms,omitempty"`
}

type ElasticsearchAggsTerm struct {
	Field string            `json:"field,omitempty"`
	Order map[string]string `json:"order,omitempty"`
	Size  uint32            `json:"size,omitempty"`
}

type ElasticsearchResponse struct {
	Timeout      bool                   `json:"time_out,omitempty"`
	Hits         ElasticsearchHistory   `json:"hits"`
	Aggregations map[string]Aggregation `json:"aggregations,omitempty"`
}

type ElasticsearchHistory struct {
	Hits []ElasticsearchHistoryData `json:"hits"`
}

type ElasticsearchHistoryData struct {
	Source ElasticsearchHistorySource `json:"_source"`
}

type ElasticsearchHistorySource struct {
	TimeStamp   string `json:"@timestamp"`
	DestIP      string `json:"dest_ip"`
	Domain      string `json:"domain"`
	SourceIP    string `json:"src_ip"`
	SourcePort  string `json:"src_port"`
	ResolveType string `json:"type"`
	View        string `json:"view"`
}

type Aggregation struct {
	Buckets []Bucket `json:"buckets,omitempty"`
}

type Bucket struct {
	Key      string `json:"key,omitempty"`
	DocCount uint64 `json:"doc_count,omitempty"`
}

func (cli *ESClient) Request(req *ElasticsearchRequest, queryIndex string) (*ElasticsearchResponse, error) {
	var resp ElasticsearchResponse
	activeIndex := atomic.LoadUint32(&cli.activeIndex)
	addrsCount := len(cli.addrs)
	err := httpclient.GetHttpClient().Post(fmt.Sprintf(ESQueryUrl, cli.addrs[activeIndex], queryIndex), req, &resp)
	for retryTimes := 0; err != nil && retryTimes < addrsCount-1; retryTimes++ {
		activeIndex = (activeIndex + 1) % uint32(addrsCount)
		err = httpclient.GetHttpClient().Post(fmt.Sprintf(ESQueryUrl, cli.addrs[activeIndex], queryIndex), req, &resp)
	}

	if err != nil {
		return nil, err
	}

	atomic.StoreUint32(&cli.activeIndex, activeIndex)
	if resp.Timeout {
		return nil, fmt.Errorf("get %s from elasticsearch timeout", queryIndex)
	}

	return &resp, nil
}
