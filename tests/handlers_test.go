package tests

import (
	"github.com/sunfmin/mgodb"
	"github.com/sunfmin/tenpu"
	"github.com/sunfmin/tenpu/gridfs"
	"io/ioutil"
	"labix.org/v2/mgo"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestUploader(t *testing.T) {

	mgodb.Setup("localhost", "tenpu_test")
	mgodb.CollectionDo(tenpu.CollectionName, func(c *mgo.Collection) {
		c.DropCollection()
	})

	st := gridfs.NewStorage()

	http.HandleFunc("/postupload", tenpu.MakeUploader("OwnerId", "posts", st))
	http.HandleFunc("/load", tenpu.MakeFileLoader("id", st))
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

	atts := tenpu.Attachments("4facead362911fa23c000001")
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

	http.HandleFunc("/errorpostupload", tenpu.MakeUploader("OwnerId", "posts", gridfs.NewStorage()))
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
