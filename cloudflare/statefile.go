package cloudflare

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"os"
	"time"

	"github.com/elastic/beats/libbeat/logp"
)

type StateFile struct {
	FileName    string
	StorageType string
	properties  Properties
	lastUpdated time.Time
}

type Properties struct {
	LastStartTS   int `json:"last_start_ts"`
	LastEndTS     int `json:"last_end_ts"`
	LastCount     int `json:"last_count"`
	LastRequestTS int `json:"last_request_ts"`
}

func NewStateFile(fileName string, storageType string) *StateFile {

	return &StateFile{
		FileName:    fileName,
		StorageType: storageType,
	}

}

func (s *StateFile) Initialize() error {
	// 2. If it doesn't exists, create it
	if s.StorageType == "disk" {
		return s.loadFromDisk()
	}
	return errors.New("Unsupported storage type")
}

func (s *StateFile) loadFromDisk() error {

	// Create it if it doesn't exist
	if _, err := os.Stat(s.FileName); os.IsNotExist(err) {
		var file, err = os.Create(s.FileName)
		defer file.Close()
		if err != nil {
			return err
		}
		return nil
	}

	// Now load the file in memory
	sfData, err := ioutil.ReadFile(s.FileName)
	if err != nil {
		return err
	}

	var dat Properties
	if err := json.Unmarshal(sfData, &dat); err != nil {
		// If the state file isn't valid json, then re-create it
		if err != nil {
			logp.Err("%s", err)
			err = os.Remove(s.FileName)
			var file, err = os.Create(s.FileName)
			defer file.Close()
			if err != nil {
				return err
			}
			return nil
		}
	}

	s.properties = dat

	return nil
}

func (s *StateFile) loadFromS3(bucketName string, fileName string, AWSAccessKey string, AWSSecretKey string) {
	// TODO
}

func (s *StateFile) loadFromConsul(filePath string) {
	// TODO
}

/*
func (s *StateFile) Update() {

	// Update the properties in the local in-memory struct
	if _, ok := properties["last_start_ts"]; ok {
		s.properties.LastStartTS = properties["last_start_ts"].(int)
	}
	if _, ok := properties["last_end_ts"]; ok {
		s.properties.LastEndTS = properties["last_end_ts"].(int)
	}
	if _, ok := properties["last_count"]; ok {
		s.properties.LastCount = properties["last_count"].(int)
	}
	if _, ok := properties["last_request_ts"]; ok {
		s.properties.LastRequestTS = properties["last_request_ts"].(int)
	}
	s.lastUpdated = time.Now()
	// Persist the state in a background routine
	go s.save()
}
*/

func (s *StateFile) GetLastStartTS() int {
	return s.properties.LastStartTS
}

func (s *StateFile) GetLastEndTS() int {
	return s.properties.LastEndTS
}

func (s *StateFile) GetLastCount() int {
	return s.properties.LastCount
}

func (s *StateFile) GetLastRequestTS() int {
	return s.properties.LastRequestTS
}

func (s *StateFile) UpdateLastStartTS(ts int) {
	s.properties.LastStartTS = ts
}

func (s *StateFile) UpdateLastEndTS(ts int) {
	s.properties.LastEndTS = ts
}

func (s *StateFile) UpdateLastCount(count int) {
	s.properties.LastCount = count
}

func (s *StateFile) UpdateLastRequestTS(ts int) {
	s.properties.LastRequestTS = ts
}

func (s *StateFile) Save() error {

	s.lastUpdated = time.Now()

	// open file using READ & WRITE permission
	var file, err = os.OpenFile(s.FileName, os.O_RDWR, 0644)
	defer file.Close()
	if err != nil {
		return err
	}

	data, _ := json.Marshal(s.properties)
	_, err = file.WriteString(string(data))
	if err != nil {
		return err
	}

	// save changes
	err = file.Sync()
	if err != nil {
		return err
	}

	return nil
}
