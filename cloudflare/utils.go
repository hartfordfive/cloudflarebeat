package cloudflare

import (
	"math"
	"reflect"
	"time"

	"github.com/elastic/beats/libbeat/common"
)

// BuildMapStr creates a valid common.MapStr struct only containing non-empty fields
func BuildMapStr(logEntry map[string]interface{}) common.MapStr {
	entry := common.MapStr{}

	// Now convert the nanosecond timestamps to millisecond timestamps
	switch logEntry["timestamp"].(type) {
	case int64:
		entry["timestamp"] = logEntry["timestamp"].(int64) / int64(time.Millisecond)
	case float64:
		entry["timestamp"] = int64(logEntry["timestamp"].(float64)) / int64(time.Millisecond)
	}

	entry["brandId"] = logEntry["brandId"]

	// ------------------ cache sub object ---------------------
	if _, ok := logEntry["cache"]; ok {
		entry["cache"] = map[string]interface{}{}
		entry["cache"].(map[string]interface{})["bckType"] = logEntry["cache"].(map[string]interface{})["bckType"]
		if !isZero(logEntry["cache"].(map[string]interface{})["cacheExternalIp"]) {
			entry["cache"].(map[string]interface{})["cacheExternalIp"] = logEntry["cache"].(map[string]interface{})["cacheExternalIp"]
		}
		entry["cache"].(map[string]interface{})["externalPort"] = logEntry["cache"].(map[string]interface{})["externalPort"]
		if !isZero(logEntry["cache"].(map[string]interface{})["cacheInternalIp"]) {
			entry["cache"].(map[string]interface{})["cacheInternalIp"] = logEntry["cache"].(map[string]interface{})["cacheInternalIp"]
		}

		entry["cache"].(map[string]interface{})["cacheServerName"] = logEntry["cache"].(map[string]interface{})["cacheServerName"]
		entry["cache"].(map[string]interface{})["cacheStatus"] = logEntry["cache"].(map[string]interface{})["cacheStatus"]
		entry["cache"].(map[string]interface{})["cacheFileKey"] = logEntry["cache"].(map[string]interface{})["cacheFileKey"]

		// Now convert the nanosecond timestamps to millisecond timestamps
		switch logEntry["cache"].(map[string]interface{})["startTimestamp"].(type) {
		case int64:
			entry["cache"].(map[string]interface{})["startTimestamp"] = logEntry["cache"].(map[string]interface{})["startTimestamp"].(int64) / int64(time.Millisecond)
			//entry["cache"].(map[string]interface{})["startTimestamp"] = common.Time(time.Unix(0, logEntry["cache"].(map[string]interface{})["startTimestamp"].(int64)))
		case float64:
			entry["cache"].(map[string]interface{})["startTimestamp"] = int64(logEntry["cache"].(map[string]interface{})["startTimestamp"].(float64)) / int64(time.Millisecond)
			//entry["cache"].(map[string]interface{})["startTimestamp"] = common.Time(time.Unix(0, int64(logEntry["cache"].(map[string]interface{})["startTimestamp"].(float64))))
		}

		switch logEntry["cache"].(map[string]interface{})["endTimestamp"].(type) {
		case int64:
			entry["cache"].(map[string]interface{})["endTimestamp"] = logEntry["cache"].(map[string]interface{})["endTimestamp"].(int64) / int64(time.Millisecond)
			//entry["cache"].(map[string]interface{})["endTimestamp"] = common.Time(time.Unix(0, logEntry["cache"].(map[string]interface{})["endTimestamp"].(int64)))
		case float64:
			entry["cache"].(map[string]interface{})["endTimestamp"] = int64(logEntry["cache"].(map[string]interface{})["endTimestamp"].(float64)) / int64(time.Millisecond)
			//entry["cache"].(map[string]interface{})["endTimestamp"] = common.Time(time.Unix(0, int64(logEntry["cache"].(map[string]interface{})["endTimestamp"].(float64))))
		}
	}

	// ------------------ cacheRequest sub object ---------------------
	if _, ok := logEntry["cacheRequest"]; ok {
		entry["cacheRequest"] = map[string]interface{}{}
		if !isZero(logEntry["cacheRequest"].(map[string]interface{})["headers"]) {
			entry["cache"].(map[string]interface{})["headers"] = logEntry["cacheRequest"].(map[string]interface{})["headers"]
		}
		entry["cacheRequest"].(map[string]interface{})["keepaliveStatus"] = logEntry["cacheRequest"].(map[string]interface{})["keepaliveStatus"]
	}

	// ------------------ cacheResponse sub object ---------------------
	if _, ok := logEntry["cacheResponse"]; ok {
		entry["cacheResponse"] = map[string]interface{}{}
		entry["cacheResponse"].(map[string]interface{})["bodyBytes"] = logEntry["cacheResponse"].(map[string]interface{})["bodyBytes"]
		entry["cacheResponse"].(map[string]interface{})["bytes"] = logEntry["cacheResponse"].(map[string]interface{})["bytes"]
		if !isZero(logEntry["cacheResponse"].(map[string]interface{})["contentType"]) {
			entry["cacheResponse"].(map[string]interface{})["contentType"] = logEntry["cacheResponse"].(map[string]interface{})["contentType"]
		}
		entry["cacheResponse"].(map[string]interface{})["retriedStatus"] = logEntry["cacheResponse"].(map[string]interface{})["retriedStatus"]
		entry["cacheResponse"].(map[string]interface{})["status"] = logEntry["cacheResponse"].(map[string]interface{})["status"]
	}

	// ------------------ client sub object ---------------------
	if _, ok := logEntry["client"]; ok {
		entry["client"] = map[string]interface{}{}
		entry["client"].(map[string]interface{})["asNum"] = logEntry["client"].(map[string]interface{})["asNum"]
		entry["client"].(map[string]interface{})["country"] = logEntry["client"].(map[string]interface{})["country"]
		entry["client"].(map[string]interface{})["deviceType"] = logEntry["client"].(map[string]interface{})["deviceType"]
		if !isZero(logEntry["client"].(map[string]interface{})["ip"]) {
			entry["client"].(map[string]interface{})["ip"] = logEntry["client"].(map[string]interface{})["ip"]
		}
		entry["client"].(map[string]interface{})["ipClass"] = logEntry["client"].(map[string]interface{})["ipClass"]
		entry["client"].(map[string]interface{})["srcPort"] = logEntry["client"].(map[string]interface{})["srcPort"]
		entry["client"].(map[string]interface{})["sslCipher"] = logEntry["client"].(map[string]interface{})["sslCipher"]
		entry["client"].(map[string]interface{})["sslFlags"] = logEntry["client"].(map[string]interface{})["sslFlags"]
		entry["client"].(map[string]interface{})["sslProtocol"] = logEntry["client"].(map[string]interface{})["sslProtocol"]
	}

	// ------------------ clientRequest sub object ---------------------
	if _, ok := logEntry["clientRequest"]; ok {
		entry["clientRequest"] = map[string]interface{}{}
		entry["clientRequest"].(map[string]interface{})["accept"] = logEntry["clientRequest"].(map[string]interface{})["accept"]
		entry["clientRequest"].(map[string]interface{})["bodyBytes"] = logEntry["clientRequest"].(map[string]interface{})["bodyBytes"]
		entry["clientRequest"].(map[string]interface{})["bytes"] = logEntry["clientRequest"].(map[string]interface{})["bytes"]
		entry["clientRequest"].(map[string]interface{})["cookies"] = logEntry["clientRequest"].(map[string]interface{})["cookies"]
		entry["clientRequest"].(map[string]interface{})["flags"] = logEntry["clientRequest"].(map[string]interface{})["flags"]
		if !isZero(logEntry["clientRequest"].(map[string]interface{})["headers"]) {
			entry["clientRequest"].(map[string]interface{})["headers"] = logEntry["clientRequest"].(map[string]interface{})["headers"]
		}
		entry["clientRequest"].(map[string]interface{})["httpHost"] = logEntry["clientRequest"].(map[string]interface{})["httpHost"]
		entry["clientRequest"].(map[string]interface{})["uri"] = logEntry["clientRequest"].(map[string]interface{})["uri"]
		entry["clientRequest"].(map[string]interface{})["referer"] = logEntry["clientRequest"].(map[string]interface{})["referer"]
		entry["clientRequest"].(map[string]interface{})["userAgent"] = logEntry["clientRequest"].(map[string]interface{})["userAgent"]
	}

	// ------------------ edge sub object ---------------------
	if _, ok := logEntry["edge"]; ok {
		entry["edge"] = map[string]interface{}{}
		entry["edge"].(map[string]interface{})["bbResult"] = logEntry["edge"].(map[string]interface{})["bbResult"]
		entry["edge"].(map[string]interface{})["cacheResponseTime"] = logEntry["edge"].(map[string]interface{})["cacheResponseTime"]
		entry["edge"].(map[string]interface{})["colo"] = logEntry["edge"].(map[string]interface{})["colo"]
		entry["edge"].(map[string]interface{})["enabledFlags"] = logEntry["edge"].(map[string]interface{})["enabledFlags"]
		entry["edge"].(map[string]interface{})["flServerIp"] = logEntry["edge"].(map[string]interface{})["flServerIp"]
		entry["edge"].(map[string]interface{})["flServerName"] = logEntry["edge"].(map[string]interface{})["flServerName"]
		entry["edge"].(map[string]interface{})["flServerPort"] = logEntry["edge"].(map[string]interface{})["flServerPort"]
		entry["edge"].(map[string]interface{})["pathingOp"] = logEntry["edge"].(map[string]interface{})["pathingOp"]
		entry["edge"].(map[string]interface{})["pathingSrc"] = logEntry["edge"].(map[string]interface{})["pathingSrc"]
		entry["edge"].(map[string]interface{})["pathingStatus"] = logEntry["edge"].(map[string]interface{})["pathingStatus"]
		entry["edge"].(map[string]interface{})["rateLimitRuleId"] = logEntry["edge"].(map[string]interface{})["rateLimitRuleId"]

		// Now convert the nanosecond timestamps to millisecond timestamps

		switch logEntry["edge"].(map[string]interface{})["startTimestamp"].(type) {
		case int64:
			entry["edge"].(map[string]interface{})["startTimestamp"] = logEntry["edge"].(map[string]interface{})["startTimestamp"].(int64) / int64(time.Millisecond)
		case float64:
			entry["edge"].(map[string]interface{})["startTimestamp"] = int64(logEntry["edge"].(map[string]interface{})["startTimestamp"].(float64)) / int64(time.Millisecond)
		}
		switch logEntry["edge"].(map[string]interface{})["endTimestamp"].(type) {
		case int64:
			entry["edge"].(map[string]interface{})["endTimestamp"] = logEntry["edge"].(map[string]interface{})["endTimestamp"].(int64) / int64(time.Millisecond)
		case float64:
			entry["edge"].(map[string]interface{})["endTimestamp"] = int64(logEntry["edge"].(map[string]interface{})["endTimestamp"].(float64)) / int64(time.Millisecond)
		}

		entry["edge"].(map[string]interface{})["usedFlags"] = logEntry["edge"].(map[string]interface{})["usedFlags"]

		// Now also populate the nested waf object, if it's set
		if _, ok := logEntry["edge"].(map[string]interface{})["waf"]; ok {
			entry["edge"].(map[string]interface{})["waf"] = map[string]interface{}{}

			// Now convert the nanosecond timestamps to millisecond timestamps
			switch logEntry["edge"].(map[string]interface{})["waf"].(map[string]interface{})["startTimestamp"].(type) {
			case int64:
				entry["edge"].(map[string]interface{})["waf"].(map[string]interface{})["startTimestamp"] = logEntry["edge"].(map[string]interface{})["waf"].(map[string]interface{})["startTimestamp"].(int64) / int64(time.Millisecond)
			case float64:
				entry["edge"].(map[string]interface{})["waf"].(map[string]interface{})["startTimestamp"] = int64(logEntry["edge"].(map[string]interface{})["waf"].(map[string]interface{})["startTimestamp"].(float64)) / int64(time.Millisecond)
			}

			switch logEntry["edge"].(map[string]interface{})["waf"].(map[string]interface{})["endTimestamp"].(type) {
			case int64:
				entry["edge"].(map[string]interface{})["waf"].(map[string]interface{})["endTimestamp"] = logEntry["edge"].(map[string]interface{})["waf"].(map[string]interface{})["endTimestamp"].(int64) / int64(time.Millisecond)
			case float64:
				entry["edge"].(map[string]interface{})["waf"].(map[string]interface{})["endTimestamp"] = int64(logEntry["edge"].(map[string]interface{})["waf"].(map[string]interface{})["endTimestamp"].(float64)) / int64(time.Millisecond)
			}

			entry["edge"].(map[string]interface{})["waf"].(map[string]interface{})["profile"] = logEntry["edge"].(map[string]interface{})["waf"].(map[string]interface{})["profile"]
			entry["edge"].(map[string]interface{})["waf"].(map[string]interface{})["ruleId"] = logEntry["edge"].(map[string]interface{})["waf"].(map[string]interface{})["profile"]
			entry["edge"].(map[string]interface{})["waf"].(map[string]interface{})["ruleMessage"] = logEntry["edge"].(map[string]interface{})["waf"].(map[string]interface{})["profile"]
			if !isZero(logEntry["edge"].(map[string]interface{})["waf"].(map[string]interface{})["ruleDetail"]) {
				entry["edge"].(map[string]interface{})["waf"].(map[string]interface{})["ruleDetail"] = logEntry["edge"].(map[string]interface{})["waf"].(map[string]interface{})["ruleDetail"]
			}
			entry["waf"].(map[string]interface{})["matchedVar"] = logEntry["edge"].(map[string]interface{})["waf"].(map[string]interface{})["matchedVar"]
			if !isZero(logEntry["edge"].(map[string]interface{})["waf"].(map[string]interface{})["activatedRules"]) {
				entry["edge"].(map[string]interface{})["waf"].(map[string]interface{})["activatedRules"] = logEntry["edge"].(map[string]interface{})["waf"].(map[string]interface{})["activatedRules"]
			}
			entry["edge"].(map[string]interface{})["waf"].(map[string]interface{})["ruleGroup"] = logEntry["edge"].(map[string]interface{})["waf"].(map[string]interface{})["ruleGroup"]
			entry["edge"].(map[string]interface{})["waf"].(map[string]interface{})["exitCode"] = logEntry["edge"].(map[string]interface{})["waf"].(map[string]interface{})["exitCode"]
			entry["edge"].(map[string]interface{})["waf"].(map[string]interface{})["xssScore"] = logEntry["edge"].(map[string]interface{})["waf"].(map[string]interface{})["xssScore"]
			entry["edge"].(map[string]interface{})["waf"].(map[string]interface{})["sqlInjectionScore"] = logEntry["edge"].(map[string]interface{})["waf"].(map[string]interface{})["sqlInjectionScore"]
			entry["edge"].(map[string]interface{})["waf"].(map[string]interface{})["anomalyScore"] = logEntry["edge"].(map[string]interface{})["waf"].(map[string]interface{})["anomalyScore"]
			entry["edge"].(map[string]interface{})["waf"].(map[string]interface{})["tags"] = logEntry["edge"].(map[string]interface{})["waf"].(map[string]interface{})["tags"]
			entry["edge"].(map[string]interface{})["waf"].(map[string]interface{})["flags"] = logEntry["edge"].(map[string]interface{})["waf"].(map[string]interface{})["flags"]

		}

	}

	// ------------------ edgeRequest sub object ---------------------
	if _, ok := logEntry["edgeRequest"]; ok {
		entry["edgeRequest"] = map[string]interface{}{}
		entry["edgeRequest"].(map[string]interface{})["bodyBytes"] = logEntry["edgeRequest"].(map[string]interface{})["bodyBytes"]
		entry["edgeRequest"].(map[string]interface{})["bytes"] = logEntry["edgeRequest"].(map[string]interface{})["bytes"]
		if !isZero(logEntry["edgeRequest"].(map[string]interface{})["headers"]) {
			entry["edgeRequest"].(map[string]interface{})["headers"] = logEntry["edgeRequest"].(map[string]interface{})["headers"]
		}
		entry["edgeRequest"].(map[string]interface{})["httpHost"] = logEntry["edgeRequest"].(map[string]interface{})["httpHost"]
		entry["edgeRequest"].(map[string]interface{})["httpMethod"] = logEntry["edgeRequest"].(map[string]interface{})["httpMethod"]
		entry["edgeRequest"].(map[string]interface{})["keepaliveStatus"] = logEntry["edgeRequest"].(map[string]interface{})["keepaliveStatus"]
		entry["edgeRequest"].(map[string]interface{})["uri"] = logEntry["edgeRequest"].(map[string]interface{})["uri"]
	}

	// ------------------ edgeResponse sub object ---------------------
	if _, ok := logEntry["edgeResponse"]; ok {
		entry["edgeResponse"] = map[string]interface{}{}
		entry["edgeResponse"].(map[string]interface{})["bodyBytes"] = logEntry["edgeResponse"].(map[string]interface{})["bodyBytes"]
		entry["edgeResponse"].(map[string]interface{})["bytes"] = logEntry["edgeResponse"].(map[string]interface{})["bytes"]
		entry["edgeResponse"].(map[string]interface{})["compressionRatio"] = logEntry["edgeResponse"].(map[string]interface{})["compressionRatio"]
		entry["edgeResponse"].(map[string]interface{})["contentType"] = logEntry["edgeResponse"].(map[string]interface{})["contentType"]
		if !isZero(logEntry["edgeResponse"].(map[string]interface{})["headers"]) {
			entry["edgeResponse"].(map[string]interface{})["headers"] = logEntry["edgeResponse"].(map[string]interface{})["headers"]
		}
		entry["edgeResponse"].(map[string]interface{})["setCookies"] = logEntry["edgeResponse"].(map[string]interface{})["setCookies"]
		entry["edgeResponse"].(map[string]interface{})["status"] = logEntry["edgeResponse"].(map[string]interface{})["status"]
	}

	entry["flags"] = logEntry["flags"]
	entry["hosterId"] = logEntry["hosterId"]

	// ------------------ origin sub object ---------------------
	if _, ok := logEntry["origin"]; ok {
		entry["origin"] = map[string]interface{}{}
		entry["origin"].(map[string]interface{})["asNum"] = logEntry["origin"].(map[string]interface{})["asNum"]
		if !isZero(logEntry["origin"].(map[string]interface{})["ip"]) {
			entry["origin"].(map[string]interface{})["ip"] = logEntry["origin"].(map[string]interface{})["ip"]
		}
		entry["origin"].(map[string]interface{})["port"] = logEntry["origin"].(map[string]interface{})["port"]
		entry["origin"].(map[string]interface{})["responseTime"] = logEntry["origin"].(map[string]interface{})["responseTime"]
		entry["origin"].(map[string]interface{})["sslCipher"] = logEntry["origin"].(map[string]interface{})["sslCipher"]
		entry["origin"].(map[string]interface{})["sslProtocol"] = logEntry["origin"].(map[string]interface{})["sslProtocol"]
	}

	// ------------------ originResponse	 sub object ---------------------
	if _, ok := logEntry["originResponse"]; ok {
		entry["originResponse"] = map[string]interface{}{}
		entry["originResponse"].(map[string]interface{})["bodyBytes"] = logEntry["originResponse"].(map[string]interface{})["bodyBytes"]
		entry["originResponse"].(map[string]interface{})["bytes"] = logEntry["originResponse"].(map[string]interface{})["bytes"]
		entry["originResponse"].(map[string]interface{})["flags"] = logEntry["originResponse"].(map[string]interface{})["flags"]
		if !isZero(logEntry["originResponse"].(map[string]interface{})["headers"]) {
			entry["originResponse"].(map[string]interface{})["headers"] = logEntry["originResponse"].(map[string]interface{})["headers"]
		}
		entry["originResponse"].(map[string]interface{})["httpExpires"] = logEntry["originResponse"].(map[string]interface{})["httpExpires"]
		entry["originResponse"].(map[string]interface{})["httpLastModified"] = logEntry["originResponse"].(map[string]interface{})["httpLastModified"]
		entry["originResponse"].(map[string]interface{})["status"] = logEntry["originResponse"].(map[string]interface{})["status"]
	}

	entry["ownerId"] = logEntry["ownerId"]
	entry["rayId"] = logEntry["rayId"]
	entry["unstable"] = logEntry["unstable"]
	entry["zoneId"] = logEntry["zoneId"]
	entry["zoneName"] = logEntry["zoneName"]
	entry["zonePlan"] = logEntry["zonePlan"]

	return entry
}

// isZero return true if the given value is the zero value for its type.
func isZero(i interface{}) bool {
	v := reflect.ValueOf(i)
	switch v.Kind() {
	case reflect.Array, reflect.String:
		return v.Len() == 0
	case reflect.Bool:
		return !v.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return v.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return v.Float() == 0
	case reflect.Interface, reflect.Map, reflect.Ptr, reflect.Slice:
		return v.IsNil()
	}
	return false
}

// GetNumLogFileSegments returns the total number of log file segments to be created based on the duration
func GetNumLogFileSegments(period time.Duration) int {

	// For the time being, this should return 1 log file segment per 2 minutes
	seconds := int64(period / time.Second)
	if seconds <= 120 {
		return 1
	}

	nSegments := int(seconds / (60 * 2))

	if math.Mod(float64(seconds), float64(60*2)) != 0 {
		nSegments++
	}

	return nSegments
}

func GenerateSegmentsList(numSegments, segmentSize, timeStart, timeEnd int) []LogFileSegment {

	/* This funciton is likely not optimal.  Should eventually refactor to make it more efficient */

	// For the time being, this should return 1 log file segment per 2 minutes
	var segments []LogFileSegment
	segments = make([]LogFileSegment, 0, numSegments)

	if numSegments == 1 {
		segments = append(segments, LogFileSegment{timeStart, timeEnd})
		return segments
	}

	totalSeconds := timeEnd - timeStart
	sr := int(math.Mod(float64(totalSeconds), float64(segmentSize))) //segmet remainder
	totalFullSegments := (totalSeconds - sr) / segmentSize

	currStart := timeStart
	for i := 0; i < totalFullSegments; i++ {
		segments = append(segments, LogFileSegment{currStart, (currStart + segmentSize)})
		currStart += (segmentSize + 1)
	}

	if sr != 0 {
		segments = append(segments, LogFileSegment{currStart, (currStart + sr)})
	}

	// Update the end time of the last segment so that it accounts for number of total full segments containing exactly segmentSize seconds worth of log entries
	segments[len(segments)-1].TimeEnd = (segments[len(segments)-1].TimeEnd - len(segments)) + 1

	return segments

}
