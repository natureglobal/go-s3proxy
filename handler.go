package s3file

import (
	"github.com/mitchellh/goamz/aws"
	"github.com/mitchellh/goamz/s3"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"time"
)

var (
	Bucket *s3.Bucket
)

const (
	ExpiresInterval time.Duration = 1 * time.Minute
	UserAgent       string        = "s3file/1.0"
)

func mustNewS3Bucket() *s3.Bucket {
	auth, err := aws.EnvAuth()
	if err != nil {
		panic(err)
	}
	bucket := os.Getenv("AWS_S3FILE_BUCKET")
	if bucket == "" {
		panic("AWS_S3FILE_BUCKET required")
	}
	client := s3.New(auth, aws.USEast)
	return client.Bucket(bucket)
}

func Handler(w http.ResponseWriter, req *http.Request) {
	if Bucket == nil {
		Bucket = mustNewS3Bucket()
	}
	last := path.Base(req.URL.Path)
	signed := Bucket.SignedURL(last, time.Now().Add(ExpiresInterval))
	req, err := http.NewRequest("GET", signed, nil)
	if err != nil {
		log.Printf("http.NewRequest error: %s", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte{})
		return
	}
	req.Header.Set("User-Agent", UserAgent)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Printf("http.DefaultClient.Do error: %s", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte{})
		return
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		w.WriteHeader(res.StatusCode)
		w.Write([]byte{})
		return
	}
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Printf("ioutil.ReadAll error: %s", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte{})
		return
	}

	// log.Printf("headers: %#v", res.Header)
	w.Header().Set("Content-Length", res.Header.Get("Content-Length"))
	w.Header().Set("Content-Type", res.Header.Get("Content-Type"))
	w.WriteHeader(res.StatusCode)
	w.Write(body)
}
