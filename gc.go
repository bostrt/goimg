package main

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/boltdb/bolt"
)

const (
	RUN int = 1 + iota
	STOP
)

var (
	cmd = make(chan int)
)

type GC struct {
	db  *bolt.DB
	dao *ImageDao
	fs  *FS
	wg  *sync.WaitGroup
	cfg *Config
}

func NewGC(db *bolt.DB, dao *ImageDao, fs *FS, wg *sync.WaitGroup) *GC {
	return &GC{
		db:  db,
		dao: dao,
		fs:  fs,
		wg:  wg,
	}
}

func (gc *GC) Start() {
	//https://golang.org/pkg/time/#Ticker
	gc.wg.Add(1)
	fmt.Println("GC Started")
	ticker := time.NewTicker(time.Duration(cfg.gcInterval) * time.Second)
	for {
		select {
		case <-ticker.C:
			fmt.Println("Tick tick")
			gc.do()
			break
		case msg := <-cmd:
			if msg == RUN {
				gc.do()
			} else if msg == STOP {
				gc.wg.Done()
				fmt.Println("GC Shut down")
				return
			}
			break
		}
	}
}

func (gc *GC) do() {
	fmt.Println("GC do")
	gc.db.Update(func(tx *bolt.Tx) error {
		gc.doGCRecent(tx)
		gc.doGCExpired(tx)
		return nil
	})
}

func (gc *GC) Stop() {
	cmd <- STOP
}

func (gc *GC) doGCRecent(tx *bolt.Tx) {
	index := 0
	bucket := tx.Bucket(B(RECENT_BUCKET))
	c := bucket.Cursor()
	var k, v []byte

	// Skip ahead by RECENT_LIMIT and start deleting
	for k, v = c.Last(); index < RECENT_LIMIT; k, v = c.Prev() {
		index++
	}

	for ; k != nil; k, v = c.Prev() {
		// Remove from recent bucket
		bucket.Delete(k)
		image, err := gc.dao.Load(string(v))
		if image != nil && err == nil {
			// Delete thumbnail used with the "recent" view
			gc.fs.DeleteThumbnail(image)
		} else {
			fmt.Println("Error during GC", err)
		}
	}
}

func (gc *GC) doGCExpired(tx *bolt.Tx) {
	counter := 0
	now := time.Now().UTC().Format(time.RFC3339)
	bucket := tx.Bucket(B(EXPIRATION_BUCKET))
	c := bucket.Cursor()
	for k, v := c.First(); k != nil && counter < cfg.gcLimit && string(k) < now; k, v = c.Next() {
		// Split string on comma
		uuids := strings.Split(string(v), ",")
		for _, uuid := range uuids {
			image, err := gc.dao.Load(string(uuid))
			if err != nil || image == nil {
				fmt.Printf("Error loading image for GC. Deleting entry [UUID:%s]\n", string(uuid))
				c.Delete()
			} else {
				fmt.Printf("GC Expired image [UUID:%s]\n", string(uuid))
				c.Delete()
				gc.dao.DeleteWithTx(image, tx)
				err := gc.fs.Delete(image)
				if err != nil {
					fmt.Printf("Error deleting image from disk for GC: %s\n", err)
				}
			}
		}
	}
}
