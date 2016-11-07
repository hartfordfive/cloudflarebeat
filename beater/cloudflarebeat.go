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
			timeStart = (timeNow - (60 * 60 * 24 * 3)) // last 72 hours
			timeEnd = timeNow - 1                      // to 1 second ago
		}

		if bt.config.Debug {
			logp.Info("Start Time: %s", time.Unix(int64(timeStart), 0).Format(time.RFC3339))
			logp.Info("  End Time: %s", time.Unix(int64(timeEnd), 0).Format(time.RFC3339))
		}

		logs, err := cc.GetLogRangeFromTimestamp(map[string]interface{}{
			"zone_tag":   bt.config.ZoneTag,
			"time_start": timeStart,
			"time_end":   timeEnd,
		})

		if err != nil {
			logp.Err("GetLogRangeFromTimestamp: %s", err.Error())
			sf.Update(map[string]interface{}{"last_request_ts": timeNow})
		} else {
			if bt.config.Debug {
				logp.Info("Total Logs: %d", len(logs))
			}
			bt.client.PublishEvents(logs)
			if bt.config.Debug {
				logp.Info("Events sent")
			}
			// Now need to update the disk-based state file that keeps track of the current state
			sf.Update(map[string]interface{}{"last_start_ts": timeStart, "last_end_ts": timeEnd, "last_count": len(logs), "last_request_ts": timeNow})
			if bt.config.Debug {
				logp.Info("Updating state file")
			}
		}

		counter++
	}

}

func (bt *Cloudflarebeat) Stop() {
	bt.client.Close()
	close(bt.done)
}
