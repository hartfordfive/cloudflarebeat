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
	"github.com/hartfordfive/cloudflarebeat/cloudflare"
	"github.com/hartfordfive/cloudflarebeat/config"
	"github.com/pquerna/ffjson/ffjson"
)

const (
	STATEFILE_NAME     = "cloudflarebeat.state"
	MIN_OFFSET_MINUTES = 30 * 60
)

type Cloudflarebeat struct {
	done        chan struct{}
	config      config.Config
	client      publisher.Client
	state       *cloudflare.StateFile
	logConsumer *cloudflare.LogConsumer
}

// Creates beater
func New(b *beat.Beat, cfg *common.Config) (beat.Beater, error) {
	config := config.DefaultConfig
	if err := cfg.Unpack(&config); err != nil {
		return nil, fmt.Errorf("Error reading config file: %v", err)
	}

	if config.Period.Minutes() < 1 || config.Period.Minutes() > 30 {
		logp.Warn("Chosen period of %s is not valid. Changing to 5m", config.Period.String())
		config.Period = 5 * time.Minute
	}

	bt := &Cloudflarebeat{
		done:        make(chan struct{}),
		config:      config,
		logConsumer: cloudflare.NewLogConsumer(config.Email, config.APIKey, 6, 6, 6),
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
	/*
		cc := cloudflare.NewClient(map[string]interface{}{
			"api_key": bt.config.APIKey,
			"email":   bt.config.Email,
			"debug":   bt.config.Debug,
		})
	*/

	if bt.state.GetLastStartTS() != 0 {
		logp.Info("Start time loaded from state file: %s", time.Unix(int64(bt.state.GetLastStartTS()), 0).Format(time.RFC3339))
		logp.Info("  End time loaded from state file: %s", time.Unix(int64(bt.state.GetLastEndTS()), 0).Format(time.RFC3339))
	}

	var timeStart, timeEnd, timeNow, timePreIndex int
	//var l map[string]interface{}
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
			timeStart = bt.state.GetLastEndTS() + 1 // last end TS as per statefile + 1 second
			//timeEnd = timeStart + (int(bt.config.Period.Minutes()) * 60) // to 30 minutes later
			//timeEnd = timeStart + (int(bt.config.Period.Minutes()) * 60) // to 30 minutes later
		} else {
			// Start 30 MINUTES - SPECIFIED PERIOD MINUTES AGO
			timeStart = timeNow - MIN_OFFSET_MINUTES - (int(bt.config.Period.Minutes()) * 60)
			//timeEnd = timeStart + (int(bt.config.Period.Minutes()) * 60) // to 30 minutes ago
		}

		// up to X minutes ago, 1 >= X <= 30
		timeEnd = timeStart + (int(bt.config.Period.Minutes()) * 60) //

		bt.state.UpdateLastRequestTS(timeNow)

		/*
			err := cc.GetLogRangeFromTimestamp(map[string]interface{}{
				"zone_tag":   bt.config.ZoneTag,
				"time_start": timeStart,
				"time_end":   timeEnd,
			}

			if err != nil {
				logp.Err("Error downloading logs from CF: %v", err)
				continue
			} else {
				logp.Info("Total download time for log file: %d seconds", (int(time.Now().UTC().Unix()) - timeNow))
			}
		*/

		go bt.logConsumer.DownloadCurrentLogFiles(bt.config.ZoneTag, timeStart, timeEnd)

		//----------------------------------------------------
		// Start of processing goroutine
		go func(bt *Cloudflarebeat) {

			filesProcessed := 0
			var l map[string]interface{}

			for {

				select {
				case logFileName := <-bt.logConsumer.LogFilesReady:

					logp.Info("Log file %s ready for processing.", logFileName)
					fh, err := os.Open(logFileName)
					if err != nil {
						logp.Err("Error opening gziped file for reading: %v", err)
						bt.RemoveLogFile(logFileName)
						continue
					}
					defer fh.Close()

					/* Now we need to read the content form the file, split line by line and itterate over them */
					logp.Info("Opening gziped log file for reading...")
					gz, err := gzip.NewReader(fh)
					if err != nil {
						logp.Err("Could not open file for reading: %v", err)
						bt.RemoveLogFile(logFileName)
						continue
					}
					defer gz.Close()

					timePreIndex = int(time.Now().UTC().Unix())
					scanner := bufio.NewScanner(gz)

					for scanner.Scan() {
						logItem = scanner.Bytes()
						err := ffjson.Unmarshal(logItem, &l)
						if err == nil {
							counter++
							evt = cloudflare.BuildMapStr(l)
							evt["@timestamp"] = common.Time(time.Unix(0, int64(l["timestamp"].(float64))))
							evt["type"] = "cloudflare"
							evt["counter"] = counter
							bt.client.PublishEvent(evt)
							currCount++
						} else {
							logp.Err("Could not load JSON: %s", err)
						}
						l = nil
					}

					logp.Info("Total processing time: %d seconds", (int(time.Now().UTC().Unix()) - timePreIndex))

					// Now delete the log file
					bt.RemoveLogFile(logFileName)

					filesProcessed++

					if filesProcessed == bt.logConsumer.TotalLogFileSegments {
						break
					}

				} // END select

			} // End for loop

		}(bt) // End of processing go routine

		//-----------------------------------------------------

		if currCount >= 1 {
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

		counter++

	}

}

func (bt *Cloudflarebeat) RemoveLogFile(logFileName string) {
	if bt.config.DeleteLogFileAfterProcessing {
		if err := os.Remove(logFileName); err != nil {
			logp.Err("Could not delete local log file %s: %s", logFileName, err.Error())
		} else {
			logp.Info("Deleted local log file %s", logFileName)
		}
	}
}

func (bt *Cloudflarebeat) Stop() {
	bt.client.Close()
	close(bt.done)
	if err := bt.state.Save(); err != nil {
		logp.Err("Could not persist state file to storage while shutting down: %s", err.Error())
	}
}
