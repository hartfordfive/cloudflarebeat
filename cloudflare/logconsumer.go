package cloudflare

import (
	"bufio"
	"compress/gzip"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/hartfordfive/cloudflarebeat/config"
	"github.com/pquerna/ffjson/ffjson"
)

const (
	NUM_PROCESSORS = 6
	SEGMENT_SIZE   = 120
)

type LogConsumer struct {
	TotalLogFileSegments  int
	cloudflareClient      *CloudflareClient
	Period                time.Duration
	LogFilesReady         chan string
	EventsReady           chan common.MapStr
	CompletedNotifier     chan bool
	ProcessorTerminateSig chan bool
	ParallelLogProcessing bool
	WaitGroup             sync.WaitGroup
	DeleteLogFile         bool
	TmpLogFilesDir        string
}

type LogFileSegment struct {
	TimeStart int
	TimeEnd   int
}

// NewLogConsumer reutrns a instance of the LogConsumer struct
//func NewLogConsumer(cfEmail string, cfAPIKey string, numSegments int, eventBufferSize int, processors int, deleteLogFile bool, parallelLogProcessing bool, tmpLogFilesDir string) *LogConsumer {
func NewLogConsumer(conf config.Config) *LogConsumer {

	return &LogConsumer{
		TotalLogFileSegments: 6,
		cloudflareClient: NewClient(map[string]interface{}{
			"api_key": conf.APIKey,
			"email":   conf.Email,
			"debug":   conf.Debug,
		}),
		Period:            conf.Period,
		LogFilesReady:     make(chan string, 120),
		EventsReady:       make(chan common.MapStr, conf.ProcessedEventsBufferSize),
		CompletedNotifier: make(chan bool, 1),
		//WorkerCompletionNotifier: []make(chan bool, numSegments),
		ProcessorTerminateSig: make(chan bool, NUM_PROCESSORS),
		ParallelLogProcessing: conf.ParallelLogProcessing,
		WaitGroup:             sync.WaitGroup{},
		DeleteLogFile:         conf.DeleteLogFileAfterProcessing,
		TmpLogFilesDir:        conf.TmpLogsDir,
	}
}

// AddCurrentLogFiles processes the existing log files
func (lc *LogConsumer) AddCurrentLogFiles(logFiles []string) {

	numFiles := len(logFiles)
	lc.WaitGroup.Add(numFiles)
	for _, filename := range logFiles {
		lc.LogFilesReady <- filename
	}
	logp.Info("Added %d log files to be processed", numFiles)
}

// DownloadCurrentLogFiles downloads the log file segments from the Cloudflare ELS API
func (lc *LogConsumer) DownloadCurrentLogFiles(zoneTag string, timeStart int, timeEnd int) {

	var timeNow int
	nSegments := GetNumLogFileSegments(lc.Period)
	segmentsList := GenerateSegmentsList(nSegments, SEGMENT_SIZE, timeStart, timeEnd)

	lc.WaitGroup.Add(nSegments)

	for i := 0; i < nSegments; i++ {
		go func(lc *LogConsumer, segmentNum int, ls LogFileSegment) {

			timeNow = int(time.Now().UTC().Unix())
			//retryMax := 2
			//retryWait := 5

			logp.Info("Downloading log segment #%d from %d to %d", segmentNum, ls.TimeStart, ls.TimeEnd)
			filename, err := lc.cloudflareClient.GetLogRangeFromTimestamp(map[string]interface{}{
				"zone_tag":     zoneTag,
				"time_start":   ls.TimeStart,
				"time_end":     ls.TimeEnd,
				"tmp_logs_dir": lc.TmpLogFilesDir,
			})

			if err != nil {
				logp.Err("Could not download logs from CF: %v", err)
				// Considering this file could not be downloaded, we must mark it as completed processing otherwise the processed
				// will just hang and forever wait for the complete waitgroup to be ack'd
				lc.WaitGroup.Done()
				return
			}

			lc.LogFilesReady <- filename
			logp.Info("Total download time for log file: %d seconds", (int(time.Now().UTC().Unix()) - timeNow))
			//logp.Info("Total download time for log file: %d seconds", (int(time.Now().UTC().Unix()) - timeNow))

		}(lc, i, segmentsList[i])

		runtime.Gosched()
	}

}

func (lc *LogConsumer) PrepareEvents() {

	completedProcessingNotifer := make(chan bool, 1)

	// goroutine that will send notification to the goroutine publishing the events to say it's done all the files
	go func() {
		lc.WaitGroup.Wait()
		lc.CompletedNotifier <- true
		completedProcessingNotifer <- true
		close(completedProcessingNotifer)
	}()

	for {

		select {

		case <-completedProcessingNotifer:
			logp.Info("Done preparing events for publishing. Returning from goroutine.")
			return
		case logFileName := <-lc.LogFilesReady:
			if lc.ParallelLogProcessing {
				go prepareEvent(logFileName, lc.DeleteLogFile, &lc.WaitGroup, lc.EventsReady)
			} else {
				prepareEvent(logFileName, lc.DeleteLogFile, &lc.WaitGroup, lc.EventsReady)
			}
		} // END select

	} // End for loop

}

func prepareEvent(logFileName string, deleteAfteProcessing bool, wg *sync.WaitGroup, eventsReady chan common.MapStr) {
	var l map[string]interface{}
	var evt common.MapStr
	var logItem []byte

	logp.Info("Log file %s ready for processing.", logFileName)
	fh, err := os.Open(logFileName)
	if err != nil {
		logp.Err("Could not open gziped file for reading: %v", err)
		if deleteAfteProcessing {
			DeleteLogFile(logFileName)
		}
		wg.Done()
		return
	}

	/* Now we need to read the content form the file, split line by line and itterate over them */
	logp.Info("Opening gziped file %s for reading...", logFileName)
	gz, err := gzip.NewReader(fh)
	if err != nil {
		logp.Err("Could not open file for reading: %v", err)
		if deleteAfteProcessing {
			DeleteLogFile(logFileName)
		}
		wg.Done()
		return
	}

	timePreIndex := int(time.Now().UTC().Unix())
	scanner := bufio.NewScanner(gz)

	logp.Info("Scanning file...")

	for scanner.Scan() {
		logItem = scanner.Bytes()
		err := ffjson.Unmarshal(logItem, &l)
		if err == nil {
			evt = BuildMapStr(l)
			evt["@timestamp"] = common.Time(time.Unix(0, int64(l["timestamp"].(float64))))
			evt["type"] = "cloudflare"
			evt["cfbeat_log_file"] = filepath.Base(logFileName)
			eventsReady <- evt
		} else {
			logp.Err("Could not load JSON: %s", err)
		}
	}

	logp.Info("Total processing time: %d seconds", (int(time.Now().UTC().Unix()) - timePreIndex))

	// Now close the related handles and delete the log file
	gz.Close()
	fh.Close()
	if deleteAfteProcessing {
		DeleteLogFile(logFileName)
	}

	wg.Done()
	runtime.Gosched()
}
