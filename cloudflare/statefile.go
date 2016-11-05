package cloudflare

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"time"
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

func (s *StateFile) Initialize() {
	// 2. If it doesn't exists, create it
	if s.StorageType == "disk" {
		s.loadFromDisk()
	}
}

func (s *StateFile) loadFromDisk() error {

	// Create it if it doesn't exist
	if _, err := os.Stat(s.FileName); os.IsNotExist(err) {
		var file, err = os.Create(s.FileName)
		defer file.Close()
		if err != nil {
			return err
		}
	}

	// Now load the file in memory
	sfData, err := ioutil.ReadFile(s.FileName)
	if err != nil {
		return err
	}

	var dat Properties
	if err := json.Unmarshal(sfData, &dat); err != nil {
		if err != nil {
			return err
		}
	}

	s.properties = dat

	// Return any errors
	return nil
}

func (s *StateFile) loadFromS3(bucketName string, fileName string, AWSAccessKey string, AWSSecretKey string) {
	// TODO
}

func (s *StateFile) loadFromConsul(filePath string) {
	// TODO
}

func (s *StateFile) Update(properties map[string]interface{}) {

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

func (s *StateFile) save() error {

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
