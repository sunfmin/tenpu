package tests

import (
	"fmt"
	"github.com/sunfmin/integrationtest"
	"github.com/sunfmin/mgodb"
	"github.com/sunfmin/tenpu"
	"github.com/sunfmin/tenpu/gridfs"
	"github.com/sunfmin/tenpu/thumbnails"
	"io"
	"io/ioutil"
	"launchpad.net/mgo"
	// "log"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestThumbnailLoader(t *testing.T) {

	mgodb.Setup("localhost", "tenpu_test")
	mgodb.CollectionDo(tenpu.CollectionName, func(c *mgo.Collection) {
		c.DropCollection()
	})

	st := gridfs.NewStorage()

	http.HandleFunc("/thumbpostupload", tenpu.MakeUploader("OwnerId", "posts", st))
	http.HandleFunc("/thumbload", thumbnails.MakeLoader(&thumbnails.Configuration{
		IdentifierName:     "id",
		ThumbnailParamName: "thumb",
		Storage:            gridfs.NewStorage(),
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

	atts := tenpu.Attachments("my12345")
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
}
