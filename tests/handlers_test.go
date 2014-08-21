package tests

import (
	"bytes"
	"errors"
	"github.com/sunfmin/mgodb"
	"github.com/theplant/tenpu"
	"github.com/theplant/tenpu/gridfs"
	"github.com/theplant/tenpu/mgometa"
	mgo "gopkg.in/mgo.v2"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

var collectionName = "attachments"

type tenpuInput struct {
	Id          string
	FileName    string
	ContentType string
	Thumb       string
	Download    bool
	OwnerId     string
}

func (d *tenpuInput) GetFileMeta() (filename string, contentType string, contentId string) {
	filename = d.FileName
	contentType = d.ContentType
	return
}

func (d *tenpuInput) GetViewMeta() (id string, thumb string, download bool) {
	id = d.Id
	thumb = d.Thumb
	download = d.Download
	return
}

func (al *tenpuInput) LoadAttachments() (atts []*tenpu.Attachment, err error) {
	return
}

func formValue(p *multipart.Part) string {
	var b bytes.Buffer
	io.CopyN(&b, p, int64(1<<20)) // Copy max: 1 MiB
	return b.String()
}

func (d *tenpuInput) SetMultipart(part *multipart.Part) (isFile bool) {
	if part.FileName() != "" {
		d.FileName = part.FileName()
		d.ContentType = part.Header["Content-Type"][0]
		isFile = true
		return
	}

	switch part.FormName() {
	case "OwnerId":
		d.OwnerId = formValue(part)
	}
	return
}

func (d *tenpuInput) SetAttrsForCreate(att *tenpu.Attachment) (err error) {
	if d.OwnerId == "" {
		err = errors.New("ownerId required")
		return
	}
	att.OwnerId = []string{d.OwnerId}
	return
}

func (d *tenpuInput) SetAttrsForDelete(att *tenpu.Attachment) (shouldUpdate bool, shouldDelete bool, err error) {
	shouldDelete = true
	return
}

type maker struct {
}

func (m *maker) MakeForRead(r *http.Request) (storage tenpu.BlobStorage, meta tenpu.MetaStorage, input tenpu.Input, err error) {
	var i *tenpuInput
	storage, meta, i, err = m.make(r)
	i.Id = r.FormValue("id")
	i.OwnerId = r.FormValue("OwnerId")
	i.Thumb = r.FormValue("thumb")
	input = i
	return
}

func (m *maker) MakeForUpload(r *http.Request) (storage tenpu.BlobStorage, meta tenpu.MetaStorage, input tenpu.UploadInput, err error) {
	storage, meta, input, err = m.make(r)
	return
}

func (m *maker) make(r *http.Request) (storage tenpu.BlobStorage, meta tenpu.MetaStorage, input *tenpuInput, err error) {
	db := mgodb.NewDatabase("localhost", "tenpu_test")
	storage = gridfs.NewStorage(db)
	meta = mgometa.NewStorage(db, collectionName)
	input = &tenpuInput{}
	return
}

var m = &maker{}

func TestUploader(t *testing.T) {
	mgodb.Setup("localhost", "tenpu_test")

	mgodb.CollectionDo(collectionName, func(c *mgo.Collection) {
		c.DropCollection()
	})

	_, meta, _, _ := m.MakeForUpload(nil)

	http.HandleFunc("/postupload", tenpu.MakeUploader(m))
	http.HandleFunc("/load", tenpu.MakeFileLoader(m))
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
		return
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

	http.HandleFunc("/errorpostupload", tenpu.MakeUploader(m))
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
