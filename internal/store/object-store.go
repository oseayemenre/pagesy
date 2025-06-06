package store

import (
	"context"
	"fmt"
	"io"

	"github.com/cloudinary/cloudinary-go/v2"
	"github.com/cloudinary/cloudinary-go/v2/api/uploader"
)

type ObjectStore interface {
	UploadFile(ctx context.Context, file io.Reader, id string) (string, error)
}

type CloudinaryStore struct {
	store *cloudinary.Cloudinary
}

func NewCloudinaryStore(store *cloudinary.Cloudinary) *CloudinaryStore {
	return &CloudinaryStore{
		store: store,
	}
}

func (s *CloudinaryStore) UploadFile(ctx context.Context, file io.Reader, id string) (string, error) {
	resp, err := s.store.Upload.Upload(ctx, file, uploader.UploadParams{PublicID: id})

	if err != nil {
		return "", fmt.Errorf("error uploading file: %+v", err)
	}

	fmt.Printf("cloudinary_response:%+v", resp)

	return resp.SecureURL, nil
}
