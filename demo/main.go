package main

import (
	// "io"
	// "io/ioutil"
	"github.com/sunfmin/mgodb"
	"github.com/sunfmin/tenpu"
	"github.com/sunfmin/tenpu/gridfs"
	"log"
	"net/http"
	// "os"
)

func assetsHandler(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, r.URL.Path[len("/"):])
}

func main() {

	mgodb.Setup("localhost", "tenpu_test")

	http.HandleFunc("/upload", func(w http.ResponseWriter, r *http.Request) {
		log.Println("uploading", r.Header.Get("Content-Length"))

		mr, err := r.MultipartReader()

		if err != nil {
			panic(err)
		}

		c := tenpu.Client{
			Storage: &gridfs.Storage{},
		}
		c.UploadAttachments(mr)

	})
	http.HandleFunc("/static/", assetsHandler)

	log.Fatal(http.ListenAndServe(":5000", nil))
}
