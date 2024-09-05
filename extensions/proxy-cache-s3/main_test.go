package main

import (
	"fmt"
	"github.com/donknap/proxy-cache-s3/util"
	"testing"
	"time"
)

func TestTtl(t *testing.T) {
	s := util.CalculateTTL("zzz")
	fmt.Printf("%v \n", s)
}

func TestCacheKey(t *testing.T) {
	s := util.GetCacheKey("$host127.0.0.1:10000-$path/get-$methodGET")
	fmt.Printf("%v \n", s)
}

func TestS3(t *testing.T) {
	a := "CLLXCjkzzLgNLE4PPqk1"
	b := "iPmmjjqxzIb54DAyRSBIibURuA5ionSV09NM409a"

	a = "LTAI5tEMGFYiQSk9kauLji4D"
	b = "gNfWd3EqA6fS67tOzL9wwP8jG1SbfB"

	region := "cn-beijing"
	bucket := "proxy-cache"
	expires := 24 * time.Hour

	sessionToken := ""       // 如果没有临时凭证，可以留空
	host := "s3.test.w7.com" // 例如, "example-bucket.s3.cn-beijing.amazonaws.com"
	key := "test.png"
	versionID := "" // 如果不需要版本ID，可以留空

	url, err := util.GeneratePresignedURL(a, b, sessionToken, region, host, bucket, key, expires, versionID)
	if err != nil {
		fmt.Println("Error generating presigned URL:", err)
		return
	}

	fmt.Println("Presigned URL:", url)
}
