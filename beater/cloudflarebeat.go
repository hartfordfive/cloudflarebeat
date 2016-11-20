package beater

import (
	"bufio"
	"compress/gzip"
	"fmt"
	"os"
	"time"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/publisher"
	//"github.com/elastic/beats/libbeat/outputs/mode/lb"
	"github.com/hartfordfive/cloudflarebeat/cloudflare"
	"github.com/hartfordfive/cloudflarebeat/config"
	"github.com/pquerna/ffjson/ffjson"
)

const (
	STATEFILE_NAME = "cloudflarebeat.state"
	NUM_MINUTES    = 5
)

type Cloudflarebeat struct {
	done   chan struct{}
	config config.Config
	client publisher.Client
	state  *cloudflare.StateFile
}

// Creates beater
func New(b *beat.Beat, cfg *common.Config) (beat.Beater, error) {
	config := config.DefaultConfig
	if err := cfg.Unpack(&config); err != nil {
		return nil, fmt.Errorf("Error reading config file: %v", err)
	}

	bt := &Cloudflarebeat{
		done:   make(chan struct{}),
		config: config,
	}

	sfConf := map[string]string{
		"filename":     STATEFILE_NAME,
		"storage_type": config.StateFileStorageType,
	}

	if config.AwsAccessKey != "" && config.AwsSecretAccessKey != "" && config.AwsS3BucketName != "" {
		sfConf["aws_access_key"] = config.AwsAccessKey
		sfConf["aws_secret_access_key"] = config.AwsSecretAccessKey
		sfConf["aws_s3_bucket_name"] = config.AwsS3BucketName
	}

	sf, err := cloudflare.NewStateFile(sfConf)
	if err != nil {
		logp.Err("Statefile error: %v", err)
		return nil, err
	}

	bt.state = sf

	return bt, nil
}

func (bt *Cloudflarebeat) Run(b *beat.Beat) error {
	logp.Info("cloudflarebeat is running! Hit CTRL-C to stop it.")

	bt.client = b.Publisher.Connect()
	ticker := time.NewTicker(bt.config.Period)
	counter := 0

	// Initialize the Cloudflare client here
	cc := cloudflare.NewClient(map[string]interface{}{
		"api_key": bt.config.APIKey,
		"email":   bt.config.Email,
		"debug":   bt.config.Debug,
	})

	if bt.state.GetLastStartTS() != 0 {
		logp.Info("Start time loaded from state file: %s", time.Unix(int64(bt.state.GetLastStartTS()), 0).Format(time.RFC3339))
		logp.Info("  End time loaded from state file: %s", time.Unix(int64(bt.state.GetLastEndTS()), 0).Format(time.RFC3339))
	}

	var timeStart, timeEnd, timeNow int
	var l common.MapStr
	var logs []common.MapStr
	var logItem []byte

	var currCount int

	for {

		select {
		case <-bt.done:
			return nil
		case <-ticker.C:
		}

		timeNow = int(time.Now().UTC().Unix())

		if bt.state.GetLastStartTS() != 0 {
			timeStart = bt.state.GetLastEndTS() + 1                // last end TS as per statefile
			timeEnd = bt.state.GetLastEndTS() + (NUM_MINUTES * 60) // to 30 minutes later, minus 1 second
		} else {
			// Set the start and end time if this is the first run
			timeStart = (timeNow - (NUM_MINUTES * 60)) - (30 * 60) // specified number of minutes ago - 30 minutes
			timeEnd = timeNow - (30 * 60)                          // Up to 30 minutes ago
		}

		bt.state.UpdateLastRequestTS(timeNow)

		err := cc.GetLogRangeFromTimestamp(map[string]interface{}{
			"zone_tag":   bt.config.ZoneTag,
			"time_start": timeStart,
			"time_end":   timeEnd,
		})

		if err != nil {
			logp.Err("Error downloading logs from CF: %v", err)
		}

		//----------------------------------------------------

		// Now go over the file scanner
		logp.Info("About to scan gzip log file...")

		fh, err := os.Open(cc.LogfileName)
		if err != nil {
			logp.Err("Error opening gziped file for reading: %v", err)
			return err
		}
		defer fh.Close()

		/* Now we need to read the content form the file, split line by line and itterate over them */
		gz, err := gzip.NewReader(fh)
		if err != nil {
			logp.Err("Could not open file for reading: %v", err)
			return err
		}
		defer gz.Close()

		scanner := bufio.NewScanner(gz)

		for scanner.Scan() {

			logItem = scanner.Bytes()

			err := ffjson.Unmarshal(logItem, &l)

			if err == nil {

				counter++
				ts, err := l.GetValue("timestamp")
				if err == nil {
					l["timestamp"] = int64(ts.(float64)) / 1000000
				}
				l["@timestamp"] = common.Time(time.Unix(0, int64(ts.(float64))).UTC())
				l["type"] = "cloudflare"
				l["counter"] = counter

				//**********************************************************************************
				//	Now fix all the nanosecond timestamps and convert them to millisecond timestamps
				//	as Elasticsearch doesn't support nanoseconds
				//**********************************************************************************/
				edge, err := l.GetValue("edge")
				if err == nil {
					switch edge.(map[string]interface{})["startTimestamp"].(type) {
					case int64:
						//ns := edge.(map[string]interface{})["startTimestamp"].(int64)
						l["edge"].(map[string]interface{})["startTimestamp"] = edge.(map[string]interface{})["startTimestamp"].(int64) / 1000000
					case float64:
						//ns := edge.(map[string]interface{})["startTimestamp"].(float64)
						l["edge"].(map[string]interface{})["startTimestamp"] = int64(edge.(map[string]interface{})["startTimestamp"].(float64)) / 1000000
					}
					switch edge.(map[string]interface{})["endTimestamp"].(type) {
					case int64:
						//ns := edge.(map[string]interface{})["endTimestamp"].(int64)
						l["edge"].(map[string]interface{})["endTimestamp"] = edge.(map[string]interface{})["endTimestamp"].(int64) / 1000000
					case float64:
						//ns := edge.(map[string]interface{})["endTimestamp"].(float64)
						l["edge"].(map[string]interface{})["endTimestamp"] = int64(edge.(map[string]interface{})["endTimestamp"].(float64)) / 1000000
					}
				}
				cache, err := l.GetValue("cache")
				if err == nil {
					switch cache.(map[string]interface{})["startTimestamp"].(type) {
					case int64:
						//ns := cache.(map[string]interface{})["startTimestamp"].(int64)
						l["cache"].(map[string]interface{})["startTimestamp"] = cache.(map[string]interface{})["startTimestamp"].(int64) / 1000000
					case float64:
						//ns := cache.(map[string]interface{})["startTimestamp"].(float64)
						l["cache"].(map[string]interface{})["startTimestamp"] = int64(cache.(map[string]interface{})["startTimestamp"].(float64)) / 1000000
					}
					switch cache.(map[string]interface{})["endTimestamp"].(type) {
					case int64:
						//ns := cache.(map[string]interface{})["endTimestamp"].(int64)
						l["cache"].(map[string]interface{})["endTimestamp"] = cache.(map[string]interface{})["endTimestamp"].(int64) / 1000000
					case float64:
						//ns := cache.(map[string]interface{})["endTimestamp"].(float64)
						l["cache"].(map[string]interface{})["endTimestamp"] = int64(cache.(map[string]interface{})["endTimestamp"].(float64)) / 1000000
					}

					logp.Info("cache.cacheInternalIp: %s", cache.(map[string]interface{})["cacheInternalIp"].(string))

					// Deal with this field potentially being empty
					if cache.(map[string]interface{})["cacheInternalIp"].(string) == "" {
						l.Delete("cache.cacheInternalIp")
					}

				}

				waf, err := l.GetValue("waf")
				if err == nil {
					switch waf.(map[string]interface{})["timestampStart"].(type) {
					case int64:
						//ns := waf.(map[string]interface{})["timestampStart"].(int64)
						l["waf"].(map[string]interface{})["timestampStart"] = waf.(map[string]interface{})["timestampStart"].(int64) / 1000000
					case float64:
						//ns := waf.(map[string]interface{})["timestampStart"].(float64)
						l["waf"].(map[string]interface{})["timestampStart"] = int64(waf.(map[string]interface{})["timestampStart"].(float64)) / 1000000
					}
					switch waf.(map[string]interface{})["timestampEnd"].(type) {
					case int64:
						//ns := waf.(map[string]interface{})["timestampEnd"].(int64)
						l["waf"].(map[string]interface{})["timestampEnd"] = waf.(map[string]interface{})["timestampEnd"].(int64) / 1000000
					case float64:
						//ns := waf.(map[string]interface{})["timestampEnd"].(float64)
						l["waf"].(map[string]interface{})["timestampEnd"] = int64(waf.(map[string]interface{})["timestampEnd"].(float64)) / 1000000
					}
				}

				or, err := l.GetValue("originResponse")
				if err == nil {
					switch or.(map[string]interface{})["httpExpires"].(type) {
					case int64:
						//ns := or.(map[string]interface{})["httpExpires"].(int64)
						l["originResponse"].(map[string]interface{})["httpExpires"] = or.(map[string]interface{})["httpExpires"].(int64) / 1000000
					case float64:
						//ns := or.(map[string]interface{})["httpExpires"].(float64)
						l["originResponse"].(map[string]interface{})["httpExpires"] = int64(or.(map[string]interface{})["httpExpires"].(float64)) / 1000000
					}

				}

				bt.client.PublishEvent(l)
				currCount++
			} else {
				logp.Err("Could not load JSON: %s", err)
			}

		}

		//-----------------------------------------------------

		if currCount >= 1 {
			//bt.client.PublishEvents(logs)
			logp.Info("Total events sent this period: %d", currCount)
			// Now need to update the disk-based state file that keeps track of the current state
			bt.state.UpdateLastStartTS(timeStart)
			bt.state.UpdateLastEndTS(timeEnd)
			bt.state.UpdateLastCount(len(logs))
			bt.state.UpdateLastRequestTS(timeNow)
			currCount = 0
		}

		if err := bt.state.Save(); err != nil {
			logp.Err("Could not persist state file to storage: %s", err.Error())
		}

		counter++

	}

}

func (bt *Cloudflarebeat) Stop() {
	bt.client.Close()
	close(bt.done)
	if err := bt.state.Save(); err != nil {
		logp.Err("Could not persist state file to storage while shutting down: %s", err.Error())
	}
}
