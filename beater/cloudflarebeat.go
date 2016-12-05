package beater

import (
	//"bufio"
	//"compress/gzip"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/publisher"
	"github.com/hartfordfive/cloudflarebeat/cloudflare"
	"github.com/hartfordfive/cloudflarebeat/config"
	//"github.com/pquerna/ffjson/ffjson"
)

const (
	STATEFILE_NAME         = "cloudflarebeat.state"
	OFFSET_PAST_MINUTES    = 30
	TOTAL_LOGFILE_SEGMENTS = 6
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
		logConsumer: cloudflare.NewLogConsumer(config.Email, config.APIKey, TOTAL_LOGFILE_SEGMENTS, 6, 6),
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

	var timeStart, timeEnd, timeNow int
	//var l map[string]interface{}
	var evt common.MapStr
	//var logItem []byte

	//var currCount int

	for {

		select {
		case <-bt.done:
			return nil
		case <-ticker.C:
		}

		var wg sync.WaitGroup
		timeNow = int(time.Now().UTC().Unix())

		if bt.state.GetLastStartTS() != 0 {
			timeStart = bt.state.GetLastEndTS() + 1 // last end TS as per statefile + 1 second
			//timeEnd = timeStart + (int(bt.config.Period.Minutes()) * 60) // to 30 minutes later
			//timeEnd = timeStart + (int(bt.config.Period.Minutes()) * 60) // to 30 minutes later
		} else {
			// Start 30 MINUTES - SPECIFIED PERIOD MINUTES AGO
			timeStart = timeNow - (OFFSET_PAST_MINUTES * 60) - (int(bt.config.Period.Minutes()) * 60)
			//timeEnd = timeStart + (int(bt.config.Period.Minutes()) * 60) // to 30 minutes ago
		}

		// up to X minutes ago, 1 >= X <= 30
		timeEnd = timeStart + (int(bt.config.Period.Minutes()) * 60) //

		bt.state.UpdateLastRequestTS(timeNow)

		wg.Add(1)

		go bt.logConsumer.DownloadCurrentLogFiles(bt.config.ZoneTag, timeStart, timeEnd)
		go bt.logConsumer.PrepareEvents()
		//----------------------------------------------------

		go func() {
			for {
				select {
				case <-bt.logConsumer.CompletedNotifier:
					logp.Info("Completed processing all events for this time period")
					wg.Done()
					break
				case evt = <-bt.logConsumer.EventsReady:
					bt.client.PublishEvent(evt)
				}
			}
		}()

		wg.Wait()

		//if currCount >= 1 {
		//logp.Info("Total events sent this period: %d", currCount)
		// Now need to update the disk-based state file that keeps track of the current state
		bt.state.UpdateLastStartTS(timeStart)
		bt.state.UpdateLastEndTS(timeEnd)
		//bt.state.UpdateLastCount(currCount)
		bt.state.UpdateLastRequestTS(timeNow)
		//currCount = 0
		//}

		if err := bt.state.Save(); err != nil {
			logp.Err("Could not persist state file to storage: %s", err.Error())
		} else {
			logp.Info("Updated state file")
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
