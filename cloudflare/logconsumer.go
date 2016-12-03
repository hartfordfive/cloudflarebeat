package cloudflare

import (
	"fmt"
)

type LogConsumer struct {
	logFileSegmentSize int
	totalDownloaders   int
	totalProcessors    int
}

func NewLogConsumer(numSegments int, donwloaders int, processors int) *LogConsumer {
	return &LogConsumer{
		logFileSegmentSize: numSegments,
		totalDownloaders:   donwloaders,
		totalProcessors:    processors,
	}
}

// DownloadCurrentLogFiles downloads the log file segments from the Cloudflare ELS API
func (lc *LogConsumer) DownloadCurrentLogFiles() {

  

}

func (lc *LogConsumer) ProcessCurrentLogFiles() {

}
