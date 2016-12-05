package cloudflare

import (
	"bufio"
	"compress/gzip"
	"os"
	"sync"
	"time"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/pquerna/ffjson/ffjson"
)

type LogConsumer struct {
	TotalLogFileSegments int
	//segmentSize          int
	totalDownloaders  int
	totalProcessors   int
	cloudflareClient  *CloudflareClient
	LogFilesReady     chan string
	EventsReady       chan common.MapStr
	CompletedNotifier chan bool
	WaitGroup         sync.WaitGroup
}

// NewLogConsumer reutrns a instance of the LogConsumer struct
func NewLogConsumer(cfEmail string, cfApiKey string, numSegments int, donwloaders int, processors int) *LogConsumer {

	lc := &LogConsumer{
		TotalLogFileSegments: numSegments,
		//segmentSize:          ((timeEnd - timeStart) / numSegments),
		totalDownloaders:  donwloaders,
		totalProcessors:   processors,
		LogFilesReady:     make(chan string, numSegments),
		EventsReady:       make(chan common.MapStr),
		CompletedNotifier: make(chan bool, 1),
		WaitGroup:         sync.WaitGroup{},
	}
	lc.cloudflareClient = NewClient(map[string]interface{}{
		"api_key": cfApiKey,
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

	}

}

func (lc *LogConsumer) PrepareEvents() {

	var l map[string]interface{}
	var evt common.MapStr
	var logItem []byte
	//filesProcessed := 0

	go func() {
		lc.WaitGroup.Wait()
		lc.CompletedNotifier <- true
		//close(c)
	}()

	for {

		select {
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
				//l = nil
			}

			logp.Info("Total processing time: %d seconds", (int(time.Now().UTC().Unix()) - timePreIndex))

			// Now delete the log file
			DeleteLogLife(logFileName)

			//filesProcessed++

			/*
				if filesProcessed == bt.logConsumer.TotalLogFileSegments {
					break
				}
			*/
			logp.Info("")
			lc.WaitGroup.Done()

		} // END select

	} // End for loop

}
