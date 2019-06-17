package s3proxy

import (
	"net/http"
	"net/http/httputil"
	"net/url"
	"path"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

var (
	Client          *s3.S3
	once            sync.Once
	ExpiresInterval time.Duration = 1 * time.Minute
	UserAgent       string        = "go-s3proxy/1.0"
)

func mustNewS3() *s3.S3 {
	sess := session.Must(session.NewSession(&aws.Config{
		Region: aws.String("us-east-1"),
	}))
	return s3.New(sess)
}

func Proxy(bucketName string) http.Handler {
	once.Do(func() {
		if Client == nil {
			Client = mustNewS3()
		}
	})

	proxy := httputil.ReverseProxy{Director: func(r *http.Request) {
		r.Header = http.Header{} // Don't send client's request headers to Amazon.
		r.Header.Set("User-Agent", UserAgent)

		base := path.Base(r.URL.Path)
		// ref. https://docs.aws.amazon.com/ja_jp/sdk-for-go/v1/developer-guide/s3-example-presigned-urls.html
		req, _ := Client.GetObjectRequest(&s3.GetObjectInput{
			Bucket: aws.String(bucketName),
			Key:    aws.String(base),
		})
		signedURL, err := req.Presign(10 * time.Second)
		if err != nil {
			r.URL = nil
			r.Host = ""
			return
		}

		parsed, err := url.Parse(signedURL)
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
