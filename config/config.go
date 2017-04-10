// Config is put into a different package to prevent cyclic imports in case
// it is needed in several locations

package config

import "time"

type Config struct {
	Period                       time.Duration `config:"period"`
	APIKey                       string        `config:"api_key"`
	Email                        string        `config:"email"`
	APIServiceKey                string        `config:"api_service_key"`
	ZoneTag                      string        `config:"zone_tag"`
	StateFileStorageType         string        `config:"state_file_storage_type"`
	StateFileName                string        `config:"state_file_name"`
	StateFilePath                string        `config:"state_file_path"`
	AwsAccessKey                 string        `config:"aws_access_key"`
	AwsSecretAccessKey           string        `config:"aws_secret_access_key"`
	AwsS3BucketName              string        `config:"aws_s3_bucket_name"`
	DeleteLogFileAfterProcessing bool          `config:"delete_logfile_after_processing"`
	ProcessedEventsBufferSize    int           `config:"processed_events_buffer_size"`
	ParallelLogProcessing        bool          `config:"parallel_log_processing"`
	NumWorkers                   int           `config:"num_workers"`
	TmpLogsDir                   string        `config:"tmp_logs_dir"`
	Debug                        bool          `config:"debug"`
}

var DefaultConfig = Config{
	Period:                       10 * time.Minute,
	StateFileStorageType:         "disk",
	StateFileName:                "cloudflarebeat",
	StateFilePath:                "/etc/cloudflarebeat/",
	DeleteLogFileAfterProcessing: true,
	ProcessedEventsBufferSize:    1000,
	TmpLogsDir:                   "tmp_logs/",
	ParallelLogProcessing:        false,
	NumWorkers:                   1,
	Debug:                        false,
}
