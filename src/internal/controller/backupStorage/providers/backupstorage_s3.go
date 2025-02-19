/*
Copyright 2023.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package backupstorageproviders

import (
	"context"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strconv"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"k8s.io/utils/ptr"

	backupoperatoriov1 "backup-operator.io/api/v1"
)

// S3Storage is an implementation of BackupStorage for Amazon S3.
type S3Storage struct {
	Endpoint         string
	Bucket           string
	Region           string `default:"us-east-1"`
	Insecure         bool   `default:"false"`
	S3ForcePathStyle bool   `default:"false"`

	credentials *credentials.Credentials
	session     *session.Session
	s3svc       *s3.S3
	uploader    *s3manager.Uploader
	// Underlying Kubernetes object
	object *backupoperatoriov1.BackupStorage
}

// Constructor Configure S3
func (s *S3Storage) Constructor(object *backupoperatoriov1.BackupStorage, parameters map[string]string, creds map[string]string) (err error) {
	// Set object field
	if object == nil {
		return errors.New("object is mandatory but nil")
	}
	s.object = object
	// Check parameters have all mandatory keys
	if _, ok := parameters["bucket"]; !ok {
		return errors.New("mandatory bucket parameter is not defined")
	}
	// Try to load access and secret keys from credentials
	func() {
		var accessKey, secretKey string
		var ok bool
		if accessKey, ok = parameters["accessKey"]; !ok {
			accessKey = "AWS_ACCESS_KEY_ID"
		}
		if secretKey, ok = parameters["secretKey"]; !ok {
			secretKey = "AWS_SECRET_ACCESS_KEY"
		}
		// Create credentials
		if accessKey, ok = creds[accessKey]; !ok {
			return
		}
		if secretKey, ok = creds[secretKey]; !ok {
			return
		}
		s.credentials = credentials.NewStaticCredentials(accessKey, secretKey, "")
	}()
	// Parse other parameters
	for key, value := range parameters {
		switch key {
		case "endpoint":
			pattern := "^https?://.+:[0-9]+$"
			var regex *regexp.Regexp
			if regex, err = regexp.Compile(pattern); err != nil {
				return fmt.Errorf("failed to compile a regex: %s", err)
			}
			if regex.MatchString(value) {
				s.Endpoint = value
			} else {
				return fmt.Errorf("endpoint does not match the regex: %s", pattern)
			}
		case "bucket":
			s.Bucket = value
		case "region":
			s.Region = value
		case "insecure":
			var b bool
			if b, err = strconv.ParseBool(value); err != nil {
				return
			}
			s.Insecure = b
		case "s3ForcePathStyle":
			var b bool
			if b, err = strconv.ParseBool(value); err != nil {
				return
			}
			s.S3ForcePathStyle = b
		}
	}
	// Create a new AWS session
	config := &aws.Config{
		Endpoint:         ptr.To[string](s.Endpoint),
		Region:           ptr.To[string](s.Region),
		DisableSSL:       ptr.To[bool](s.Insecure),
		S3ForcePathStyle: ptr.To[bool](s.S3ForcePathStyle),
		Credentials:      s.credentials,
	}
	if s.session, err = session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
		Config:            *config,
	}); err != nil {
		return fmt.Errorf("failed to make a S3 session: %s", err)
	}
	// Create an S3 service client
	s.s3svc = s3.New(s.session)
	// Create new uploader
	s.uploader = s3manager.NewUploader(s.session)
	// Check connection
	if _, err = s.s3svc.ListObjects(&s3.ListObjectsInput{
		Bucket: &s.Bucket,
	}); err != nil {
		return fmt.Errorf("failed to test S3 connection: %s", err)
	}
	return
}

// Destructor S3 storage object destructor.
func (s *S3Storage) Destructor() error {
	return nil
}

// Put Upload file.
func (s *S3Storage) Put(ctx context.Context, path string, reader io.Reader) error {
	// Upload the file to S3/MinIO bucket
	_, err := s.uploader.UploadWithContext(ctx, &s3manager.UploadInput{
		Bucket: &s.Bucket,
		Key:    &path,
		Body:   reader,
		// ContentType: aws.String("application/octet-stream"),
	})
	return err
}

// Get Download file.
func (s *S3Storage) Get(ctx context.Context, path string) (io.ReadCloser, error) {
	output, err := s.s3svc.GetObjectWithContext(ctx, &s3.GetObjectInput{
		Bucket: &s.Bucket,
		Key:    &path,
	})
	return output.Body, err
}

// List path.
func (s *S3Storage) List(ctx context.Context, path string) ([]string, error) {
	output, err := s.s3svc.ListObjectsWithContext(ctx, &s3.ListObjectsInput{
		Bucket: &s.Bucket,
		Prefix: &path,
	})
	var list []string
	for _, object := range output.Contents {
		list = append(list, *object.Key)
	}
	return list, err
}

// Delete Remove path
func (s *S3Storage) Delete(ctx context.Context, path string) error {
	_, err := s.s3svc.DeleteObjectWithContext(ctx, &s3.DeleteObjectInput{
		Bucket: &s.Bucket,
		Key:    &path,
	})
	return err
}

// Get underlying Kubernetes object
func (s *S3Storage) GetObject() *backupoperatoriov1.BackupStorage {
	return s.object
}

// Get underlying Kubernetes object
func (s *S3Storage) GetSize(ctx context.Context, path string) (size uint, err error) {
	var head *s3.HeadObjectOutput
	if head, err = s.s3svc.HeadObjectWithContext(ctx, &s3.HeadObjectInput{
		Bucket: &s.Bucket,
		Key:    &path,
	}); err != nil {
		return
	}
	if *head.ContentLength < 0 {
		size = 0
	} else {
		size = uint(*head.ContentLength)
	}
	return
}
