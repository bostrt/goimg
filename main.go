package main

import (
	"sync"

	"github.com/boltdb/bolt"
)

func main() {
	var wg sync.WaitGroup
	db, _ := bolt.Open("test.db", 0600, nil)
	// TODO: Catch an error
	//    if err != nil {
	//		log.Fatal(err)
	//	}
	defer db.Close()

	dao := NewImageDao(db)
	fs := NewFS()
	gc := NewGC(db, dao, fs, &wg)
	go gc.Start()

	db.Update(func(tx *bolt.Tx) error {
		// Ensure "recent" and "gc" buckets are present.
		tx.CreateBucketIfNotExists(B(RECENT_BUCKET))
		tx.CreateBucketIfNotExists(B(EXPIRATION_BUCKET))
		tx.CreateBucketIfNotExists(B(IMAGE_BUCKET))

		return nil
	})

	NewServer(dao, fs, Config{}).ListenAndServe()

	wg.Wait()
}
