package tests

import (
	"fmt"
	"github.com/sunfmin/integrationtest"
	"github.com/sunfmin/mgodb"
	"github.com/sunfmin/tenpu"
	"github.com/sunfmin/tenpu/thumbnails"
	"io"
	"io/ioutil"
	"labix.org/v2/mgo/bson"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

var thumbnailsCollectionName = "thumbnails"

type ThumbnailStorageMaker struct {
}

func (m *ThumbnailStorageMaker) Make(r *http.Request) (storage *thumbnails.Storage, err error) {
	db := mgodb.NewDatabase("localhost", "tenpu_test")
	storage = thumbnails.NewStorage(db, "thumbnails")
	return
}

func TestThumbnailLoader(t *testing.T) {

	mgodb.Setup("localhost", "tenpu_test")
	mgodb.DropCollections(collectionName, thumbnailsCollectionName)

	m := &maker{}
	_, meta, _, _ := m.MakeForUpload(nil)

	http.HandleFunc("/thumbpostupload", tenpu.MakeUploader(m))
	http.HandleFunc("/thumbload", thumbnails.MakeLoader(&thumbnails.Configuration{
		Maker: m,
		ThumbnailStorageMaker: &ThumbnailStorageMaker{},
		ThumbnailSpecs: []*thumbnails.ThumbnailSpec{
			{Name: "icon", Width: 100},
		},
	}))

	ts := httptest.NewServer(http.DefaultServeMux)
	defer ts.Close()

	var err error

	s := integrationtest.NewSession()
	res := integrationtest.Must(s.PostMultipart(ts.URL+"/thumbpostupload", func(w *multipart.Writer) {
		w.WriteField("OwnerId", "my12345")
		p1, err := w.CreateFormFile("t", "t.jpg")
		if err != nil {
			panic(err)
		}
		tf, err1 := os.Open("t.jpg")
		if err1 != nil {
			panic(err1)
		}
		defer tf.Close()

		io.Copy(p1, tf)
	}))

	b, _ := ioutil.ReadAll(res.Body)
	strb := string(b)
	if !strings.Contains(strb, "my12345") {
		t.Errorf("%+v", strb)
	}

	atts := meta.Attachments("my12345")
	if len(atts) != 1 {
		t.Errorf("%+v", atts)
	}

	res, err = http.Get(ts.URL + fmt.Sprintf("/thumbload?id=%s&thumb=icon", atts[0].Id))
	if err != nil {
		panic(err)
	}

	f, err2 := os.OpenFile("thumbGenerated.jpg", os.O_CREATE|os.O_RDWR, 0666)
	if err2 != nil {
		panic(err2)
	}

	defer f.Close()

	io.Copy(f, res.Body)

	http.Get(ts.URL + fmt.Sprintf("/thumbload?id=%s&thumb=icon", atts[0].Id))

	var thumbs []thumbnails.Thumbnail
	mgodb.FindAll(thumbnailsCollectionName, bson.M{}, &thumbs)
	if len(thumbs) != 1 {
		t.Errorf("%+v", thumbs)
	}
}
