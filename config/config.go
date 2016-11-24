// Config is put into a different package to prevent cyclic imports in case
// it is needed in several locations

package config

import "time"

type Config struct {
	Period               time.Duration `config:"period"`
	APIKey               string        `config:"api_key"`
	Email                string        `config:"email"`
	APIServiceKey        string        `config:"api_service_key"`
	ZoneTag              string        `config:"zone_tag"`
	StateFileStorageType string        `config:"state_file_storage_type"`
	StateFileName        string        `config:"state_file_name"`
	StateFilePath        string        `config:"state_file_path"`
	AwsAccessKey         string        `config:"aws_access_key"`
	AwsSecretAccessKey   string        `config:"aws_secret_access_key"`
	AwsS3BucketName      string        `config:"aws_s3_bucket_name"`
	Debug                bool          `config:"debug"`
}

var DefaultConfig = Config{
	Period:               30 * time.Minute,
	StateFileStorageType: "disk",
	StateFileName:        "cloudflarebeat.state",
	StateFilePath:        "/etc/cloudflarebeat/",
	Debug:                false,
}
