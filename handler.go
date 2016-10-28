package s3proxy

import (
	"net/http"
	"net/http/httputil"
	"net/url"
	"path"
	"sync"
	"time"

	"github.com/mitchellh/goamz/aws"
	"github.com/mitchellh/goamz/s3"
)

var (
	Client          *s3.S3
	once            sync.Once
	ExpiresInterval time.Duration = 1 * time.Minute
	UserAgent       string        = "go-s3proxy/1.0"
)

func mustNewS3() *s3.S3 {
	auth, err := aws.EnvAuth()
	if err != nil {
		panic(err)
	}
	return s3.New(auth, aws.USEast)
}

func Proxy(bucketName string) http.Handler {
	once.Do(func() {
		if Client == nil {
			Client = mustNewS3()
		}
	})

	bucket := Client.Bucket(bucketName)

	proxy := httputil.ReverseProxy{Director: func(r *http.Request) {
		r.Header = http.Header{} // Don't send client's request headers to Amazon.
		r.Header.Set("User-Agent", UserAgent)

		base := path.Base(r.URL.Path)
		signed := bucket.SignedURL(base, time.Now().Add(ExpiresInterval))
		parsed, err := url.Parse(signed)
		if err != nil {
			r.URL = nil
			r.Host = ""
			return
		}
		r.URL = parsed
		r.Host = parsed.Host
	}}
	return &proxy
}
