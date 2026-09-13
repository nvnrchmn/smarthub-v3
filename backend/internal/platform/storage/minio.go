// Package storage — penyimpanan objek privat (MinIO) untuk dokumen KTP & KK.
// Bucket PII tidak pernah dibuat publik; berkas hanya keluar lewat API setelah
// pemeriksaan izin dan pencatatan audit.
package storage

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type Store struct {
	cl     *minio.Client
	bucket string
}

func New(endpoint, access, secret, bucket string) (*Store, error) {
	cl, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(access, secret, ""),
		Secure: false,
	})
	if err != nil {
		return nil, fmt.Errorf("minio: %w", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ok, err := cl.BucketExists(ctx, bucket)
	if err != nil {
		return nil, fmt.Errorf("minio bucket: %w", err)
	}
	if !ok {
		if err := cl.MakeBucket(ctx, bucket, minio.MakeBucketOptions{}); err != nil {
			return nil, fmt.Errorf("minio buat bucket: %w", err)
		}
	}
	return &Store{cl: cl, bucket: bucket}, nil
}

// Put — menyimpan berkas (dipakai untuk unggahan KTP/KK).
func (s *Store) Put(ctx context.Context, key string, r io.Reader, size int64, contentType string) error {
	_, err := s.cl.PutObject(ctx, s.bucket, key, r, size, minio.PutObjectOptions{ContentType: contentType})
	return err
}

// Get — mengambil berkas untuk diteruskan ke pengurus yang berhak.
func (s *Store) Get(ctx context.Context, key string) (io.ReadCloser, int64, string, error) {
	obj, err := s.cl.GetObject(ctx, s.bucket, key, minio.GetObjectOptions{})
	if err != nil {
		return nil, 0, "", err
	}
	st, err := obj.Stat()
	if err != nil {
		return nil, 0, "", err
	}
	return obj, st.Size, st.ContentType, nil
}
