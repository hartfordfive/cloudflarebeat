package cloudflare

import (
	"time"

	"github.com/elastic/beats/libbeat/logp"
)

type LogConsumer struct {
	TotalLogFileSegments int
	//segmentSize          int
	totalDownloaders int
	totalProcessors  int
	cloudflareClient *CloudflareClient
	LogFilesReady    chan string
}

// NewLogConsumer reutrns a instance of the LogConsumer struct
func NewLogConsumer(cfEmail string, cfApiKey string, numSegments int, donwloaders int, processors int) *LogConsumer {

	lc := &LogConsumer{
		TotalLogFileSegments: numSegments,
		//segmentSize:          ((timeEnd - timeStart) / numSegments),
		totalDownloaders: donwloaders,
		totalProcessors:  processors,
		LogFilesReady:    make(chan string, numSegments),
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

func (lc *LogConsumer) ProcessCurrentLogFiles() {

	// 1.  Get log file from LogFilesReady channel
	// 2. Process the file

}
