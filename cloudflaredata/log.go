package cloudflaredata

import (
	"encoding/json"
)

type Log struct {
	BrandID        *int            `json:"brandId"`
	Cache          *Cache          `json:"cache"`
	CacheRequest   *CacheRequest   `json:"cacheRequest"`
	CacheResponse  *CacheResponse  `json:"cacheResponse"`
	Client         *Client         `json:"client"`
	ClientRequest  *ClientRequest  `json:"clientRequest"`
	Edge           *Edge           `json:"edge"`
	EdgeRequest    *EdgeRequest    `json:"edgeRequest"`
	EdgeResponse   *EdgeResponse   `json:"edgeResponse"`
	Flags          *int            `json:"flags"`
	HosterID       *int            `json:"hosterId"`
	Origin         *Origin         `json:"origin"`
	OriginResponse *OriginResponse `json:"originResponse"`
	OwnerID        *int            `json:"ownerId"`
	RayID          *string         `json:"rayId"`
	SecurityLevel  *string         `json:"securityLevel"`
	Timestamp      *int64          `json:"timestamp"`
	ZoneID         *int            `json:"zoneId"`
	ZoneName       *string         `json:"zoneName"`
	ZonePlan       *string         `json:"zonePlan"`
}

type Cache struct {
	StartTimestamp    *int64  `json:"startTimestamp"`
	EndTimestamp      *int64  `json:"endTimestamp"`
	CacheServerName   *string `json:"cacheServerName"`
	CacheFileKey      *string `json:"cacheFileKey"`
	BckType           *string `json:"bckType"`
	CacheStatus       *string `json:"cacheStatus"`
	CacheInternalIP   *string `json:"cacheInternalIp"`
	CacheExternalIP   *string `json:"cacheExternalIp"`
	CacheExternalPort *int    `json:"cacheExternalPort"`
}

type CacheRequest struct {
	KeepaliveStatus *string `json:"keepaliveStatus"`
}

type CacheResponse struct {
	Status      *int    `json:"status"`
	Bytes       *uint64 `json:"bytes"`
	BodyBytes   *uint64 `json:"bodyBytes"`
	ContentType *string `json:"contentType"`
}

type Client struct {
	IP          *string `json:"ip"`
	SrcPort     *int    `json:"srcPort"`
	IPClass     *string `json:"ipClass"`
	Country     *string `json:"country"`
	SslProtocol *string `json:"sslProtocol"`
	SslCipher   *string `json:"sslCipher"`
	SslFlags    *int    `json:"sslFlags"`
	DeviceType  *string `json:"deviceType"`
	AsNum       *int    `json:"asNum"`
}

type ClientRequest struct {
	Bytes        *int64           `json:"bytes"`
	BodyBytes    *int64           `json:"bodyBytes"`
	HTTPHost     *string          `json:"httpHost"`
	HTTPMethod   *string          `json:"httpMethod"`
	URI          *string          `json:"uri"`
	HTTPProtocol *string          `json:"httpProtocol"`
	UserAgent    *string          `json:"userAgent"`
	Flags        *int             `json:"flags"`
	Headers      *json.RawMessage `json:"headers"` // *[]string
	Cookies      *json.RawMessage `json:"cookies"` // *[]string
}

type Edge struct {
	StartTimestamp    *int64  `json:"startTimestamp"`
	EndTimestamp      *int64  `json:"endTimestamp"`
	Color             *int    `json:"color"`
	FlServerIP        *string `json:"flServerIp"`
	FlServerPort      *int    `json:"flServerPort"`
	FlServerName      *string `json:"flServerName"`
	EnabledFlags      *int    `json:"enabledFlags"`
	UsedFlags         *int    `json:"usedFlags"`
	PathingOp         *string `json:"pathingOp"`
	PathingSrc        *string `json:"pathingSrc"`
	PathingStatus     *string `json:"pathingStatus"`
	BbResult          *string `json:"bbResult"`
	CacheResponseTime *int64  `json:"cacheResponseTime"`
	Waf               *Waf    `json:"waf"`
}

type Waf struct {
	TimestampStart    *int64           `json:"timestampStart"`
	TimestampEnd      *int64           `json:"timestampEnd"`
	Profile           *string          `json:"profile"`
	RuleID            *string          `json:"ruleId"`
	RuleMessage       *string          `json:"ruleMessage"`
	Action            *string          `json:"action"`
	RuleDetail        *json.RawMessage `json:"ruleDetail"` // *[]string
	MatchedVar        *string          `json:"matchedVar"`
	ActivatedRules    *json.RawMessage `json:"activatedRules"` // *[]string
	RuleGroup         *string          `json:"ruleGroup"`
	ExitCode          *int             `json:"exitCode"`
	XSSScore          *int             `json:"xssScore"`
	SQLInjectionScore *int             `json:"sqlInjectionScore"`
	AnomalyScore      *int             `json:"anomalyScore"`
	Tags              *json.RawMessage `json:"tags"` // *[]string
	Flags             *int             `json:"flags"`
}

type EdgeRequest struct {
	Bytes           *int64  `json:"bytes"`
	BodyBytes       *int64  `json:"bodyBytes"`
	HTTPHost        *string `json:"httpHost"`
	HTTPMethod      *string `json:"httpMethod"`
	URI             *string `json:"uri"`
	KeepaliveStatus *string `json:"keepaliveStatus"`
}

type EdgeResponse struct {
	Status           *int             `json:"status"`
	Bytes            *int64           `json:"bytes"`
	BodyBytes        *int64           `json:"bodyBytes"`
	CompressionRatio *float64         `json:"compressionRatio"`
	Headers          *json.RawMessage `json:"headers"` // *[]string
	ContentType      *string          `json:"contentType"`
	SetCookies       *json.RawMessage `json:"setCookiesc"` // *[]string
}

type Origin struct {
	IP              *string `json:"ip"`
	Port            *int    `json:"port"`
	SslProtocol     *string `json:"sslProtocol"`
	SslCipher       *string `json:"sslCipher"`
	CfRailgun       *string `json:"cfRailgun"`
	RailgunWanError *string `json:"railgunWanError"`
	ResponseTime    *int64  `json:"responseTime"`
}

type OriginResponse struct {
	Status           *int             `json:"status"`
	Bytes            *int64           `json:"bytes"`
	BodyBytes        *int64           `json:"bodyBytes"`
	HTTPLastModified *int64           `json:"httpLastModified"`
	HTTPExpires      *int64           `json:"httpExpires"`
	Flags            *int             `json:"flags"`
	Headers          *json.RawMessage `json:"headers"` // *[]string
}

/*
//  UnmarshalJSON is the custom decoding function to expand the data into the custom Cache struct
func (s *Cache) UnmarshalJSON(data []byte) error {
	var i interface{}
	if err := json.Unmarshal(data, &i); err != nil {
		return err
	}
	m := i.(map[string]interface{})

	s.StartTimestamp =
		s.EndTimestamp
	s.CacheServerName
	s.CacheFileKey
	s.BckType
	s.CacheStatus
	s.CacheInternalIP
	s.CacheExternalIP
	s.CacheExternalPort
	return nil
}

//  UnmarshalJSON is the custom decoding function to expand the data into the custom CacheRequest struct
func (s *CacheRequest) UnmarshalJSON(data []byte) error {
	var v [2]float64
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	l.Price = v[0]
	o.Volume = v[1]
	return nil
}

//  UnmarshalJSON is the custom decoding function to expand the data into the custom CacheResponse struct
func (s *CacheResponse) UnmarshalJSON(data []byte) error {
	var v [2]float64
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	l.Price = v[0]
	o.Volume = v[1]
	return nil
}

//  UnmarshalJSON is the custom decoding function to expand the data into the custom Client struct
func (s *Client) UnmarshalJSON(data []byte) error {
	var v [2]float64
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	l.Price = v[0]
	o.Volume = v[1]
	return nil
}

//  UnmarshalJSON is the custom decoding function to expand the data into the custom ClientRequest struct
func (s *ClientRequest) UnmarshalJSON(data []byte) error {
	var v [2]float64
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	l.Price = v[0]
	o.Volume = v[1]
	return nil
}

//  UnmarshalJSON is the custom decoding function to expand the data into the custom Edge struct
func (s *Edge) UnmarshalJSON(data []byte) error {
	var v [2]float64
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	l.Price = v[0]
	o.Volume = v[1]
	return nil
}

// UnmarshalJSON is the custom decoding function to expand the data into the custom EdgeRequest struct
func (s *EdgeRequest) UnmarshalJSON(data []byte) error {
	var v [2]float64
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	l.Price = v[0]
	o.Volume = v[1]
	return nil
}

// UnmarshalJSON is the custom decoding function to expand the data into the custom EdgeResponse struct
func (s *EdgeResponse) UnmarshalJSON(data []byte) error {
	var v [2]float64
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	l.Price = v[0]
	o.Volume = v[1]
	return nil
}

// UnmarshalJSON is the custom decoding function to expand the data into the custom Origin struct
func (s *Origin) UnmarshalJSON(data []byte) error {
	var v [2]float64
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	l.Price = v[0]
	o.Volume = v[1]
	return nil
}

// UnmarshalJSON is the custom decoding function to expand the data into the custom OriginResponse struct
func (s *OriginResponse) UnmarshalJSON(data []byte) error {
	var v [2]float64
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	l.Price = v[0]
	o.Volume = v[1]
	return nil
}

*/
