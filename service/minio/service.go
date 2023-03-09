package minio

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"oss/cache"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/minio/minio-go/pkg/s3utils"
	miniov7 "github.com/minio/minio-go/v7"
	"oss/config"
	logger "oss/lib/log"
	"oss/lib/minio_ext"
)

const (
	PresignedUploadPartUrlExpireTime = time.Hour * 24 * 7
)

type ComplPart struct {
	PartNumber int    `json:"partNumber"`
	ETag       string `json:"eTag"`
}

type CompleteParts struct {
	Data []ComplPart `json:"completedParts"`
}

// completedParts is a collection of parts sortable by their part numbers.
// used for sorting the uploaded parts before completing the multipart request.
type completedParts []miniov7.CompletePart

func (a completedParts) Len() int           { return len(a) }
func (a completedParts) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a completedParts) Less(i, j int) bool { return a[i].PartNumber < a[j].PartNumber }

// completeMultipartUpload container for completing multipart upload.
type completeMultipartUpload struct {
	XMLName xml.Name               `xml:"http://s3.amazonaws.com/doc/2006-03-01/ CompleteMultipartUpload" json:"-"`
	Parts   []miniov7.CompletePart `xml:"Part"`
}

func NewMultipart(ctx *gin.Context) {
	var uploadID string
	// 以文件hash值作为文件名
	objectName := ctx.Query("md5")
	totalChunkCounts, err := strconv.Atoi(ctx.Query("totalChunkCounts"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, "totalChunkCounts is illegal.")
		return
	}

	if totalChunkCounts > minio_ext.MaxPartsCount || totalChunkCounts <= 0 {
		ctx.JSON(http.StatusBadRequest, "totalChunkCounts is illegal.")
		return
	}

	fileSize, err := strconv.ParseInt(ctx.Query("size"), 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, "size is illegal.")
		return
	}

	if fileSize > minio_ext.MaxMultipartPutObjectSize || fileSize <= 0 {
		ctx.JSON(http.StatusBadRequest, "size is illegal.")
		return
	}

	//uuid = gouuid.NewV4().String()
	uploadID, err = newMultiPartUpload(ctx, objectName)
	if err != nil {
		fmt.Printf("newMultiPartUpload failed: %s", err.Error())
		ctx.JSON(http.StatusInternalServerError, "newMultiPartUpload failed.")
		return
	}

	_, err = cache.InsertFileChunk(&cache.FileChunk{
		UUID:        objectName,
		UploadID:    uploadID,
		Md5:         ctx.Query("md5"),
		Size:        fileSize,
		FileName:    ctx.Query("fileName"),
		TotalChunks: totalChunkCounts,
	})

	if err != nil {
		fmt.Println("InsertFileChunk failed:", err.Error())
		ctx.JSON(http.StatusInternalServerError, "InsertFileChunk failed.")
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"uuid":     objectName,
		"uploadID": uploadID,
	})
}

func GetMultipartUploadUrl(ctx *gin.Context) {
	var url string
	uuid := ctx.Query("uuid")
	uploadID := ctx.Query("uploadID")

	partNumber, err := strconv.Atoi(ctx.Query("chunkNumber"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, "chunkNumber is illegal.")
		return
	}

	size, err := strconv.ParseInt(ctx.Query("size"), 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, "size is illegal.")
		return
	}
	if size > minio_ext.MinPartSize {
		ctx.JSON(http.StatusBadRequest, "size is illegal.")
		return
	}

	url, err = genMultiPartSignedUrl(uuid, uploadID, partNumber, size)
	if err != nil {
		fmt.Println("genMultiPartSignedUrl failed:", err.Error())
		ctx.JSON(http.StatusInternalServerError, "genMultiPartSignedUrl failed.")
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"url": url,
	})
}

func CompleteMultipart(ctx *gin.Context) {
	uuid := ctx.PostForm("uuid")
	uploadID := ctx.PostForm("uploadID")

	//fileChunk, err := models.GetFileChunkByUUID(uuid)
	//if err != nil {
	//	fmt.Println("GetFileChunkByUUID failed:", err.Error())
	//	ctx.JSON(http.StatusInternalServerError, "GetFileChunkByUUID failed.")
	//	return
	//}

	_, err := completeMultiPartUpload(ctx, uuid, uploadID)
	if err != nil {
		fmt.Println("completeMultiPartUpload failed:", err.Error())
		ctx.JSON(http.StatusInternalServerError, err.Error())
		return
	}

	//fileChunk.IsUploaded = models.FileUploaded
	//
	//err = models.UpdateFileChunk(fileChunk)
	//if err != nil {
	//	fmt.Println("UpdateFileChunk failed:", err.Error())
	//	ctx.JSON(http.StatusInternalServerError, "UpdateFileChunk failed.")
	//	return
	//}

	ctx.JSON(http.StatusOK, gin.H{})
}

func UpdateMultipart(ctx *gin.Context) {
	uuid := ctx.PostForm("uuid")
	etag := ctx.PostForm("etag")

	fileChunk, err := cache.GetFileChunkByUUID(uuid)
	if err != nil {
		fmt.Println("GetFileChunkByUUID failed:", err.Error())
		ctx.JSON(http.StatusInternalServerError, "GetFileChunkByUUID failed.")
		return
	}

	fileChunk.CompletedParts += ctx.PostForm("chunkNumber") + "-" + strings.Replace(etag, "\"", "", -1) + ","

	err = cache.UpdateFileChunk(fileChunk)
	if err != nil {
		fmt.Println("UpdateFileChunk failed:", err.Error())
		ctx.JSON(http.StatusInternalServerError, "UpdateFileChunk failed.")
		return
	}

	ctx.JSON(http.StatusOK, gin.H{})
}

func newMultiPartUpload(ctx context.Context, objectName string) (string, error) {
	_, core, _, err := getClients()
	if err != nil {
		fmt.Println("getClients failed:", err.Error())
		return "", err
	}

	bucketName := config.MinioBucket
	// TODO: 上传文件名
	uploadId, err := core.NewMultipartUpload(ctx, bucketName, objectName, miniov7.PutObjectOptions{
		UserMetadata:       map[string]string{"File_name": "ParallelsDesktop_18_1_0_53311_2_MacPedia.dmg"},
		ContentDisposition: "attachment; filename=ParallelsDesktop_18_1_0_53311_2_MacPedia.dmg",
	})
	if err != nil {
		return "", err
	}
	return uploadId, nil
}

func genMultiPartSignedUrl(uuid string, uploadId string, partNumber int, partSize int64) (string, error) {
	//_, _, minioClient, err := getClients()
	//if err != nil {
	//	fmt.Println("getClients failed:", err.Error())
	//	return "", err
	//}

	bucketName := config.MinioBucket
	objectName := uuid

	//return minioClient.GenUploadPartSignedUrl(uploadId, bucketName, objectName, partNumber, partSize, PresignedUploadPartUrlExpireTime, config.MinioLocation)
	return multiPartSignedUrl(uploadId, bucketName, objectName, partNumber, partSize, PresignedUploadPartUrlExpireTime, config.MinioLocation)
}

func multiPartSignedUrl(uploadID string, bucketName string, objectName string, partNumber int, size int64, expires time.Duration, bucketLocation string) (string, error) {
	signedUrl := ""

	// Input validation.
	if err := s3utils.CheckValidBucketName(bucketName); err != nil {
		return signedUrl, err
	}
	if err := s3utils.CheckValidObjectName(objectName); err != nil {
		return signedUrl, err
	}
	if size > 1024*1024*1024*5 {
		return signedUrl, errors.New("size is illegal")
	}
	if size <= -1 {
		return signedUrl, errors.New("size is illegal")
	}
	if partNumber <= 0 {
		return signedUrl, errors.New("partNumber is illegal")
	}
	if uploadID == "" {
		return signedUrl, errors.New("uploadID is illegal")
	}

	// Get resources properly escaped and lined up before using them in http request.
	urlValues := make(url.Values)
	// Set part number.
	urlValues.Set("partNumber", strconv.Itoa(partNumber))
	// Set upload id.
	urlValues.Set("uploadId", uploadID)
	client, _, _, err := getClients()
	if err != nil {
		fmt.Println("getClients failed:", err.Error())
		return "", err
	}
	req, err := client.Presign("PUT", bucketName, objectName, expires, urlValues)
	if err != nil {
		log.Println("newRequest failed:", err.Error())
		return signedUrl, err
	}

	signedUrl = req.String()
	return signedUrl, nil
}

func completeMultiPartUpload(ctx context.Context, uuid string, uploadID string) (string, error) {
	_, core, client, err := getClients()
	if err != nil {
		fmt.Println("getClients failed:", err.Error())
		return "", err
	}

	bucketName := config.MinioBucket
	objectName := uuid

	partInfos, err := client.ListObjectParts(bucketName, objectName, uploadID)
	if err != nil {
		fmt.Println("ListObjectParts failed:", err.Error())
		return "", err
	}

	var complMultipartUpload completeMultipartUpload
	for _, partInfo := range partInfos {
		complMultipartUpload.Parts = append(complMultipartUpload.Parts, miniov7.CompletePart{
			PartNumber: partInfo.PartNumber,
			ETag:       partInfo.ETag,
		})
	}

	// Sort all completed parts.
	sort.Sort(completedParts(complMultipartUpload.Parts))
	return core.CompleteMultipartUpload(ctx, bucketName, objectName, uploadID, complMultipartUpload.Parts, miniov7.PutObjectOptions{})
}

func GetSuccessChunks(ctx *gin.Context) {
	var res = -1
	var uuid, uploaded, uploadID, chunks string

	fileMD5 := ctx.Query("md5")
	for {
		fileChunk, err := cache.GetFileChunkByMD5(fileMD5)
		if err != nil {
			fmt.Println("GetFileChunkByMD5 failed:", err.Error())
			break
		}

		uuid = fileChunk.UUID
		uploaded = strconv.Itoa(fileChunk.IsUploaded)
		uploadID = fileChunk.UploadID

		bucketName := config.MinioBucket
		objectName := fileMD5

		isExist, err := isObjectExist(bucketName, objectName)
		if err != nil {
			fmt.Println("isObjectExist failed:", err.Error())
			break
		}

		if isExist {
			uploaded = "1"
			if fileChunk.IsUploaded != cache.FileUploaded {
				logger.LOG.Info("the file has been uploaded but not recorded")
				fileChunk.IsUploaded = 1
				if err = cache.UpdateFileChunk(fileChunk); err != nil {
					fmt.Println("UpdateFileChunk failed:", err.Error())
				}
			}
			res = 0
			break
		} else {
			uploaded = "0"
			if fileChunk.IsUploaded == cache.FileUploaded {
				logger.LOG.Info("the file has been recorded but not uploaded")
				fileChunk.IsUploaded = 0
				if err = cache.UpdateFileChunk(fileChunk); err != nil {
					fmt.Println("UpdateFileChunk failed:", err.Error())
				}
			}
		}

		_, _, client, err := getClients()
		if err != nil {
			fmt.Println("getClients failed:", err.Error())
			break
		}

		partInfos, err := client.ListObjectParts(bucketName, objectName, uploadID)
		if err != nil {
			fmt.Println("ListObjectParts failed:", err.Error())
			break
		}

		for _, partInfo := range partInfos {
			chunks += strconv.Itoa(partInfo.PartNumber) + "-" + partInfo.ETag + ","
		}

		break
	}

	{ // 无依赖版本
		//objectName := ctx.Query("md5")
		//uploadID := ctx.Query("uploadID") // 如果没有uploadID，说明是第一次上传，需要上传所有分片
		//uuid = gouuid.NewV4().String()
		//bucketName := config.MinioBucket
		//isExist, err := isObjectExist(bucketName, objectName)
		//if err != nil {
		//	fmt.Println("isObjectExist failed:", err.Error())
		//	ctx.JSON(http.StatusInternalServerError, "isObjectExist failed")
		//}
		//if isExist {
		//	ctx.JSON(http.StatusOK, gin.H{
		//		"resultCode": "0",
		//		"uuid":       uuid,
		//		"uploaded":   "1",
		//		"uploadID":   uploadID,
		//		"chunks":     chunks,
		//	})
		//}
		//_, _, client, err := getClients()
		//if err != nil {
		//	fmt.Println("getClients failed:", err.Error())
		//	ctx.JSON(http.StatusInternalServerError, "getClients failed")
		//}
		//
		//// 列出已经上传的分片
		//partInfos, err := client.ListObjectParts(bucketName, objectName, uploadID)
		//if err != nil {
		//	fmt.Println("ListObjectParts failed:", err.Error())
		//	ctx.JSON(http.StatusInternalServerError, "listObjectParts failed")
		//}
		//
		//for _, partInfo := range partInfos {
		//	chunks += strconv.Itoa(partInfo.PartNumber) + "-" + partInfo.ETag + ","
		//}
	}

	ctx.JSON(http.StatusOK, gin.H{
		"resultCode": strconv.Itoa(res),
		"uuid":       uuid,
		"uploaded":   uploaded,
		"uploadID":   uploadID,
		"chunks":     chunks,
	})
}

func isObjectExist(bucketName string, objectName string) (bool, error) {
	isExist := false
	doneCh := make(chan struct{})
	defer close(doneCh)

	client, _, _, err := getClients()
	if err != nil {
		fmt.Println("getClients failed:", err.Error())
		return isExist, err
	}

	objectCh := client.ListObjects(bucketName, objectName, false, doneCh)
	for object := range objectCh {
		if object.Err != nil {
			fmt.Println(object.Err)
			return isExist, object.Err
		}
		isExist = true
		break
	}

	return isExist, nil
}
