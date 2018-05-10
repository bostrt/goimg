package main

import (
	"time"
)

type Image struct {
	UUID      string
	path      string
	thumbPath string
	Added     string // RFC3339
	Unlisted  bool
	Expires   string // RFC3339
	Delete    string
	Owner     string
	cookie    string
	RecentKey []byte
}

func NewImage(owner string, UUID string, path string, thumbPath string, unlisted bool, expire string, delete string, cookie string) *Image {
	now := time.Now()
	var eternity bool = false
	switch expire {
	case "day":
		now = now.AddDate(0, 0, 1)
		break
	case "month":
		now = now.AddDate(0, 1, 0)
		break
	case "forever":
		eternity = true
	}
	image := &Image{
		Owner:     owner,
		UUID:      UUID,
		path:      path,
		thumbPath: thumbPath,
		Added:     time.Now().UTC().Format(time.RFC3339),
		Unlisted:  unlisted,
		Delete:    delete,
		cookie:    cookie,
	}

	if !eternity {
		image.Expires = now.UTC().Format(time.RFC3339)
	}

	return image
}
