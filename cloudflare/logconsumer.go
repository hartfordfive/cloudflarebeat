package cloudflare

import (
	"bufio"
	"compress/gzip"
	"os"
	"runtime"
	"sync"
	"time"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/pquerna/ffjson/ffjson"
)

type LogConsumer struct {
	TotalLogFileSegments  int
	cloudflareClient      *CloudflareClient
	LogFilesReady         chan string
	EventsReady           chan common.MapStr
	CompletedNotifier     chan bool
	ProcessorTerminateSig chan bool
	WaitGroup             sync.WaitGroup
}

// NewLogConsumer reutrns a instance of the LogConsumer struct
func NewLogConsumer(cfEmail string, cfAPIKey string, numSegments int, eventBufferSize int, processors int) *LogConsumer {

	lc := &LogConsumer{
		TotalLogFileSegments:  numSegments,
		LogFilesReady:         make(chan string, numSegments*10),
		EventsReady:           make(chan common.MapStr, eventBufferSize),
		CompletedNotifier:     make(chan bool, 1),
		ProcessorTerminateSig: make(chan bool, processors),
		WaitGroup:             sync.WaitGroup{},
	}
	lc.cloudflareClient = NewClient(map[string]interface{}{
		"api_key": cfAPIKey,
		"email":   cfEmail,
		"debug":   true,
	})
	return lc
}

// DownloadCurrentLogFiles downloads the log file segments from the Cloudflare ELS API
func (lc *LogConsumer) DownloadCurrentLogFiles(zoneTag string, timeStart int, timeEnd int) {

	segmentSize := (timeEnd - timeStart) / lc.TotalLogFileSegments
	currTimeStart := timeStart
	currTimeEnd := currTimeStart + segmentSize
	var timeNow int

	lc.WaitGroup.Add(lc.TotalLogFileSegments)

	for i := 0; i < lc.TotalLogFileSegments; i++ {
		go func(lc *LogConsumer, segmentNum int, currTimeStart int, currTimeEnd int) {

			timeNow = int(time.Now().UTC().Unix())

			logp.Info("Downloading log segment #%d from %d to %d", segmentNum, currTimeStart, currTimeEnd)

			filename, err := lc.cloudflareClient.GetLogRangeFromTimestamp(map[string]interface{}{
				"zone_tag":   zoneTag,
				"time_start": currTimeStart,
				"time_end":   currTimeEnd,
			})

			if err != nil {
				logp.Err("Error downloading logs from CF: %v", err)
				return
			}

			lc.LogFilesReady <- filename
			logp.Info("Total download time for log file: %d seconds", (int(time.Now().UTC().Unix()) - timeNow))

		}(lc, i, currTimeStart, currTimeEnd)

		currTimeStart = currTimeEnd + 1
		currTimeEnd = currTimeStart + segmentSize

		runtime.Gosched()
	}

}

func (lc *LogConsumer) PrepareEvents() {

	var l map[string]interface{}
	var evt common.MapStr
	var logItem []byte
	completedProcessingNotifer := make(chan bool, 1)

	// goroutine that will send notification to the goroutine publishing the events to say it's done all the files
	go func() {
		lc.WaitGroup.Wait()
		lc.CompletedNotifier <- true
		completedProcessingNotifer <- true
	}()

	for {

		select {

		case <-completedProcessingNotifer:
			logp.Info("Done preparing events for publishing. Returning from goroutine.")
			return
		case logFileName := <-lc.LogFilesReady:

			logp.Info("Log file %s ready for processing.", logFileName)
			fh, err := os.Open(logFileName)
			if err != nil {
				logp.Err("Error opening gziped file for reading: %v", err)
				DeleteLogLife(logFileName)
				lc.WaitGroup.Done()
				continue
			}
			defer fh.Close()

			/* Now we need to read the content form the file, split line by line and itterate over them */
			logp.Info("Opening gziped file %s for reading...", logFileName)
			gz, err := gzip.NewReader(fh)
			if err != nil {
				logp.Err("Could not open file for reading: %v", err)
				DeleteLogLife(logFileName)
				lc.WaitGroup.Done()
				continue
			}
			defer gz.Close()

			timePreIndex := int(time.Now().UTC().Unix())
			scanner := bufio.NewScanner(gz)

			for scanner.Scan() {
				logItem = scanner.Bytes()
				err := ffjson.Unmarshal(logItem, &l)
				if err == nil {
					evt = BuildMapStr(l)
					evt["@timestamp"] = common.Time(time.Unix(0, int64(l["timestamp"].(float64))))
					evt["type"] = "cloudflare"
					lc.EventsReady <- evt
				} else {
					logp.Err("Could not load JSON: %s", err)
				}
			}

			logp.Info("Total processing time: %d seconds", (int(time.Now().UTC().Unix()) - timePreIndex))

			// Now delete the log file
			DeleteLogLife(logFileName)
			lc.WaitGroup.Done()
			runtime.Gosched()

		} // END select

	} // End for loop

}
