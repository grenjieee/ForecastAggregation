package httpclient

import (
	"ForecastSync/internal/config"
	"compress/gzip"
	"io" // 新增：导入io包（ReadCloser属于io包）
	"net/http"
	"net/url"
	"time"

	"github.com/sirupsen/logrus"
)

// NewHTTPClient 通用HTTP客户端构建方法（支持代理、超时、自动解压）
func NewHTTPClient(cfg *config.PlatformConfig, logger *logrus.Logger) *http.Client {
	transport := &http.Transport{
		MaxIdleConns:        100,
		IdleConnTimeout:     30 * time.Second,
		DisableCompression:  false,
		TLSHandshakeTimeout: 10 * time.Second,
	}

	// 配置代理
	if cfg.Proxy != "" {
		proxyURL, err := url.Parse(cfg.Proxy)
		if err != nil {
			logger.WithError(err).WithField("proxy", cfg.Proxy).Warn("代理地址解析失败，将不使用代理")
		} else {
			transport.Proxy = http.ProxyURL(proxyURL)
			logger.WithField("proxy", cfg.Proxy).Info("HTTP客户端已配置代理")
		}
	}

	return &http.Client{
		Timeout:   time.Duration(cfg.Timeout) * time.Second,
		Transport: &compressedTransport{transport: transport, logger: logger},
	}
}

// ========== 修正核心：使用io.ReadCloser替代错误的http.CloseReader ==========
type compressedTransport struct {
	transport http.RoundTripper
	logger    *logrus.Logger
}

func (c *compressedTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Add("Accept-Encoding", "gzip")
	resp, err := c.transport.RoundTrip(req)
	if err != nil {
		return nil, err
	}

	// 处理gzip解压
	if resp.Header.Get("Content-Encoding") == "gzip" {
		gzReader, err := gzip.NewReader(resp.Body)
		if err != nil {
			c.logger.WithError(err).Warn("gzip解压失败，返回原始响应")
			return resp, nil
		}
		// 核心修正：resp.Body的类型是io.ReadCloser（不是http.CloseReader）
		resp.Body = &gzipReadCloser{
			Reader: gzReader,
			closer: resp.Body, // resp.Body本身就是io.ReadCloser类型
		}
		resp.Header.Del("Content-Encoding")
	}

	return resp, nil
}

// gzipReadCloser 正确包装io.ReadCloser，实现Close()方法
type gzipReadCloser struct {
	*gzip.Reader
	closer io.ReadCloser // 修正：使用io.ReadCloser
}

// Close 正确关闭所有资源（解压reader + 原始响应体）
func (g *gzipReadCloser) Close() error {
	// 先关闭gzip reader，再关闭原始响应体
	if err := g.Reader.Close(); err != nil {
		return err
	}
	return g.closer.Close()
}
