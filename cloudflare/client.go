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

func (c *CloudflareClient) doRequest(actionType string, params map[string]interface{}) ([]common.MapStr, error) {

	var logs []common.MapStr

	qsa := url.Values{}
	url := API_BASE + fmt.Sprintf(c.methods[actionType][1], params["zone_tag"].(string))

	if actionType == "get_range_from_ray_id" {

		if _, ok := params["ray_id"]; ok {
			qsa.Set("start_id", params["ray_id"].(string))
		}
		if _, ok := params["time_end"]; ok {
			qsa.Set("end", fmt.Sprintf("%d", params["time_end"].(int)))
		}
		if _, ok := params["count"]; ok {
			qsa.Set("count", fmt.Sprintf("%d", params["count"].(int)))
		}

	} else if actionType == "get_range_from_timestamp" {

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

	response, err := req.Do()
	if err != nil {
		return logs, err
	}

	responseBodyString, err := response.Body.ToString()

	if err != nil {
		return logs, err
	} else if responseBodyString == "" {
		return logs, errors.New("Request body is empty")
	}

	// Split the response body into individual lines
	responseLines := strings.Split(responseBodyString, "\n")

	//var ts float64
	var l common.MapStr

	for _, logItem := range responseLines {

		if strings.TrimSpace(logItem) == "" {
			continue
		}

		err := ffjson.Unmarshal([]byte(logItem), &l)

		if err == nil {
			c.counter++
			ts, err := l.GetValue("timestamp")
			if err == nil {
				l["timestamp"] = int64(ts.(float64)) / 1000000
			}
			l["@timestamp"] = common.Time(time.Unix(0, int64(ts.(float64))).UTC())
			l["type"] = "cloudflare"
			l["counter"] = c.counter

			/**********************************************************************************
				Now fix all the nanosecond timestamps and convert them to millisecond timestamps
				as Elasticsearch doesn't support nanoseconds
			**********************************************************************************/
			edge, err := l.GetValue("edge")
			if err == nil {
				ns := edge.(map[string]interface{})["startTimestamp"].(float64)
				l["edge"].(map[string]interface{})["startTimestamp"] = int64(ns) / 1000000
				ns = edge.(map[string]interface{})["endTimestamp"].(float64)
				l["edge"].(map[string]interface{})["endTimestamp"] = int64(ns) / 1000000
			}
			cache, err := l.GetValue("cache")
			if err == nil {
				ns := cache.(map[string]interface{})["startTimestamp"].(float64)
				l["cache"].(map[string]interface{})["startTimestamp"] = int64(ns) / 1000000
				ns = cache.(map[string]interface{})["endTimestamp"].(float64)
				l["cache"].(map[string]interface{})["endTimestamp"] = int64(ns) / 1000000
			}
			waf, err := l.GetValue("waf")
			if err == nil {
				ns := waf.(map[string]interface{})["timestampStart"].(float64)
				l["waf"].(map[string]interface{})["timestampStart"] = int64(ns) / 1000000
				ns = waf.(map[string]interface{})["timestampEnd"].(float64)
				l["waf"].(map[string]interface{})["timestampEnd"] = int64(ns) / 1000000
			}
			or, err := l.GetValue("originResponse")
			if err == nil {
				ns := or.(map[string]interface{})["httpExpires"].(float64)
				l["originResponse"].(map[string]interface{})["httpExpires"] = int64(ns) / 1000000

			}

			logs = append(logs, l)
		} else {
			logp.Err("Could not load JSON: %s", err)
		}

	} // END of range responseLines

	return logs, nil
}

/*
func (c *CloudflareClient) GetLogRangeFromRayId(zoneTag string, rayId string) {
	response, respBody, errs := c.doRequest("get_single", map[string]interface{}{"zone_tag": zoneTag, "rayid": rayId})
}
*/

func (c *CloudflareClient) GetLogRangeFromTimestamp(opts map[string]interface{}) ([]common.MapStr, error) {

	//var logs []common.MapStr
	return c.doRequest("get_range_from_timestamp", opts)

}
