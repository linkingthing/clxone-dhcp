package httpclient

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"time"
)

var gHttpClient *HttpClient

func init() {
	gHttpClient = &HttpClient{&http.Client{
		Timeout:   30 * time.Second,
		Transport: &http.Transport{DisableKeepAlives: true},
	}}
}

type HttpClient struct {
	*http.Client
}

func GetHttpClient() *HttpClient {
	return gHttpClient
}

func (cli *HttpClient) Post(url string, req, resp interface{}) error {
	return cli.request(http.MethodPost, url, req, resp)
}

func (cli *HttpClient) Get(url string, resp interface{}) error {
	return cli.request(http.MethodGet, url, nil, resp)
}

func (cli *HttpClient) request(httpMethod, url string, req, resp interface{}) error {
	var httpReqBody io.Reader
	if req != nil {
		reqBody, err := json.Marshal(req)
		if err != nil {
			return fmt.Errorf("marshal request failed: %s", err.Error())
		}

		httpReqBody = bytes.NewBuffer(reqBody)
	}

	httpReq, err := http.NewRequest(httpMethod, url, httpReqBody)
	if err != nil {
		return fmt.Errorf("new http request failed: %s", err.Error())
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpResp, err := cli.Do(httpReq)
	if err != nil {
		return fmt.Errorf("send http request failed: %s", err.Error())
	}

	defer httpResp.Body.Close()
	body, err := ioutil.ReadAll(httpResp.Body)
	if err != nil {
		return fmt.Errorf("read http response body failed: %s", err.Error())
	}

	if err := json.Unmarshal(body, &resp); err != nil {
		return fmt.Errorf("unmarshal http response failed: %s", err.Error())
	}

	return nil
}
