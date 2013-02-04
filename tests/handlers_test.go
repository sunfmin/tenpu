package tests

import (
	"github.com/sunfmin/mgodb"
	"github.com/sunfmin/tenpu"
	"github.com/sunfmin/tenpu/gridfs"
	"github.com/sunfmin/tenpu/mgometa"
	"io/ioutil"
	"labix.org/v2/mgo"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

var collectionName = "attachments"

type maker struct {
}

func (m *maker) Make(r *http.Request) (storage tenpu.BlobStorage, meta tenpu.MetaStorage, err error) {
	db := mgodb.NewDatabase("localhost", "tenpu_test")
	storage = gridfs.NewStorage(db)
	meta = mgometa.NewStorage(db, collectionName)
	return
}

func TestUploader(t *testing.T) {
	mgodb.Setup("localhost", "tenpu_test")

	mgodb.CollectionDo(collectionName, func(c *mgo.Collection) {
		c.DropCollection()
	})
	m := &maker{}
	_, meta, _ := m.Make(nil)

	http.HandleFunc("/postupload", tenpu.MakeUploader("OwnerId", "posts", m))
	http.HandleFunc("/load", tenpu.MakeFileLoader("id", m, true))
	ts := httptest.NewServer(http.DefaultServeMux)
	defer ts.Close()

	req, _ := http.NewRequest("POST", ts.URL+"/postupload", strings.NewReader(multipartContent))
	req.Header.Set("Content-Type", "multipart/form-data; boundary=----WebKitFormBoundarySHaDkk90eMKgsVUj")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		panic(err)
	}
	b, _ := ioutil.ReadAll(res.Body)
	strb := string(b)
	if !strings.Contains(strb, "4facead362911fa23c000001") {
		t.Errorf("%+v", strb)
	}

	atts := meta.Attachments("4facead362911fa23c000001")
	if len(atts) != 2 {
		t.Errorf("%+v", atts[0])
	}

	res, err = http.Get(ts.URL + "/load?id=" + atts[0].Id)
	if err != nil {
		panic(err)
	}

	b, _ = ioutil.ReadAll(res.Body)
	strb = string(b)
	if strb != "the file content a\n" {
		t.Errorf("%+v", strb)
	}
}

func TestUploadWithoutOwnerId(t *testing.T) {
	mgodb.Setup("localhost", "tenpu_test")

	m := &maker{}

	http.HandleFunc("/errorpostupload", tenpu.MakeUploader("OwnerId", "posts", m))
	ts := httptest.NewServer(http.DefaultServeMux)
	defer ts.Close()

	req, _ := http.NewRequest("POST", ts.URL+"/errorpostupload", strings.NewReader(noOwnerIdPostContent))
	req.Header.Set("Content-Type", "multipart/form-data; boundary=----WebKitFormBoundarySHaDkk90eMKgsVUj")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		panic(err)
	}
	b, _ := ioutil.ReadAll(res.Body)
	strb := string(b)
	if !strings.Contains(strb, "ownerId required") {
		t.Errorf("%+v", strb)
	}

}

const singlePartContent = `

------WebKitFormBoundarySHaDkk90eMKgsVUj
Content-Disposition: form-data; name="OwnerId"

4facead362911fa23c000002
------WebKitFormBoundarySHaDkk90eMKgsVUj
Content-Disposition: form-data; name="Files[]"; filename="filec.txt"
Content-Type: text/plain

the file content c

------WebKitFormBoundarySHaDkk90eMKgsVUj--
`

const multipartContent = `

------WebKitFormBoundarySHaDkk90eMKgsVUj
Content-Disposition: form-data; name="OwnerId"

4facead362911fa23c000001
------WebKitFormBoundarySHaDkk90eMKgsVUj
Content-Disposition: form-data; name="Files[]"; filename="filea.txt"
Content-Type: text/plain

the file content a

------WebKitFormBoundarySHaDkk90eMKgsVUj
Content-Disposition: form-data; name="Files[]"; filename="fileb.txt"
Content-Type: text/plain

the file content b

------WebKitFormBoundarySHaDkk90eMKgsVUj--
`

const noOwnerIdPostContent = `

------WebKitFormBoundarySHaDkk90eMKgsVUj
Content-Disposition: form-data; name="Files[]"; filename="filea.txt"
Content-Type: text/plain

the file content a

------WebKitFormBoundarySHaDkk90eMKgsVUj
Content-Disposition: form-data; name="Files[]"; filename="fileb.txt"
Content-Type: text/plain

the file content b

------WebKitFormBoundarySHaDkk90eMKgsVUj--
`
