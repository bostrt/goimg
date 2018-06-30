package main

import (
	"bytes"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/corona10/goimghdr"
	"github.com/disintegration/imaging"
	"github.com/unrolled/logger"
)

type FS struct {
	cfg    Config
	logger *logger.Logger
}

func NewFS(cfg Config, logger *logger.Logger) *FS {
	return &FS{
		cfg:    cfg,
		logger: logger,
	}
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
		fs.logger.Println(err)
		return "", ""
	}

	// create file on disk
	newPath := filepath.Join(cfg.data, id+"."+fileType)
	newFile, err := os.Create(newPath)
	if err != nil {
		fs.logger.Println(err)
		return "", ""
	}
	defer newFile.Close()

	// write file to disk
	if _, err = newFile.Write(fileBytes); err != nil {
		fs.logger.Println(err)
		return "", ""
	}

	// create thumbnail file on disk
	thumbPath := filepath.Join(cfg.data, id+"_thumb."+fileType)
	fs.logger.Printf("New image upload: %s %s\n", newPath, thumbPath)
	thumbFile, err := os.Create(thumbPath)
	if err != nil {
		fs.logger.Println(err)
		// TODO Clean up original image if thumbnail cannot be generated.
		return "", ""
	}
	defer thumbFile.Close()

	// create image reader
	reader := bytes.NewReader(fileBytes)
	imageObj, err := imaging.Decode(reader)
	if err != nil {
		fs.logger.Println("Error decoding: ", err)
		// TODO Clean up thumbnail and original upon failure.
		return "", ""
	}

	// Make thumbnail
	thumbnailImage := imaging.Fit(imageObj, 500, 500, imaging.Lanczos)

	// Save to disk
	err = imaging.Save(thumbnailImage, thumbPath)
	if err != nil {
		fs.logger.Println("Error saving: ", err)
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
	fs.logger.Println("Deleted image: ", image.UUID)

	err = fs.DeleteThumbnail(image)
	if err != nil {
		return err
	}

	return nil
}

func (fs *FS) DeleteThumbnail(image *Image) error {
	err := os.Remove(image.thumbPath)
	if err != nil {
		return err
	}
	fs.logger.Println("Deleted thumbnail: ", image.UUID)
	return nil
}

// Ensure returns two booleans. First is true if original image is
// present on disk. Second boolean is true if thumbnail is present
// on disk.
func (fs *FS) Ensure(image *Image) (bool, bool) {
	orig, thumb := true, true
	if _, err := os.Stat(image.path); os.IsNotExist(err) {
		orig = false
	}
	if _, err := os.Stat(image.thumbPath); os.IsNotExist(err) {
		thumb = false
	}
	return orig, thumb
}
