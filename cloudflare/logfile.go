package cloudflare

import (
	"errors"
	"io"
	"os"

	"github.com/elastic/beats/libbeat/logp"
	"github.com/franela/goreq"
)

type RequestLogFile struct {
	Filename string
}

func NewRequestLogFile(filename string) *RequestLogFile {
	return &RequestLogFile{
		Filename: filename,
	}
}

func (l *RequestLogFile) SaveFromHttpResponseBody(respBody *goreq.Body) (int64, error) {

	fh, err := os.Create(l.Filename)
	if err != nil {
		logp.Err("Error creating output file: %v", err)
		return 0, err
	}

	nBytes, err := io.Copy(fh, respBody)
	if err != nil {
		logp.Err("Error copying byte stream to %s: %v", l.Filename, err)
		return 0, err
	}
	if nBytes == 0 {
		l.Destroy()
	}
	fh.Close()

	if err != nil {
		return 0, err
	} else if nBytes == 0 {
		return 0, errors.New("Request body is empty")
	}

	return nBytes, nil
}

func (l *RequestLogFile) Destroy() {
	if err := os.Remove(l.Filename); err != nil {
		logp.Err("Could not delete local log file %s: %s", l.Filename, err.Error())
	}
}

func DeleteLogLife(filename string) {
	if err := os.Remove(filename); err != nil {
		logp.Err("Could not delete local log file %s: %s", filename, err.Error())
	}
}
