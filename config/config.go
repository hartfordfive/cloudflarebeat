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
	AwsAccessKey         string        `config:"aws_access_key"`
	AwsSecretAccessKey   string        `config:"aws_secret_access_key"`
	StateFileStorageType string        `config:"state_file_storage_type"`
	Exclude              Conditions    `config:"exclude"`
	Debug                bool          `config:"debug"`
	//ExcludeConditions    map[string][]string `config:"exclude_conditions"`
}

type Conditions struct {
	Or  []Condition `config:"or"`
	And []Condition `config:"and"`
}

type Condition struct {
	Query string `config:"query"`
	Value string `config:"value"`
}

var DefaultConfig = Config{
	Period:               30 * time.Minute,
	StateFileStorageType: "disk",
	Debug:                false,
	Exclude: Conditions{
		Or:  make([]Condition, 4),
		And: make([]Condition, 4),
	},
}
