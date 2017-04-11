package beater

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/publisher"
	"github.com/hartfordfive/cloudflarebeat/cloudflare"
	"github.com/hartfordfive/cloudflarebeat/config"
)

const (
	STATEFILE_NAME      = "cloudflarebeat.state"
	OFFSET_PAST_MINUTES = 30
	MIN_PERIOD_LENGTH   = 2
	MAX_PERIOD_LENGTH   = 30
)

type Cloudflarebeat struct {
	done            chan struct{}
	config          config.Config
	client          publisher.Client
	state           *cloudflare.StateFile
	logConsumer     *cloudflare.LogConsumer
	importFromFiles bool
	logFilesDir     string
}

var timeStart, timeEnd, timeNow int
var (
	importFromFiles = flag.Bool("f", false, "Import from *.tar.gz files and exit")
	logFilesDir     = flag.String("fd", "logs/", "Directory from which to read *.tar.gz files (use with -f)")
)

// Creates beater
func New(b *beat.Beat, cfg *common.Config) (beat.Beater, error) {

	config := config.DefaultConfig
	if err := cfg.Unpack(&config); err != nil {
		return nil, fmt.Errorf("Error reading config file: %v", err)
	}

	if config.Period.Minutes() < MIN_PERIOD_LENGTH || config.Period.Minutes() > MAX_PERIOD_LENGTH {
		logp.Warn("Chosen period of %s is not valid (not between %d and %d mins). Changing to 6m", config.Period.String(), MIN_PERIOD_LENGTH, MAX_PERIOD_LENGTH)
		config.Period = MIN_PERIOD_LENGTH * time.Minute
	}

	bt := &Cloudflarebeat{
		done:            make(chan struct{}),
		config:          config,
		logConsumer:     cloudflare.NewLogConsumer(config),
		importFromFiles: *importFromFiles,
		logFilesDir:     *logFilesDir,
	}

	sf, err := cloudflare.NewStateFile(config)
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

	if bt.importFromFiles {
		bt.ReadFilesAndPublish()
		bt.Stop()
		return nil
	}
	/*
		If a state file already exists and is loaded, download and process the cloudflare logs
		immediately from now to the last end timestamp
	*/
	if bt.state.GetLastEndTS() != 0 {

		timeNow = int(time.Now().UTC().Unix())
		timeDiff := int((timeNow - (OFFSET_PAST_MINUTES * 60)) - (bt.state.GetLastEndTS() + 1))
		timeStart = bt.state.GetLastEndTS() + 1

		// If the time difference from NOW to the last time the DownloadAndPublish ran is greater than the configured period,
		// then sleep for the resulting delta, then download and process the logs for the period
		if timeDiff < int(bt.config.Period.Seconds()) {
			timeEnd := timeStart + int(bt.config.Period.Seconds())
			timeWait := int(bt.config.Period.Seconds()) - timeDiff
			logp.Info("Waiting for %d seconds before catching up", timeWait)
			time.Sleep(time.Duration(timeWait) * time.Second)
			logp.Info("Catching up. Processing logs between %s to %s", time.Unix(int64(timeStart), 0), time.Unix(int64(timeEnd), 0))
			bt.DownloadAndPublish(int(time.Now().UTC().Unix()), timeStart, timeEnd)
		} else {
			// In this case, the time difference from NOW to the last time the DownloadAndPublish ran is greater than
			// the configured period, so run immediately before starting the ticker
			timeEnd := timeNow - (OFFSET_PAST_MINUTES * 60)
			logp.Info("Catching up. Immediately processing logs between %s to %s", time.Unix(int64(timeStart), 0), time.Unix(int64(timeEnd), 0))
			bt.DownloadAndPublish(int(time.Now().UTC().Unix()), timeStart, timeEnd)
		}

	}

	logp.Info("Starting ticker with period of %d minute(s)", int(bt.config.Period.Minutes()))
	ticker := time.NewTicker(bt.config.Period)

	for {

		select {
		case <-bt.done:
			return nil
		case <-ticker.C:
		}

		timeNow = int(time.Now().UTC().Unix())
		if bt.state.GetLastStartTS() != 0 {
			timeStart = bt.state.GetLastEndTS() + 1 // last end TS as per statefile + 1 second
		} else {
			timeStart = timeNow - (OFFSET_PAST_MINUTES * 60) - (int(bt.config.Period.Minutes()) * 60) // Start 30 MINUTES - SPECIFIED PERIOD MINUTES AGO
		}
		timeEnd = timeStart + (int(bt.config.Period.Minutes()) * 60) // up to X minutes ago, 1 >= X <= 30

		bt.DownloadAndPublish(timeNow, timeStart, timeEnd)

	}

}

func (bt *Cloudflarebeat) DownloadAndPublish(timeNow int, timeStart int, timeEnd int) {

	bt.state.UpdateLastRequestTS(timeNow)

	// Download the log segement files seperately/in-parallel in a seperate goroutine
	go bt.logConsumer.DownloadCurrentLogFiles(bt.config.ZoneTag, timeStart, timeEnd)

	// As log files become ready, process it it and generate the events in a seperate goroutine
	go bt.logConsumer.PrepareEvents()

	// Finally, publish all the events as they're placed on the channel, then update the state file once completed
	go func(bt *Cloudflarebeat) {
		logp.Info("Creating worker to publish events")
		for {
			select {
			case <-bt.logConsumer.CompletedNotifier:
				logp.Info("Completed processing all events for this time period")
				break
			case evt := <-bt.logConsumer.EventsReady:
				bt.client.PublishEvent(evt)
			}
		}
		bt.state.UpdateLastStartTS(timeStart)
		bt.state.UpdateLastEndTS(timeEnd)
		bt.state.UpdateLastRequestTS(timeNow)
		if err := bt.state.Save(); err != nil {
			logp.Info("[ERROR] Could not persist state file to storage: %s", err.Error())
		} else {
			logp.Info("Updated state file")
		}
	}(bt)

	logp.Info("Log files for time period %d to %d have been queued for download/processing.", timeStart, timeEnd)

}

func (bt *Cloudflarebeat) ReadFilesAndPublish() {

	// *************** Get the list of files in the specified directory **************
	dir, err := os.Open(bt.logFilesDir)
	if isError(err) {
		bt.Stop()
		return
	}
	defer dir.Close()

	fi, err := dir.Stat()
	if isError(err) {
		bt.Stop()
		return
	}

	filenames := make([]string, 0)

	if fi.IsDir() {
		fis, err := dir.Readdir(-1) // -1 means return all the FileInfos
		if isError(err) {
			bt.Stop()
			return
		}
		for _, fileinfo := range fis {
			if !fileinfo.IsDir() {
				filenames = append(filenames, filepath.Join(bt.logFilesDir, fileinfo.Name()))
			}
		}
	}

	if len(filenames) == 0 {
		bt.Stop()
		return
	}

	logp.Info("Total files to process: %d", len(filenames))

	// ****************  Now process each file ******************
	bt.logConsumer.AddCurrentLogFiles(filenames)
	wg := sync.WaitGroup{}
	wg.Add(1)

	numPublisherWorkers := 4
	//var wgPublisherGroup map[int]sync.WaitGroup

	go bt.logConsumer.PrepareEvents()

	// Finally, publish all the events as they're placed on the channel, then update the state file once completed

	go func(bt *Cloudflarebeat, wg *sync.WaitGroup) {
		logp.Info("Waiting for completion of event publishing")
		for {
			select {
			case <-bt.logConsumer.CompletedNotifier:
				logp.Info("Completed processing all event logs from local files")
				close(bt.logConsumer.EventsReady)
				wg.Done()
				break
			}
		}
	}(bt, &wg)

	for i := 0; i < numPublisherWorkers; i++ {
		go func(bt *Cloudflarebeat) {
			logp.Info("Creating worker %d to publish events", i)
			for {
				select {
				case evt := <-bt.logConsumer.EventsReady:
					bt.client.PublishEvent(evt)
				}
			}
		}(bt)
	}

	wg.Wait()

}

func (bt *Cloudflarebeat) Stop() {
	if err := bt.state.Save(); err != nil {
		logp.Info("[ERROR] Could not persist state file to storage while shutting down: %s", err.Error())
	}
	bt.client.Close()
	close(bt.done)
}

func isError(err error) bool {
	if err != nil {
		logp.Err("%v", err)
		return true
	}
	return false
}
