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
	RequestLogFile *RequestLogFile
	LogfileName    string
	uri            string
	debug          bool
}

// NewClient returns a new instance of a CloudflareClient struct
func NewClient(params map[string]interface{}) *CloudflareClient {

	c := &CloudflareClient{
		uri: "/client/v4/zones/%s/logs/requests",
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

func (c *CloudflareClient) doRequest(params map[string]interface{}) (string, error) {

	qsa := url.Values{}
	apiURL := API_BASE + fmt.Sprintf(c.uri, params["zone_tag"].(string))

	if _, ok := params["time_start"]; ok {
		qsa.Set("start", fmt.Sprintf("%d", params["time_start"].(int)))
	}
	if _, ok := params["time_end"]; ok {
		qsa.Set("end", fmt.Sprintf("%d", params["time_end"].(int)))
	}
	if _, ok := params["count"]; ok {
		qsa.Set("count", fmt.Sprintf("%d", params["count"].(int)))
	}

	req := goreq.Request{
		Uri:         apiURL,
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

	logp.Debug("http", "Downloading log file...")

	response, err := req.Do()
	if err != nil {
		return "", err
	}

	// Now need to save all the resposne content to a file
	logFileName := fmt.Sprintf("cloudflare_logs_%d_to_%d.txt.gz", params["time_start"].(int), params["time_end"].(int))
	rlf := NewRequestLogFile(logFileName)
	nBytes, err := rlf.SaveFromHttpResponseBody(response.Body)
	if err != nil {
		return "", err
	}

	logp.Debug("http", "Downloaded %d bytes", nBytes)

	return logFileName, nil
}

func (c *CloudflareClient) GetLogRangeFromTimestamp(opts map[string]interface{}) (string, error) {
	filename, err := c.doRequest(opts)
	if err != nil {
		return "", err
	}
	return filename, nil
}
