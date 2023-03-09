package main

import (
	"embed"

	"oss/config"
	_ "oss/docs"
	"oss/lib/cors"
	"oss/lib/postgres"
	models "oss/model"
	minioService "oss/service/minio"

	"github.com/gin-contrib/static"
	//"github.com/gin-contrib/static"
	"github.com/gin-gonic/gin"
)

//go:embed web/dist
var fronted embed.FS

// @title minio-breakpoint-upload API
// @version 1.0
// @description  This is a minio upload server.
// @BasePath /api/v1/
func main() {
	config.Init()
	postgres.Init()
	models.Init()
	router := gin.New()
	router.Use(cors.Cors())
	//router.GET("/swagger/*any", gs.WrapHandler(swaggerFiles.Handler))

	minio := router.Group("/minio")
	{
		minio.GET("/get_chunks", minioService.GetSuccessChunks)
		minio.GET("/new_multipart", minioService.NewMultipart)
		minio.GET("/get_multipart_url", minioService.GetMultipartUploadUrl)
		minio.POST("/complete_multipart", minioService.CompleteMultipart)
		minio.POST("/update_chunk", minioService.UpdateMultipart)
	}

	router.Use(static.Serve("/", static.LocalFile("./web/dist/", false)))

	router.Run(":" + config.PORT)
}
