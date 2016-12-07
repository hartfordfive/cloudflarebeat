package cloudflare

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/elastic/beats/libbeat/logp"
)

const (
	FETCH_PERIOD_SECONDS = 1800
)

type StateFile struct {
	FileName    string
	FilePath    string
	ZoneName    string
	StorageType string
	properties  Properties
	lastUpdated time.Time
	s3settings  *awsS3Settings
}

type Properties struct {
	LastStartTS   int `json:"last_start_ts"`
	LastEndTS     int `json:"last_end_ts"`
	LastCount     int `json:"last_count"`
	LastRequestTS int `json:"last_request_ts"`
}

type awsS3Settings struct {
	awsAccesKey        string
	awsSecretAccessKey string
	s3BucketName       string
}

func (p *Properties) ToJsonBytes() []byte {
	b, _ := json.Marshal(p)
	return b
}

func NewStateFile(config map[string]string) (*StateFile, error) {

	sf := &StateFile{
		StorageType: config["storage_type"],
	}

	if sf.StorageType == "s3" {
		if _, ok := config["aws_access_key"]; !ok {
			return nil, errors.New("Must specify aws_access_key when using S3 storage.")
		}
		if _, ok := config["aws_secret_access_key"]; !ok {
			return nil, errors.New("Must specify aws_secret_access_key when using S3 storage.")
		}
		if _, ok := config["aws_s3_bucket_name"]; !ok {
			return nil, errors.New("Must specify aws_secret_access_key when using S3 storage.")
		}
		sf.s3settings = &awsS3Settings{config["aws_access_key"], config["aws_secret_access_key"], config["aws_s3_bucket_name"]}
	} else if _, ok := config["filepath"]; ok {
		sf.FilePath = config["filepath"]
	}

	if _, ok := config["zone_tag"]; ok {
		sf.FileName = config["filename"] + "-" + config["zone_tag"] + ".state"
	} else {
		sf.FileName = config["filename"] + ".state"
	}

	sf.initialize()
	return sf, nil
}

func (s *StateFile) initialize() error {
	// 2. If it doesn't exists, create it
	logp.Info("Initializing state file '%s' with storage type '%s'", s.FileName, s.StorageType)

	if s.StorageType == "disk" {
		return s.loadFromDisk()
	} else if s.StorageType == "s3" {
		return s.loadFromS3()
	}

	return errors.New("Unsupported storage type")
}

func (s *StateFile) initializeStateFileValues() {
	s.properties.LastStartTS = int(time.Now().UTC().Unix()) - FETCH_PERIOD_SECONDS - 1
	s.properties.LastEndTS = int(time.Now().UTC().Unix())
}

func (s *StateFile) loadFromDisk() error {

	sfName := filepath.Join(s.FilePath, s.FileName)

	// Create it if it doesn't exist
	if _, err := os.Stat(sfName); os.IsNotExist(err) {
		var file, err = os.Create(sfName)
		defer file.Close()
		if err != nil {
			return err
		}
		s.initializeStateFileValues()
		return nil
	}

	// Now load the file in memory
	sfData, err := ioutil.ReadFile(sfName)
	if err != nil {
		return err
	}

	var dat Properties
	if err := json.Unmarshal(sfData, &dat); err != nil {
		// If the state file isn't valid json, then re-create it
		if err != nil {
			logp.Err("%s", err)
			logp.Info("State file contents: %s", string(sfData))
			err = os.Remove(sfName)
			var file, err = os.Create(sfName)
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

func (s *StateFile) loadFromS3() error {

	svc, err := s.getAwsSession()
	if err != nil {
		return err
	}

	// 1. Check if the file exists and if not, create it
	// 2. Otherwise, fetch the object's contents, and store it in the local state instance
	params := &s3.GetObjectInput{
		Bucket: aws.String(s.s3settings.s3BucketName),
		Key:    aws.String(s.FileName),
	}
	resp, err := svc.GetObject(params)

	if err != nil && err.Error() == "NoSuchKey: The specified key does not exist." {
		// Create the file here as it doesn't exist
		s.initializeStateFileValues()
		//s.writeToS3(svc)
		return err
	} else if err != nil {
		return err
	}

	// File was successfully loaded.  Unmarshall into state attribute
	var p Properties
	if err := json.Unmarshal([]byte(fmt.Sprint(resp)), &p); err != nil {
		return err
	}

	s.properties = p
	return nil
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

	if s.StorageType == "disk" {
		return s.saveToDisk()
	} else if s.StorageType == "s3" {
		return s.saveToS3()
	}
	return nil
}

func (s *StateFile) saveToDisk() error {
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

func (s *StateFile) writeToS3(svc *s3.S3) (*s3.PutObjectOutput, error) {
	params := &s3.PutObjectInput{
		Bucket: aws.String(s.s3settings.s3BucketName), // Required
		Key:    aws.String(s.FileName),                // Required
		Body:   bytes.NewReader(s.properties.ToJsonBytes()),
	}
	return svc.PutObject(params)
}

func (s *StateFile) saveToS3() error {

	svc, err := s.getAwsSession()
	if err != nil {
		return err
	}
	_, err = s.writeToS3(svc)

	if err != nil {
		return err
	}

	return nil
}

func (s *StateFile) getAwsSession() (*s3.S3, error) {

	sess := session.New(&aws.Config{
		Region: aws.String("us-east-1"),
	})

	/*
		Or with debugging on:
		sess := session.New((&aws.Config{
			Region: aws.String("us-east-1"),
		}).WithLogLevel(aws.LogDebugWithRequestRetries | aws.LogDebugWithRequestErrors))
	*/

	token := ""
	creds := credentials.NewStaticCredentials(s.s3settings.awsAccesKey, s.s3settings.awsSecretAccessKey, token)
	_, err := creds.Get()
	if err != nil {
		logp.Err("AWS Credentials Error: %v", err)
		return nil, err
	}

	svc := s3.New(sess, &aws.Config{
		Region:      aws.String("us-east-1"),
		Credentials: creds,
	})

	return svc, nil
}
