package minio

import (
	"net"
	"net/http"
	"sync"
	"time"

	"oss/config"
	"oss/lib/minio_ext"

	miniov7 "github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

var minioClient *miniov7.Client = nil

var coreClient *miniov7.Core = nil

var minioClientExt *minio_ext.Client = nil

var mutex *sync.Mutex

func init() {
	mutex = new(sync.Mutex)
}

func getClients() (*miniov7.Client, *miniov7.Core, *minio_ext.Client, error) {
	var client1 *miniov7.Client
	var client2 *miniov7.Core
	var client3 *minio_ext.Client
	mutex.Lock()

	if nil != minioClient && nil != coreClient && nil != minioClientExt {
		client1 = minioClient
		client2 = coreClient
		client3 = minioClientExt
		mutex.Unlock()
		return client1, client2, client3, nil
	}

	aliasedURL := config.MinioAddress
	accessKeyID := config.MinioAccessKeyId
	secretAccessKey := config.MinioSecretAccessKey
	secure := config.MinioSecure == "true"

	var err error
	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		MaxIdleConns:          10,
		IdleConnTimeout:       30 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 10 * time.Second,
		DisableCompression:    true,
	}

	opts := &miniov7.Options{
		Creds:     credentials.NewStaticV4(config.MinioAccessKeyId, config.MinioSecretAccessKey, ""),
		Transport: transport,
	}
	if nil == minioClient {
		minioClient, err = miniov7.New(aliasedURL, opts)
	}

	if nil != err {
		mutex.Unlock()
		return nil, nil, nil, err
	}

	client1 = minioClient

	if nil == coreClient {
		coreClient, err = miniov7.NewCore(aliasedURL, opts)
	}

	client2 = coreClient

	if nil == minioClientExt {
		minioClientExt, err = minio_ext.New(aliasedURL, accessKeyID, secretAccessKey, secure)
	}

	if nil != err {
		mutex.Unlock()
		return nil, nil, nil, err
	}

	client3 = minioClientExt

	mutex.Unlock()

	return client1, client2, client3, nil
}
