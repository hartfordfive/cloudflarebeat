package beater

import (
	"fmt"
	"os"
	"time"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/publisher"

	"github.com/hartfordfive/cloudflarebeat/cloudflare"
	"github.com/hartfordfive/cloudflarebeat/config"
)

type Cloudflarebeat struct {
	done   chan struct{}
	config config.Config
	client publisher.Client
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
		"exclude": bt.config.Exclude,
	})

	sf := cloudflare.NewStateFile("cloudflarebeat.state", bt.config.StateFileStorageType)
	logp.Info("Initializing state file 'cloudflarebeats.state' with storage type '" + bt.config.StateFileStorageType + "'")
	err := sf.Initialize()

	if err != nil {
		logp.Err("Could not load statefile: %s", err.Error())
		os.Exit(1)
	}

	if sf.GetLastStartTS() != 0 {
		logp.Info("Start time loaded from state file: %s", time.Unix(int64(sf.GetLastStartTS()), 0).Format(time.RFC3339))
		logp.Info("  End time loaded from state file: %s", time.Unix(int64(sf.GetLastEndTS()), 0).Format(time.RFC3339))
	}

	var timeStart, timeEnd, timeNow int

	for {
		select {
		case <-bt.done:
			return nil
		case <-ticker.C:
		}

		timeNow = int(time.Now().UTC().Unix())

		if sf.GetLastStartTS() != 0 {
			timeStart = sf.GetLastEndTS() + 1       // last time start
			timeEnd = sf.GetLastEndTS() + (30 * 60) // to 30 minutes later, minus 1 second
		} else {
			timeStart = timeNow - (30 * 60) // last end TS as per statefile
			timeEnd = timeNow               // to 1 second ago
		}

		logp.Info("Next request start time: %s", time.Unix(int64(timeStart), 0).Format(time.RFC3339))
		logp.Info("  Next request end Time: %s", time.Unix(int64(timeEnd), 0).Format(time.RFC3339))

		logs, err := cc.GetLogRangeFromTimestamp(map[string]interface{}{
			"zone_tag":   bt.config.ZoneTag,
			"time_start": timeStart,
			"time_end":   timeEnd,
			//"count":      10,
		})

		if err != nil {
			logp.Err("GetLogRangeFromTimestamp: %s", err.Error())
			sf.UpdateLastRequestTS(timeNow)
		} else {
			bt.client.PublishEvents(logs)
			logp.Info("Total events sent this period: %d", len(logs))
			// Now need to update the disk-based state file that keeps track of the current state
			sf.UpdateLastStartTS(timeStart)
			sf.UpdateLastEndTS(timeEnd)
			sf.UpdateLastCount(len(logs))
			sf.UpdateLastRequestTS(timeNow)
		}

		err = sf.Save()
		if err != nil {
			logp.Err("Could not persist state file to storage: %s", err.Error())
		}

		counter++
	}

}

func (bt *Cloudflarebeat) Stop() {
	bt.client.Close()
	close(bt.done)
}
