package util

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/url"
	"time"
)

// GeneratePresignedURL 生成一个 AWS S3 预签名 URL
func GeneratePresignedURL(accessKey, secretKey, sessionToken, region, host, bucket, key string, expires time.Duration, versionID string) (string, error) {
	method := "GET"
	service := "s3"
	endpoint := fmt.Sprintf("https://%s/%s%s", host, bucket, key)
	parsedURL, err := url.Parse(endpoint)
	if err != nil {
		return "", err
	}

	// 生成当前时间
	t := time.Now().UTC()
	amzDate := t.Format("20060102T150405Z")
	credentialDate := t.Format("20060102")
	expiry := t.Add(expires).UTC()
	iso8601Expiry := expiry.Format("20060102T150405Z")

	// 构建 Canonical Request
	canonicalURI := fmt.Sprintf("/%s/%s", bucket, key)
	canonicalQueryString := parsedURL.Query().Encode()
	canonicalHeaders := fmt.Sprintf("host:%s\n", host)
	signedHeaders := "host"
	payloadHash := "UNSIGNED-PAYLOAD"

	canonicalRequest := fmt.Sprintf("%s\n%s\n%s\n%s\n%s\n%s",
		method,
		canonicalURI,
		canonicalQueryString,
		canonicalHeaders,
		signedHeaders,
		payloadHash)

	canonicalRequestHash := hashSHA256(canonicalRequest)

	// 构建 String to Sign
	stringToSign := fmt.Sprintf("AWS4-HMAC-SHA256\n%s\n%s/%s/%s/aws4_request\n%s",
		amzDate,
		credentialDate,
		region,
		service,
		canonicalRequestHash)

	signingKey := getSignatureKey(secretKey, credentialDate, region, service)
	signature := hmacSHA256(signingKey, stringToSign)
	signatureHex := hex.EncodeToString(signature)

	// 构建预签名 URL
	presignedURL := fmt.Sprintf("%s?X-Amz-Algorithm=AWS4-HMAC-SHA256&X-Amz-Credential=%s%%2F%s%%2F%s%%2F%s%%2Faws4_request&X-Amz-Date=%s&X-Amz-Expires=%d&X-Amz-SignedHeaders=%s&X-Amz-Signature=%s",
		endpoint,
		url.QueryEscape(accessKey),
		credentialDate,
		region,
		service,
		iso8601Expiry,
		int(expires.Seconds()),
		signedHeaders,
		signatureHex)

	// 如果有 sessionToken，添加到 URL
	if sessionToken != "" {
		presignedURL += fmt.Sprintf("&X-Amz-Security-Token=%s", url.QueryEscape(sessionToken))
	}

	// 如果有 versionID，添加到 URL
	if versionID != "" {
		presignedURL += fmt.Sprintf("&versionId=%s", url.QueryEscape(versionID))
	}

	return presignedURL, nil
}

// hashSHA256 对输入字符串进行 SHA256 哈希
func hashSHA256(data string) string {
	hash := sha256.New()
	hash.Write([]byte(data))
	return hex.EncodeToString(hash.Sum(nil))
}

// hmacSHA256 对输入字符串进行 HMAC-SHA256 加密
func hmacSHA256(key []byte, message string) []byte {
	h := hmac.New(sha256.New, key)
	h.Write([]byte(message))
	return h.Sum(nil)
}

// getSignatureKey 生成签名密钥
func getSignatureKey(secretKey, date, region, service string) []byte {
	kDate := hmacSHA256([]byte("AWS4"+secretKey), date)
	kRegion := hmacSHA256(kDate, region)
	kService := hmacSHA256(kRegion, service)
	kSigning := hmacSHA256(kService, "aws4_request")
	return kSigning
}
