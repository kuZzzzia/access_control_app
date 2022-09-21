package service

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

var ErrorObjectNotFound = errors.New("object not found")

type Service struct {
}

type ImageInfo struct {
	ID        uuid.UUID
	CreatedAt time.Time

	PeopleNumber int

	Name        string
	ContentType string
	Extension   string
	Size        int64

	BucketName string
	UserID     uuid.UUID
}
