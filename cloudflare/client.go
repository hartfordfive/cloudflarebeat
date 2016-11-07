package cloudflare

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/franela/goreq"
	"github.com/pquerna/ffjson/ffjson"
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
	counter        int64
}

func NewClient(params map[string]interface{}) *CloudflareClient {

	c := &CloudflareClient{
		methods: map[string][]string{
			// Download logs starting from a RayID
			// /client/v4/zones/:zone_tag/logs/requests?start_id=<rayid>[&end=<unix_ts>&count=<number>]
			//"get_range_from_ray_id": {"GET", "/client/v4/zones/%s/logs/requests?start_id=%s&end=%d&count=%d"},
			"get_range_from_ray_id": {"GET", "/client/v4/zones/%s/logs/requests"},
			// Download logs starting from a specific timestamp
			// /client/v4/zones/:zone_tag/logs/requests?start=<unix_ts>[&end=<unix_ts>&count=<number>]
			//"get_range_from_timestamp": {"GET", "/client/v4/zones/%s/logs/requests?start=%d&end=%d"},
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

func (c *CloudflareClient) doRequest(actionType string, params map[string]interface{}) (*goreq.Response, error) {

	qsa := url.Values{}
	url := API_BASE + fmt.Sprintf(c.methods[actionType][1], params["zone_tag"].(string))

	if actionType == "get_range_from_ray_id" {

		if _, ok := params["ray_id"]; ok {
			qsa.Set("start_id", params["ray_id"].(string))
		}
		if _, ok := params["end_timestamp"]; ok {
			qsa.Set("end", params["end_timestamp"].(string))
		}
		if _, ok := params["count"]; ok {
			qsa.Set("count", params["count"].(string))
		}

	} else if actionType == "get_range_from_timestamp" {

		if _, ok := params["start_timestamp"]; ok {
			qsa.Set("start", params["start_timestamp"].(string))
		}
		if _, ok := params["end_timestamp"]; ok {
			qsa.Set("end", params["end_timestamp"].(string))
		}
		if _, ok := params["count"]; ok {
			qsa.Set("count", params["count"].(string))
		}
	}

	req := goreq.Request{
		Uri:         url,
		Timeout:     60 * time.Second,
		ShowDebug:   c.debug,
		Compression: goreq.Gzip(),
		QueryString: qsa,
	}

	req.AddHeader("Accept-encoding", "gzip")
	if c.UserServiceKey != "" {
		req.AddHeader("X-User-Service-Key", c.UserServiceKey)
	} else {
		req.AddHeader("X-Auth-Key", c.ApiKey)
		req.AddHeader("X-Auth-Email", c.Email)
	}

	res, err := req.Do()

	if err != nil {
		logp.Err("%v", err)
	}
	return res, err

}

/*
func (c *CloudflareClient) GetLogRangeFromRayId(zoneTag string, rayId string) {
	response, respBody, errs := c.doRequest("get_single", map[string]interface{}{"zone_tag": zoneTag, "rayid": rayId})
}
*/

//func (c *CloudflareClient) GetLogRangeFromTimestamp(zoneTag string, startTimestamp int, timeEnd int) ([]common.MapStr, error) {
func (c *CloudflareClient) GetLogRangeFromTimestamp(opts map[string]interface{}) ([]common.MapStr, error) {

	var logs []common.MapStr

	/*
		var options map[string]interface{}

		options["zone_tag"] = zoneTag
		options["start_timestamp"] = startTimestamp
		options["end_timestamp"] = timeEnd

		if _, ok := opts["count"]; ok {
			options["count"] = opts["count"]
		}
	*/

	response, err := c.doRequest("get_range_from_timestamp", opts)

	responseBodyString, err := response.Body.ToString()

	/*
		res.Uri // return final URL location of the response (fulfilled after redirect was made)
		res.StatusCode // return the status code of the response
		res.Body // gives you access to the body
		res.Body.ToString() // will return the body as a string
		res.Header.Get("Content-Type") //
	*/

	if err != nil {
		return logs, err
	} else if responseBodyString == "" {
		return logs, errors.New("Request body is empty")
	}

	// Split the response body into individual lines
	responseLines := strings.Split(responseBodyString, "\n")

	var ts float64
	for _, logItem := range responseLines {

		if strings.TrimSpace(logItem) == "" {
			continue
		}

		var l common.MapStr
		err := ffjson.Unmarshal([]byte(logItem), &l)

		if err == nil {
			c.counter++
			ts = l["timestamp"].(float64)
			l["@timestamp"] = common.Time(time.Unix(0, int64(ts)).UTC())
			l["type"] = "cloudflare"
			l["counter"] = c.counter
			logs = append(logs, l)
		} else {
			logp.Err("Could not load JSON: %s", err)
		}

	} // END of range responseLines

	return logs, nil
}
