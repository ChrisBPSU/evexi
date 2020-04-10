package evexi

import (
	"bytes"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
)

var fileDateFormat = "2006_01_02-15_04_05"

// ExportToS3 exports to the specified bucket, folders and fileNamePrefix are joined
func ExportToS3(sess *session.Session, bucket string, folders []string, fileNamePrefix string) func([]byte) {
	sb := strings.Builder{}
	for _, folder := range folders {
		sb.WriteString(folder)
		sb.WriteByte('/')
	}
	sb.WriteString(fileNamePrefix)

	prefix := sb.String()
	uploader := s3manager.NewUploader(sess)

	return func(logData []byte) {
		if len(logData) > 0 {
			logName := fmt.Sprintf("%s_%s.txt", prefix, time.Now().Format(fileDateFormat))
			_, err := uploader.Upload(&s3manager.UploadInput{
				Bucket: aws.String(bucket),
				Key:    aws.String(logName),
				Body:   bytes.NewBuffer(logData),
			})
			if err != nil {
				log.Println("Unable to upload to S3:", err.Error())
			}
		}
	}
}
