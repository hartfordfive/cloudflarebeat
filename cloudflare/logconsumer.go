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

type LogConsumer struct {
	TotalLogFileSegments  int
	cloudflareClient      *CloudflareClient
	LogFilesReady         chan string
	LogFileSegments       chan LogFileSegment
	EventsReady           chan common.MapStr
	CompletedNotifier     chan bool
	ProcessorTerminateSig chan bool
	ParallelLogProcessing bool
	NumWorkers            int
	Period                time.Duration
	WaitGroup             sync.WaitGroup
	WaitGroupDL           sync.WaitGroup
	DeleteLogFile         bool
	TmpLogFilesDir        string
}

type LogFileSegment struct {
	TimeStart int
	TimeEnd   int
}

// NewLogConsumer reutrns a instance of the LogConsumer struct
//func NewLogConsumer(cfEmail string, cfAPIKey string, numSegments int, eventBufferSize int, processors int, deleteLogFile bool, parallelLogProcessing bool, tmpLogFilesDir string) *LogConsumer {
func NewLogConsumer(cfg config.Config) *LogConsumer {

	nSegments := GetNumLogFileSegments(cfg.Period)

	lc := &LogConsumer{
		TotalLogFileSegments: nSegments,
		LogFilesReady:        make(chan string, nSegments),
		LogFileSegments:      make(chan LogFileSegment, nSegments),
		EventsReady:          make(chan common.MapStr, cfg.ProcessedEventsBufferSize),
		CompletedNotifier:    make(chan bool, 1),
		//WorkerCompletionNotifier: []make(chan bool, numSegments),
		//ProcessorTerminateSig: make(chan bool, processors),
		ParallelLogProcessing: cfg.ParallelLogProcessing,
		NumWorkers:            cfg.NumWorkers,
		Period:                cfg.Period,
		WaitGroup:             sync.WaitGroup{},
		WaitGroupDL:           sync.WaitGroup{},
		DeleteLogFile:         cfg.DeleteLogFileAfterProcessing,
		TmpLogFilesDir:        cfg.TmpLogsDir,
		cloudflareClient: NewClient(map[string]interface{}{
			"api_key": cfg.APIKey,
			"email":   cfg.Email,
			"debug":   false,
		}),
	}
	return lc
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
/*
func (lc *LogConsumer) DownloadCurrentLogFiles(zoneTag string, timeStart int, timeEnd int) {

	segmentSize := (timeEnd - timeStart) / lc.TotalLogFileSegments
	currTimeStart := timeStart
	currTimeEnd := currTimeStart + segmentSize
	var timeNow int

	lc.WaitGroup.Add(lc.TotalLogFileSegments)

	for i := 0; i < lc.TotalLogFileSegments; i++ {
		go func(lc *LogConsumer, segmentNum int, currTimeStart int, currTimeEnd int) {

			timeNow = int(time.Now().UTC().Unix())
			//retryMax := 2
			//retryWait := 5

			logp.Info("Downloading log segment #%d from %d to %d", segmentNum, currTimeStart, currTimeEnd)
			//logp.Info("Downloading log segment #%d from %d to %d", segmentNum, currTimeStart, currTimeEnd)

			filename, err := lc.cloudflareClient.GetLogRangeFromTimestamp(map[string]interface{}{
				"zone_tag":     zoneTag,
				"time_start":   currTimeStart,
				"time_end":     currTimeEnd,
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

		}(lc, i, currTimeStart, currTimeEnd)

		currTimeStart = currTimeEnd + 1
		currTimeEnd = currTimeStart + segmentSize

		runtime.Gosched()
	}

}
*/

/***********************************************************************/

func (lc *LogConsumer) DownloadCurrentLogFiles(zoneTag string, timeStart int, timeEnd int) {

	/*
		segmentSize := (timeEnd - timeStart) / lc.TotalLogFileSegments
		currTimeStart := timeStart
		currTimeEnd := currTimeStart + segmentSize
	*/

	completedDownloadingNotifer := make(chan bool, 1)
	nSegments := GetNumLogFileSegments(lc.Period)
	lc.WaitGroupDL.Add(nSegments)

	go func() {
		lc.WaitGroupDL.Wait()
		//lc.CompletedNotifier <- true
		//completedProcessingNotifer <- true
		// This will notify all goroutines to return from the for/select processing loop
		close(completedDownloadingNotifer)
	}()

	//timeNow := int(time.Now().UTC().Unix())
	// ----------------

	//fmt.Printf("Total number of segments: %d\n", nSegments)

	/*
		now := time.Now().Unix()
		timeStart := int(now - (int64(duration / time.Second)) - (1800))
		timeEnd := int(now - 1800)
		fmt.Printf("Time start: %d, Time end: %d\n", timeStart, timeEnd)
		segments := GetLogFileSegments(nSegments, timeStart, timeEnd)
	*/

	// ----------------

	for i := 0; i < lc.NumWorkers; i++ {
		lc.logDownloadingWorker(completedDownloadingNotifer, zoneTag)
	}

}

func (lc *LogConsumer) logDownloadingWorker(completedDownloadingNotifer chan bool, zoneTag string) {

	for {
		select {
		case <-completedDownloadingNotifer:
			logp.Info("Done preparing events for publishing. Returning from goroutine.")
			return
		case ls := <-lc.LogFileSegments:
			logp.Info("Downloading log segment (%d to %d)", ls.TimeStart, ls.TimeEnd)

			dlStart := int(time.Now().UTC().Unix())

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
			logp.Info("Total download time for log file: %d seconds", (int(time.Now().UTC().Unix()) - dlStart))
		}
	}

	runtime.Gosched()
}

/***********************************************************************/

func (lc *LogConsumer) PrepareEvents() {

	completedProcessingNotifer := make(chan bool, 1)

	// goroutine that will send notification to the goroutine publishing the events to say it's done all the files
	go func() {
		lc.WaitGroup.Wait()
		//lc.CompletedNotifier <- true
		//completedProcessingNotifer <- true
		// This will notify all goroutines to return from the for/select processing loop
		close(completedProcessingNotifer)
	}()

	/*
		for {

			select {

			case <-completedProcessingNotifer:
				logp.Info("Done preparing events for publishing. Returning from goroutine.")
				return
			case logFileName := <-lc.LogFilesReady:
				if lc.ParallelLogProcessing {
					go processLogSegmentFile(logFileName, lc.DeleteLogFile, &lc.WaitGroup, lc.EventsReady)
				} else {
					processLogSegmentFile(logFileName, lc.DeleteLogFile, &lc.WaitGroup, lc.EventsReady)
				}
			} // END select

		} // End for loop
	*/
	// Create a pool of workers to process the log file segments
	for w := 0; w <= lc.NumWorkers; w++ {
		go lc.logProcessinggWorker(completedProcessingNotifer)
	}
}

func (lc *LogConsumer) logProcessinggWorker(completedProcessingNotifer chan bool) {
	for {
		select {
		case <-completedProcessingNotifer:
			logp.Info("Done preparing events for publishing. Returning from goroutine.")
			return
		case logFileName := <-lc.LogFilesReady:
			/*
				if lc.ParallelLogProcessing {
					go processLogSegmentFile(logFileName, lc.DeleteLogFile, &lc.WaitGroup, lc.EventsReady)
				} else {
					processLogSegmentFile(logFileName, lc.DeleteLogFile, &lc.WaitGroup, lc.EventsReady)
				}
			*/
			processLogSegmentFile(logFileName, lc.DeleteLogFile, &lc.WaitGroup, lc.EventsReady)
		} // END select
	} // End for loop
}

func processLogSegmentFile(logFileName string, deleteAfteProcessing bool, wg *sync.WaitGroup, eventsReady chan common.MapStr) {
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
