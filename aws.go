package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
)

// setup a new aws session
func awsSession(pub, priv string) *session.Session {
	sess, err := session.NewSession(&aws.Config{
		Region:      aws.String("ca-central-1"),
		Credentials: credentials.NewStaticCredentials(pub, priv, ""),
	})
	if err != nil {
		// TODO
		panic(err)
	}
	return sess
}

// download txDir/name from s3 bucket and save it to $PWD/name
func awsDownload(sess *session.Session, bucketName, txDir, name string) (*os.File, error) {

	file, err := os.Create(name)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	downloader := s3manager.NewDownloader(sess)

	numBytes, err := downloader.Download(file,
		&s3.GetObjectInput{
			Bucket: aws.String(bucketName),
			Key:    aws.String(filepath.Join(txDir, name)),
		})
	if err != nil {
		return nil, err
	}
	_ = numBytes

	return file, nil
}

// upload the dataBytes to txDir/name in the s3 bucket
func awsUpload(sess *session.Session, bucketName, txDir, name string, dataBytes []byte) error {
	uploader := s3manager.NewUploader(sess)
	_, err := uploader.Upload(&s3manager.UploadInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(filepath.Join(txDir, name)),
		Body:   bytes.NewBuffer(dataBytes),
	})

	return err
}

// make a directory (ie. an empty file) called dirName in the bucket
func awsMkdir(sess *session.Session, bucketName, dirName string) error {
	svc := s3.New(sess)
	_, err := svc.PutObject(&s3.PutObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(dirName),
	})
	return err
}

// delete an object from the bucket
func awsDelete(sess *session.Session, bucketName, objName string) error {
	svc := s3.New(sess)
	_, err := svc.DeleteObject(&s3.DeleteObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(objName),
	})
	return err
}

// download all files in the dir and return list of file names
func fetchFilesInDir(sess *session.Session, bucketName, dirPath string) ([]string, error) {
	svc := s3.New(sess)

	// list all items in bucket
	resp, err := svc.ListObjectsV2(&s3.ListObjectsV2Input{Bucket: aws.String(bucketName)})
	if err != nil {
		return nil, err
	}

	dirPath = strings.TrimSuffix(dirPath, "/")

	// get only those in our folder
	files := []string{}
	for _, item := range resp.Contents {
		key := *item.Key
		keyDir := filepath.Dir(key)
		keyBase := strings.TrimPrefix(key, keyDir)
		keyBase = strings.TrimPrefix(keyBase, "/")
		if keyDir == dirPath && keyBase != "" {
			files = append(files, keyBase)
		}
	}

	if len(files) == 0 {
		return []string{}, nil
	}

	for _, f := range files {
		// download all files in folder
		_, err := awsDownload(sess, bucketName, dirPath, f)
		if err != nil {
			return nil, err
		}
	}
	return files, nil
}