package cloudflare

import (
	//"bufio"
	"fmt"
	"net/url"
	"time"

	"github.com/elastic/beats/libbeat/logp"
	"github.com/franela/goreq"
)

/**
	View details of API calls here: https://support.cloudflare.com/hc/en-us/articles/216672448-Enterprise-Log-Share-REST-API
**/

const (
	API_BASE = "https://api.cloudflare.com"
)

type CloudflareClient struct {
	ApiKey         string
	Email          string
	UserServiceKey string
	methods        map[string][]string
	debug          bool
	RequestLogFile *RequestLogFile
	counter        int64
	LogfileName    string
}

func NewClient(params map[string]interface{}) *CloudflareClient {

	c := &CloudflareClient{
		methods: map[string][]string{
			// Download logs starting from a specific timestamp
			// /client/v4/zones/:zone_tag/logs/requests?start=<unix_ts>[&end=<unix_ts>&count=<number>]
			"get_range_from_timestamp": {"GET", "/client/v4/zones/%s/logs/requests"},
		},
		counter: 0,
	}

	if _, ok := params["api_key"]; ok {
		c.ApiKey = params["api_key"].(string)
		c.Email = params["email"].(string)
	} else {
		c.UserServiceKey = params["user_service_key"].(string)
	}

	if _, ok := params["debug"]; ok {
		c.debug = params["debug"].(bool)
	}

	return c
}

func (c *CloudflareClient) doRequest(actionType string, params map[string]interface{}) error {

	qsa := url.Values{}
	url := API_BASE + fmt.Sprintf(c.methods[actionType][1], params["zone_tag"].(string))

	if actionType == "get_range_from_timestamp" {

		if _, ok := params["time_start"]; ok {
			qsa.Set("start", fmt.Sprintf("%d", params["time_start"].(int)))
		}
		if _, ok := params["time_end"]; ok {
			qsa.Set("end", fmt.Sprintf("%d", params["time_end"].(int)))
		}
		if _, ok := params["count"]; ok {
			qsa.Set("count", fmt.Sprintf("%d", params["count"].(int)))
		}
	}

	req := goreq.Request{
		Uri:         url,
		Timeout:     10 * time.Minute,
		ShowDebug:   c.debug,
		QueryString: qsa,
	}

	req.AddHeader("Accept-encoding", "gzip")
	if c.UserServiceKey != "" {
		req.AddHeader("X-User-Service-Key", c.UserServiceKey)
	} else {
		req.AddHeader("X-Auth-Key", c.ApiKey)
		req.AddHeader("X-Auth-Email", c.Email)
	}

	logp.Info("Downloading log file...")

	response, err := req.Do()
	if err != nil {
		return err
	}

	// Now need to save all the resposne content to a file
	c.LogfileName = fmt.Sprintf("cloudflare_logs_%d_to_%d.txt.gz", params["time_start"].(int), params["time_end"].(int))
	rlf := NewRequestLogFile(c.LogfileName)
	c.RequestLogFile = rlf
	nBytes, err := c.RequestLogFile.SaveFromHttpResponseBody(response.Body)
	if err != nil {
		return err
	}
	logp.Info("Downloaded %d bytes", nBytes)

	return nil
}

/*
func (c *CloudflareClient) GetLogRangeFromRayId(zoneTag string, rayId string) {
	response, respBody, errs := c.doRequest("get_single", map[string]interface{}{"zone_tag": zoneTag, "rayid": rayId})
}
*/

/*
func (c *CloudflareClient) GetLogRangeFromTimestamp(opts map[string]interface{}) ([]common.MapStr, error) {
	//var logs []common.MapStr
	return c.doRequest("get_range_from_timestamp", opts)
}
*/

func (c *CloudflareClient) GetLogRangeFromTimestamp(opts map[string]interface{}) error {
	err := c.doRequest("get_range_from_timestamp", opts)
	if err != nil {
		return err
	}
	return nil
}
