package cache

import (
	jsoniter "github.com/json-iterator/go"
)

const (
	FileNotUploaded int = iota
	FileUploaded
)

type FileChunk struct {
	UUID           string
	Md5            string
	IsUploaded     int
	UploadID       string
	TotalChunks    int
	Size           int64
	FileName       string
	CompletedParts string // chunkNumber+etag eg: ,1-asqwewqe21312312.2-123hjkas
}

// GetFileChunkByMD5 returns fileChunk by given md5
func GetFileChunkByMD5(md5 string) (*FileChunk, error) {
	fileChunk := new(FileChunk)
	data, err := Cache.Get(md5)
	if err != nil {
		return fileChunk, nil
	}
	if err = jsoniter.Unmarshal(data, &fileChunk); err != nil {
		return fileChunk, nil
	}
	return fileChunk, nil
}

// GetFileChunkByUUID returns attachment by given uuid
func GetFileChunkByUUID(uuid string) (*FileChunk, error) {
	fileChunk := new(FileChunk)
	data, err := Cache.Get(uuid)
	if err != nil {
		return fileChunk, nil
	}
	if err = jsoniter.Unmarshal(data, &fileChunk); err != nil {
		return fileChunk, nil
	}
	return fileChunk, nil
}

// InsertFileChunk insert a record into file_chunk.
func InsertFileChunk(fileChunk *FileChunk) (_ *FileChunk, err error) {
	data, _ := jsoniter.Marshal(fileChunk)
	err = Cache.Set(fileChunk.Md5, data)
	if err != nil {
		return nil, err
	}
	return fileChunk, nil
}

// UpdateFileChunk updates the given fileChunk in database
func UpdateFileChunk(fileChunk *FileChunk) error {
	var chunk *FileChunk
	data, err := Cache.Get(fileChunk.UUID)
	if err != nil {
		return nil
	}
	if err = jsoniter.Unmarshal(data, &chunk); err != nil {
		return nil
	}

	chunk.IsUploaded = fileChunk.IsUploaded
	chunk.CompletedParts = fileChunk.CompletedParts
	data, _ = jsoniter.Marshal(chunk)
	err = Cache.Set(fileChunk.UUID, data)
	if err != nil {
		return err
	}
	return nil
}
