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
	NUM_MINUTES    = 30
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
		"filename":     config.StateFileName,
		"filepath":     config.StateFilePath,
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

	var timeStart, timeEnd, timeNow, timePreIndex int
	//var l common.MapStr
	var l map[string]interface{}
	//var logs []common.MapStr
	var evt common.MapStr
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
			timeStart = bt.state.GetLastEndTS() + 1  // last end TS as per statefile + 1 second
			timeEnd = timeStart + (NUM_MINUTES * 60) // to 30 minutes later
		} else {
			// Set the start and end time if this is the first run
			//timeStart = (timeNow - (NUM_MINUTES * 60)) - (30 * 60) // specified number of minutes ago - 30 minutes
			//timeEnd = timeNow - (30 * 60)                          // Up to 30 minutes ago
			timeStart = timeNow - 3600               // start an hour ago
			timeEnd = timeStart + (NUM_MINUTES * 60) // to 30 minutes ago
		}

		bt.state.UpdateLastRequestTS(timeNow)

		err := cc.GetLogRangeFromTimestamp(map[string]interface{}{
			"zone_tag":   bt.config.ZoneTag,
			"time_start": timeStart,
			"time_end":   timeEnd,
		})

		if err != nil {
			logp.Err("Error downloading logs from CF: %v", err)
		} else {

			logp.Info("Total download time for log file: %d seconds", (int(time.Now().UTC().Unix()) - timeNow))
		}

		//----------------------------------------------------

		// Now go over the file scanner

		fh, err := os.Open(cc.LogfileName)
		if err != nil {
			logp.Err("Error opening gziped file for reading: %v", err)
			return err
		}
		defer fh.Close()

		/* Now we need to read the content form the file, split line by line and itterate over them */
		logp.Info("Opening gziped log file for reading...")
		gz, err := gzip.NewReader(fh)
		if err != nil {
			logp.Err("Could not open file for reading: %v", err)
			return err
		}
		defer gz.Close()

		scanner := bufio.NewScanner(gz)

		timePreIndex = int(time.Now().UTC().Unix())

		for scanner.Scan() {

			logItem = scanner.Bytes()

			err := ffjson.Unmarshal(logItem, &l)

			if err == nil {

				counter++

				evt = cloudflare.BuildMapStr(l)

				ts, err := evt.GetValue("timestamp")
				if err == nil {
					evt["timestamp"] = int64(ts.(float64)) / int64(time.Millisecond)
				}
				evt["@timestamp"] = common.Time(time.Unix(0, int64(ts.(float64))))
				evt["type"] = "cloudflare"
				evt["counter"] = counter

				//**********************************************************************************
				//	Now fix all the nanosecond timestamps and convert them to millisecond timestamps
				//	as Elasticsearch doesn't support nanoseconds
				//**********************************************************************************/

				//cloudflare.FixTimestampFields(l)
				//evt := cloudflare.BuildMapStr(l)

				bt.client.PublishEvent(evt)
				currCount++
			} else {
				logp.Err("Could not load JSON: %s", err)
			}

		}

		logp.Info("Total processing time: %d seconds", (int(time.Now().UTC().Unix()) - timePreIndex))

		//-----------------------------------------------------

		if currCount >= 1 {
			//bt.client.PublishEvents(logs)
			logp.Info("Total events sent this period: %d", currCount)
			// Now need to update the disk-based state file that keeps track of the current state
			bt.state.UpdateLastStartTS(timeStart)
			bt.state.UpdateLastEndTS(timeEnd)
			bt.state.UpdateLastCount(currCount)
			bt.state.UpdateLastRequestTS(timeNow)
			currCount = 0
		}

		if err := bt.state.Save(); err != nil {
			logp.Err("Could not persist state file to storage: %s", err.Error())
		}

		// Now delete the log file
		if err := os.Remove(cc.LogfileName); err != nil {
			logp.Err("Could not delete local log file %s: %s", cc.LogfileName, err.Error())
		} else {
			logp.Info("Deleted local log file %s", cc.LogfileName)
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
