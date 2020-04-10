package evexi

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"time"
)

// ExportToDisk exports to the specified path, else the current directory
func ExportToDisk(path string, fileNamePrefix string) (func([]byte), error) {
	dir := path
	if path == "" {
		var err error
		dir, err = os.Getwd()
		if err != nil {
			return nil, err
		}
	}

	return func(logData []byte) {
		if len(logData) > 0 {
			logPath := fmt.Sprintf("%s%s%s_%s.txt", dir, string(os.PathSeparator), fileNamePrefix, time.Now().Format(fileDateFormat))
			err := ioutil.WriteFile(logPath, logData, 0644)
			if err != nil {
				log.Println("Unable to export to disk:", err.Error())
			}
		}
	}, nil
}

// MustExportToDisk panics if os.Getwd() fails
func MustExportToDisk(path string, fileNamePrefix string) func([]byte) {
	fn, err := ExportToDisk(path, fileNamePrefix)
	if err != nil {
		panic(err)
	}

	return fn
}
