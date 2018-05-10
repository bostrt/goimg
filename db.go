package main

import (
	"bytes"
	"fmt"
	"strconv"

	"github.com/boltdb/bolt"
)

const (
	IMAGE_BUCKET      string = "images"
	EXPIRATION_BUCKET string = "expiration"
	RECENT_BUCKET     string = "recent"
	RECENT_LIMIT      int    = 5
)

type ImageDao struct {
	db *bolt.DB
}

func NewImageDao(db *bolt.DB) *ImageDao {
	return &ImageDao{
		db: db,
	}
}

func (dao *ImageDao) Save(image *Image) error {
	err := dao.db.Update(func(tx *bolt.Tx) error {
		var bucket *bolt.Bucket

		if !image.Unlisted {
			// Add image to public listing
			bucket = tx.Bucket(B(RECENT_BUCKET))
			id, err := bucket.NextSequence()
			if err != nil {
				return err
			}
			bucket.Put(itob(int(id)), B(image.UUID))
			image.RecentKey = itob(int(id))
		}

		if image.Expires != "" {
			// Get today
			bucket = tx.Bucket(B(EXPIRATION_BUCKET))
			today := bucket.Get(B(image.Expires))
			if today == nil || string(today) == "" {
				// We're creating a new entry
				today = B(image.UUID)
			} else {
				// Append to existing entry
				today = append(today, ',')
				today = append(today, B(image.UUID)...)
			}
			bucket.Put(B(image.Expires), today)
		}

		//bucket, err = tx.CreateBucketb([]byte(image.UUID))
		bucket = tx.Bucket(B(IMAGE_BUCKET))
		bucket.Put(B(image.UUID+":path"), B(image.path))
		bucket.Put(B(image.UUID+":thumbpath"), B(image.thumbPath))
		bucket.Put(B(image.UUID+":added"), B(image.Added))
		bucket.Put(B(image.UUID+":expires"), B(image.Expires))
		bucket.Put(B(image.UUID+":delete"), B(image.Delete))
		bucket.Put(B(image.UUID+":unlisted"), B(strconv.FormatBool(image.Unlisted)))
		bucket.Put(B(image.UUID+":cookie"), B(image.cookie))
		bucket.Put(B(image.UUID+":owner"), B(image.Owner))
		bucket.Put(B(image.UUID+":recentkey"), image.RecentKey)

		return nil
	})

	return err
}

func (dao *ImageDao) Load(UUID string) (*Image, error) {
	var image *Image
	err := dao.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(B(IMAGE_BUCKET))
		if bucket == nil {
			return nil
		}
		image = dao.BucketToImage(UUID, bucket)
		return nil
	})

	return image, err
}

func (dao *ImageDao) DeleteWithTx(image *Image, tx *bolt.Tx) error {
	imageBucket := tx.Bucket(B(IMAGE_BUCKET))
	c := imageBucket.Cursor()
	k, _ := c.Seek(B(image.UUID))
	if k == nil {
		return fmt.Errorf("Error locating image: %s", image.UUID)
	}

	for ; k != nil && bytes.HasPrefix(k, B(image.UUID)); k, _ = c.Next() {
		// Delete image keys
		c.Delete()
	}

	recent := tx.Bucket(B(RECENT_BUCKET))
	recent.Delete(image.RecentKey)
	return nil
}

func (dao *ImageDao) Delete(image *Image) error {
	err := dao.db.Update(func(tx *bolt.Tx) error {
		return dao.DeleteWithTx(image, tx)
	})
	return err
}

func (dao *ImageDao) ListRecent() []string {
	recent := make([]string, RECENT_LIMIT)
	dao.db.View(func(tx *bolt.Tx) error {
		var index = 0

		recentBucket := tx.Bucket(B(RECENT_BUCKET))
		c := recentBucket.Cursor()

		for k, v := c.Last(); index < RECENT_LIMIT; k, v = c.Prev() {
			if k == nil {
				index++
			}
			if k != nil && v != nil {
				recent[index] = string(v)
				index++
			}
		}
		return nil
	})
	return recent
}

func (dao *ImageDao) BucketToImage(UUID string, bucket *bolt.Bucket) *Image {
	unlisted, err := strconv.ParseBool(string(bucket.Get(B(UUID + ":unlisted"))))
	if err != nil {
		unlisted = true // better safe than sorry
	}
	image := &Image{
		UUID:      UUID,
		path:      string(bucket.Get(B(UUID + ":path"))),
		thumbPath: string(bucket.Get(B(UUID + ":thumbpath"))),
		Added:     string(bucket.Get(B(UUID + ":added"))),
		Unlisted:  unlisted,
		Expires:   string(bucket.Get(B(UUID + ":expires"))),
		Delete:    string(bucket.Get(B(UUID + ":delete"))),
		cookie:    string(bucket.Get(B(UUID + ":cookie"))),
		Owner:     string(bucket.Get(B(UUID + ":owner"))),
		RecentKey: bucket.Get(B(UUID + ":recentkey")),
	}

	return image
}

func B(s string) []byte {
	return []byte(s)
}
