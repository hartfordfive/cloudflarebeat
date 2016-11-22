package cloudflare

import (
	"time"

	"github.com/elastic/beats/libbeat/common"
)

//**********************************************************************************
//	Now fix all the nanosecond timestamps and convert them to millisecond timestamps
//	as Elasticsearch doesn't support nanoseconds
//**********************************************************************************/

func buildMapStr() common.MapStr {

}

func FixTimestampFields(evt common.MapStr) {

	edge, err := evt.GetValue("edge")
	if err == nil {
		switch edge.(map[string]interface{})["startTimestamp"].(type) {
		case int64:
			evt["edge"].(map[string]interface{})["startTimestamp"] = edge.(map[string]interface{})["startTimestamp"].(int64) / int64(time.Millisecond)
		case float64:
			evt["edge"].(map[string]interface{})["startTimestamp"] = int64(edge.(map[string]interface{})["startTimestamp"].(float64)) / int64(time.Millisecond)
		}
		switch edge.(map[string]interface{})["endTimestamp"].(type) {
		case int64:
			evt["edge"].(map[string]interface{})["endTimestamp"] = edge.(map[string]interface{})["endTimestamp"].(int64) / int64(time.Millisecond)
		case float64:
			evt["edge"].(map[string]interface{})["endTimestamp"] = int64(edge.(map[string]interface{})["endTimestamp"].(float64)) / int64(time.Millisecond)
		}
	}
	cache, err := evt.GetValue("cache")
	if err == nil {
		switch cache.(map[string]interface{})["startTimestamp"].(type) {
		case int64:
			evt["cache"].(map[string]interface{})["startTimestamp"] = cache.(map[string]interface{})["startTimestamp"].(int64) / int64(time.Millisecond)
		case float64:
			evt["cache"].(map[string]interface{})["startTimestamp"] = int64(cache.(map[string]interface{})["startTimestamp"].(float64)) / int64(time.Millisecond)
		}
		switch cache.(map[string]interface{})["endTimestamp"].(type) {
		case int64:
			evt["cache"].(map[string]interface{})["endTimestamp"] = cache.(map[string]interface{})["endTimestamp"].(int64) / int64(time.Millisecond)
		case float64:
			evt["cache"].(map[string]interface{})["endTimestamp"] = int64(cache.(map[string]interface{})["endTimestamp"].(float64)) / int64(time.Millisecond)
		}

	}

	waf, err := evt.GetValue("waf")
	if err == nil {
		switch waf.(map[string]interface{})["timestampStart"].(type) {
		case int64:
			evt["waf"].(map[string]interface{})["timestampStart"] = waf.(map[string]interface{})["timestampStart"].(int64) / int64(time.Millisecond)
		case float64:
			evt["waf"].(map[string]interface{})["timestampStart"] = int64(waf.(map[string]interface{})["timestampStart"].(float64)) / int64(time.Millisecond)
		}
		switch waf.(map[string]interface{})["timestampEnd"].(type) {
		case int64:
			evt["waf"].(map[string]interface{})["timestampEnd"] = waf.(map[string]interface{})["timestampEnd"].(int64) / int64(time.Millisecond)
		case float64:
			evt["waf"].(map[string]interface{})["timestampEnd"] = int64(waf.(map[string]interface{})["timestampEnd"].(float64)) / int64(time.Millisecond)
		}
	}

	or, err := evt.GetValue("originResponse")
	if err == nil {
		switch or.(map[string]interface{})["httpExpires"].(type) {
		case int64:
			evt["originResponse"].(map[string]interface{})["httpExpires"] = or.(map[string]interface{})["httpExpires"].(int64) / int64(time.Millisecond)
		case float64:
			evt["originResponse"].(map[string]interface{})["httpExpires"] = int64(or.(map[string]interface{})["httpExpires"].(float64)) / int64(time.Millisecond)
		}

	}
}
