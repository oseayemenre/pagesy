package main

import (
	"context"
	"fmt"
	"io"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/cloudinary/cloudinary-go/v2"
	"github.com/cloudinary/cloudinary-go/v2/api/uploader"
)

type objectStore interface {
	upload(ctx context.Context, key string, body io.Reader) (string, error)
}

type s3Object struct {
	client *s3.Client
}

type cloudinaryObject struct {
	client *cloudinary.Cloudinary
}

func news3Object(client *s3.Client) *s3Object {
	return &s3Object{client}
}

func newcloudinaryObject(client *cloudinary.Cloudinary) *cloudinaryObject {
	return &cloudinaryObject{client}
}

func (o *s3Object) upload(ctx context.Context, key string, body io.Reader) (string, error) {
	_, err := o.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String("pagesy"),
		Key:    aws.String(key),
		Body:   body,
	})

	if err != nil {
		return "", fmt.Errorf("error uploading object to s3, %v", err)
	}
	return "", nil
}

func (o *cloudinaryObject) upload(ctx context.Context, key string, body io.Reader) (string, error) {
	resp, err := o.client.Upload.Upload(context.Background(), body, uploader.UploadParams{PublicID: key, Folder: "pagesy"})

	if err != nil {
		return "", fmt.Errorf("error uploading file, %+v", err)
	}

	return resp.SecureURL, nil
}
