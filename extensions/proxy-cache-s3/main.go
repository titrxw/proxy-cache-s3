package main

import (
	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
	"github.com/donknap/proxy-cache-s3/util"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
	"github.com/tidwall/gjson"
	"net/http"
	"strings"
)

const (
	ConfigMethodPURGE = "PURGE" // 主动清理缓存

	CacheKeyHost   = "$host"
	CacheKeyPath   = "$path"
	CacheKeyMethod = "$method"
	CacheKeyCookie = "$cookie"

	CacheHttpStatusCodeOk = 200

	DefaultCacheTTL = "300s"
)

func main() {
	wrapper.SetCtx(
		"w7-proxy-cache",
		wrapper.ParseConfigBy(parseConfig),
		wrapper.ProcessRequestHeadersBy(onHttpRequestHeaders),
	)
}

type W7ProxyCache struct {
	client   wrapper.HttpClient
	cacheKey []string
	setting  struct {
		cacheTTL    string
		cacheHeader bool
		s3SecretId  string
		s3SecretKey string
		s3Region    string
		s3Bucket    string
		s3Endpoint  string
	}
}

func parseConfig(json gjson.Result, config *W7ProxyCache, log wrapper.Log) error {
	// create default config
	config.cacheKey = []string{CacheKeyHost, CacheKeyPath, CacheKeyMethod}

	config.client = wrapper.NewClusterClient(wrapper.FQDNCluster{
		FQDN: "proxy-cache-s3-httpbin-1",
		Port: 80,
	})

	// get cache ttl
	if json.Get("cache_ttl").Exists() {
		cacheTTL := json.Get("cache_ttl").String()
		cacheTTL = strings.Replace(cacheTTL, " ", "", -1)
		config.setting.cacheTTL = cacheTTL
	}

	if json.Get("cache_header").Exists() {
		value := json.Get("cache_header").Bool()
		config.setting.cacheHeader = value
		if config.setting.cacheHeader {
			config.cacheKey = append(config.cacheKey, CacheKeyCookie)
		}
	}

	if json.Get("s3_secret_id").Exists() {
		value := json.Get("s3_secret_id").String()
		config.setting.s3SecretId = strings.Replace(value, " ", "", -1)
	}

	if json.Get("s3_secret_key").Exists() {
		value := json.Get("s3_secret_key").String()
		config.setting.s3SecretKey = strings.Replace(value, " ", "", -1)
	}

	if json.Get("s3_region").Exists() {
		value := json.Get("s3_region").String()
		config.setting.s3Region = strings.Replace(value, " ", "", -1)
	}

	if json.Get("s3_bucket").Exists() {
		value := json.Get("s3_bucket").String()
		config.setting.s3Bucket = strings.Replace(value, " ", "", -1)
	}

	if json.Get("s3_endpoint").Exists() {
		value := json.Get("s3_endpoint").String()
		config.setting.s3Endpoint = strings.Replace(value, " ", "", -1)
	}

	if config.setting.cacheTTL == "" {
		log.Error("cache ttl is empty")
		return types.ErrorStatusBadArgument
	}

	if config.setting.s3SecretId == "" ||
		config.setting.s3SecretKey == "" ||
		config.setting.s3Region == "" ||
		config.setting.s3Bucket == "" ||
		config.setting.s3Endpoint == "" {
		log.Error("s3 setting is empty")
		return types.ErrorStatusBadArgument
	}
	log.Info("cachekey: " + strings.Join(config.cacheKey, "==="))

	return nil
}

func onHttpRequestHeaders(ctx wrapper.HttpContext, config W7ProxyCache, log wrapper.Log) types.Action {
	cacheKeyList := make([]string, 0)
	for _, cacheKey := range config.cacheKey {
		switch cacheKey {
		case CacheKeyHost:
			host := ctx.Host()
			cacheKey = CacheKeyHost + host
		case CacheKeyPath:
			path := ctx.Path()
			cacheKey = CacheKeyPath + path
		case CacheKeyMethod:
			method := ctx.Method()
			cacheKey = CacheKeyMethod + method
		case CacheKeyCookie:
			cookie, err := proxywasm.GetHttpRequestHeader("cookie")
			if err != nil {
				log.Error("parse request cookie failed")
				return types.ActionContinue
			}
			cacheKey = CacheKeyCookie + cookie
		default:
			log.Errorf("invalid cache key: %s", cacheKey)
		}
		cacheKeyList = append(cacheKeyList, cacheKey)
	}
	cacheKey := util.GetCacheKey(strings.Join(cacheKeyList, "-"))

	// 使用client的Get方法发起HTTP Get调用，此处省略了timeout参数，默认超时时间500毫秒
	config.client.Get("/", nil,
		// 回调函数，将在响应异步返回时被执行
		func(statusCode int, responseHeaders http.Header, responseBody []byte) {
			// 请求没有返回200状态码，进行处理
			if statusCode != http.StatusOK {
				log.Errorf("http call failed, status: %d", statusCode)
				proxywasm.SendHttpResponse(http.StatusInternalServerError, nil,
					[]byte("http call failed"), -1)
				return
			}
			// 打印响应的HTTP状态码和应答body
			log.Infof("get status: %d, response body: %s", statusCode, responseBody)
			// 从应答头中解析token字段设置到原始请求头中
			// 恢复原始请求流程，继续往下处理，才能正常转发给后端服务
			proxywasm.ResumeHttpRequest()
		})

	proxywasm.SendHttpResponseWithDetail(http.StatusOK, "hello-world", [][2]string{
		{
			"Hello", "World",
		},
	}, []byte("hello world 324234234234 \n"+cacheKey), -1)
	return types.ActionPause
}
