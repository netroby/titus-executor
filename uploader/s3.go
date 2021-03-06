package uploader

import (
	"errors"
	"os"
	"path"

	"io"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/ec2metadata"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/sirupsen/logrus"
)

const (
	defaultS3ContentType = "text/plain"
	defaultS3ACL         = s3.ObjectCannedACLBucketOwnerFullControl
	defaultS3PartSize    = 64 * 1024 * 1024 // 64MB per part
)

// S3Uploader uploads logs to S3
type S3Uploader struct {
	log        logrus.FieldLogger
	bucketName string
	s3Uploader *s3manager.Uploader
}

// NewS3Uploader creates a new instance of an S3 uploader
func NewS3Uploader(log logrus.FieldLogger, bucket string) Uploader {
	region, err := getEC2Region()
	if err != nil {
		panic(err)
	}

	u := &S3Uploader{
		log:        log,
		bucketName: bucket,
	}

	session, err := session.NewSession(&aws.Config{
		Logger: &logAdapter{log},
		Region: &region,
	})
	if err != nil {
		panic(err)
	}
	u.s3Uploader = s3manager.NewUploader(session, func(u *s3manager.Uploader) {
		u.PartSize = defaultS3PartSize
	})

	return u
}

func getEC2Region() (string, error) {
	if region := os.Getenv("EC2_REGION"); region != "" {
		return region, nil
	}

	sess := session.Must(session.NewSession())
	ec2metadatasvc := ec2metadata.New(sess)
	if !ec2metadatasvc.Available() {
		return "", errors.New("Unable to determine EC2 Region, and EC2 metadata service unavailable")
	}
	return ec2metadatasvc.Region()
}

// Upload writes a single file only to S3!
func (u *S3Uploader) Upload(local string, remote string, ctypeFunc ContentTypeInferenceFunction) error {
	u.log.Printf("Attempting to upload file from: %s to: %s", local, path.Join(u.bucketName, remote))

	f, err := os.Open(local)
	if err != nil {
		return err
	}
	contentType := ctypeFunc(local)
	if contentType == "" {
		contentType = defaultS3ContentType
	}
	defer func() {
		if err = f.Close(); err != nil {
			u.log.Printf("Failed to close %s: %s", f.Name(), err)
		}
	}()

	return u.uploadFile(f, remote, contentType)
}

// UploadFile writes a single file only to S3!
func (u *S3Uploader) uploadFile(local io.Reader, remote string, contentType string) error {
	u.log.Printf("Attempting to upload file from: %s to: %s", local, path.Join(u.bucketName, remote))
	if contentType == "" {
		contentType = defaultS3ContentType
	}

	result, err := u.s3Uploader.Upload(&s3manager.UploadInput{
		ACL:         aws.String(defaultS3ACL),
		ContentType: aws.String(contentType),
		Bucket:      aws.String(u.bucketName),
		Key:         aws.String(remote),
		Body:        local,
	})
	if err != nil {
		return err
	}

	u.log.Printf("Successfully uploaded file from: %s to: %s", local, result.Location)

	return nil
}

// UploadPartOfFile copies a single file only. It doesn't preserve the cursor location in the file.
func (u *S3Uploader) UploadPartOfFile(local io.ReadSeeker, start, length int64, remote, contentType string) error {
	if _, err := local.Seek(start, io.SeekStart); err != nil {
		return err
	}
	if contentType == "" {
		contentType = defaultS3ContentType
	}
	limitLocal := io.LimitReader(local, length)
	return u.uploadFile(limitLocal, remote, contentType)
}

type logAdapter struct {
	log logrus.StdLogger
}

func (a *logAdapter) Log(args ...interface{}) {
	a.log.Print(args...)
}
