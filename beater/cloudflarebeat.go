package beater

import (
	"fmt"
	"time"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/publisher"

	"github.com/hartfordfive/cloudflarebeat/cloudflare"
	"github.com/hartfordfive/cloudflarebeat/config"
)

const STATEFILE_NAME = "cloudflarebeat.state"

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
	counter := 1

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

	for {

		select {
		case <-bt.done:
			return nil
		case <-ticker.C:
		}

		timeNow = int(time.Now().UTC().Unix())

		if bt.state.GetLastStartTS() != 0 {
			timeStart = bt.state.GetLastEndTS() + 1       // last end TS as per statefile
			timeEnd = bt.state.GetLastEndTS() + (30 * 60) // to 30 minutes later, minus 1 second
		} else {
			// Get the last 3 hours for first run
			timeStart = timeNow - (60 * 60)
			timeEnd = timeNow
		}

		logs, err := cc.GetLogRangeFromTimestamp(map[string]interface{}{
			"zone_tag":   bt.config.ZoneTag,
			"time_start": timeStart,
			"time_end":   timeEnd,
		})

		if err != nil {
			logp.Err("GetLogRangeFromTimestamp: %s", err.Error())
			bt.state.UpdateLastRequestTS(timeNow)
		} else {
			bt.client.PublishEvents(logs)
			logp.Info("Total events sent this period: %d", len(logs))
			// Now need to update the disk-based state file that keeps track of the current state
			bt.state.UpdateLastStartTS(timeStart)
			bt.state.UpdateLastEndTS(timeEnd)
			bt.state.UpdateLastCount(len(logs))
			bt.state.UpdateLastRequestTS(timeNow)
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
