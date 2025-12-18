package upload

import (
	"context"
	"errors"
	"path/filepath"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type options struct {
	client          *minio.Client
	bucketName      string
	address         string
	accessKeyID     string
	secretAccessKey string
	token           string
	secure          bool
}

func WithS3Address(address string) func(*options) {
	return func(opts *options) {
		opts.address = address
	}
}

func WithS3AccessKeyID(accessKeyID string) func(*options) {
	return func(opts *options) {
		opts.accessKeyID = accessKeyID
	}
}

func WithS3SecretAccessKey(secretAccessKey string) func(*options) {
	return func(opts *options) {
		opts.secretAccessKey = secretAccessKey
	}
}

func WithS3Token(token string) func(*options) {
	return func(opts *options) {
		opts.token = token
	}
}

func WithS3BucketName(bucket string) func(*options) {
	return func(opts *options) {
		opts.bucketName = bucket
	}
}

func WithSecure(secure bool) func(*options) {
	return func(opts *options) {
		opts.secure = secure
	}
}

func NewS3Uploader(opts ...func(*options)) (*options, error) {
	o := new(options)

	for _, opt := range opts {
		opt(o)
	}
	if o.bucketName == "" {
		return nil, errors.New("s3 bucket name is required")
	}

	s3credentials := credentials.NewStaticV4(o.accessKeyID, o.secretAccessKey, o.token)

	s3Client, err := minio.New(o.address, &minio.Options{
		Creds:  s3credentials,
		Secure: o.secure,
	})
	if err != nil {
		return nil, err
	}
	o.client = s3Client

	return o, nil
}

// Upload uploads the file at filePath to S3.
func (f *options) Upload(ctx context.Context, filePath string, md5sum string) (any, error) {
	var options minio.PutObjectOptions
	if md5sum != "" {
		options.UserMetadata = map[string]string{"md5": md5sum}
	}
	// f.client.TraceOn(os.Stdout)
	// defer f.client.TraceOff()

	bucketName := f.bucketName
	objectName := filepath.Base(filePath)

	return f.client.FPutObject(ctx, bucketName, objectName, filePath, options)

}
