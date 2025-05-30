package client

import (
	"net/http"
	"net/url"

	"github.com/rs/zerolog/log"
)

type ProxyClient struct {
	*http.Client
}

func NewProxyClient() *ProxyClient {
	proxyURL, err := url.Parse("http://172.27.0.1:9000")
	if err != nil {
		log.Fatal().Err(err).Msg("解析代理URL失败")
	}
	return &ProxyClient{
		Client: &http.Client{
			Transport: &http.Transport{
				Proxy: http.ProxyURL(proxyURL),
			},
		},
	}
}
