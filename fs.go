package main

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/corona10/goimghdr"
	"github.com/disintegration/imaging"
)

type FS struct {
	cfg Config
}

func NewFS(cfg Config) *FS {
	return &FS{cfg: cfg}
}

// Save saves a file to disk using the provided id as a name.
// Returns two strings: the image path and the thumbnail path.
// Returns two empty strings upon failure.
func (fs *FS) Save(file io.Reader, id string) (string, string) {
	// read in file bytes for later
	fileBytes, err := ioutil.ReadAll(file)
	if err != nil {
		return "", ""
	}

	// validate file type
	fileType, err := goimghdr.WhatFromReader(bytes.NewReader(fileBytes))
	if fileType == "" || err != nil {
		fmt.Println(err)
		return "", ""
	}

	// create file on disk
	newPath := filepath.Join(cfg.data, id)
	newFile, err := os.Create(newPath)
	if err != nil {
		fmt.Println(err)
		return "", ""
	}
	defer newFile.Close()

	// write file to disk
	if _, err = newFile.Write(fileBytes); err != nil {
		fmt.Println(err)
		return "", ""
	}

	// create thumbnail file on disk
	thumbPath := filepath.Join(cfg.data, id+"_thumb."+fileType)
	fmt.Printf("FileType: %s, File: %s, Thumb: %s\n", fileType, newPath, thumbPath)
	thumbFile, err := os.Create(thumbPath)
	if err != nil {
		fmt.Println(err)
		// TODO Clean up original image if thumbnail cannot be generated.
		return "", ""
	}
	defer thumbFile.Close()

	// create image reader
	reader := bytes.NewReader(fileBytes)
	imageObj, err := imaging.Decode(reader)
	if err != nil {
		fmt.Println("Error decoding: ", err)
		// TODO Clean up thumbnail and original upon failure.
		return "", ""
	}

	// Make thumbnail
	thumbnailImage := imaging.Fit(imageObj, 500, 500, imaging.Lanczos)

	// Save to disk
	err = imaging.Save(thumbnailImage, thumbPath)
	if err != nil {
		fmt.Println("Error saving: ", err)
		// TODO Clean up thumbnail and original upon failure.
		return "", ""
	}

	return newPath, thumbPath
}

func (fs *FS) Delete(image *Image) error {
	err := os.Remove(image.path)
	if err != nil {
		return err
	}
	fs.DeleteThumbnail(image)
	return nil
}

func (fs *FS) DeleteThumbnail(image *Image) {
	err := os.Remove(image.thumbPath)
	if err != nil {
		//fmt.Printf("Error deleting thumbnail [%s]: %s\n", image.thumbPath, err)
	}
}
