package tenpu

import (
	"errors"
	"io"
	"labix.org/v2/mgo"
	"log"
	"mime/multipart"
	"net/http"
	"path"
	"time"
)

type BlobStorage interface {
	// Get(attachment *Attachment) (r io.Reader, err error)
	Put(filename string, contentType string, body io.Reader, attachment *Attachment) (err error)
	Delete(attachmentId string) (err error)
	Copy(attachment *Attachment, w io.Writer) (err error)
	CopyToStorage(attachment *Attachment, toBlob BlobStorage) (err error)
	// Find(collectionName string, query interface{}, result interface{}) (err error)
	Zip(attachments []*Attachment, w io.Writer) (err error)
}

type MetaStorage interface {
	Put(att *Attachment) (err error)
	Remove(id string) (err error)
	Attachments(ownerid string) (r []*Attachment)
	AttachmentsByOwnerIds(ownerids []string) (r []*Attachment)
	AttachmentsCountByOwnerIds(ownerids []string) (r int)
	AttachmentById(id string) (r *Attachment)
	AttachmentByIds(ids []string) (r []*Attachment)
	AttachmentsByGroupId(groupId string) (r *Attachment)
}

type Input interface {
	GetFileMeta() (filename string, contentType string, contentId string)
	GetViewMeta() (id string, thumb string, download bool)
	SetAttrsForDelete(att *Attachment) (shouldUpdate bool, shouldDelete bool, err error)
	LoadAttachments() (r []*Attachment, err error)
}

type UploadInput interface {
	GetFileMeta() (filename string, contentType string, contentId string)
	SetMultipart(part *multipart.Part) (isFile bool)
	SetAttrsForCreate(att *Attachment) (err error)
}

type StorageMaker interface {
	MakeForRead(r *http.Request) (blob BlobStorage, meta MetaStorage, input Input, err error)
	MakeForUpload(r *http.Request) (blob BlobStorage, meta MetaStorage, input UploadInput, err error)
}

type Attachment struct {
	Id            string `bson:"_id"`
	OwnerId       []string
	Category      string
	Filename      string
	ContentType   string
	ContentId     string
	MD5           string
	ContentLength int64
	Error         string
	GroupId       []string
	UploadTime    time.Time
	Width         int
	Height        int
}

func (att *Attachment) MakeId() interface{} {
	return att.Id
}

func (att *Attachment) IsImage() (r bool) {
	switch att.ContentType {
	default:
		r = false
	case "image/png", "image/jpeg", "image/jpg", "image/gif", "image/x-png", "image/pjpeg", "image/bmp":
		r = true
	}
	return

}

func (att *Attachment) Extname() (r string) {
	r = path.Ext(att.Filename)
	if len(r) > 0 {
		r = r[1:]
	}
	return
}

func DeleteAttachment(input Input, blob BlobStorage, meta MetaStorage) (att *Attachment, deleted bool, err error) {

	id, _, _ := input.GetViewMeta()

	if id == "" {
		err = errors.New("id required.")
		return
	}

	att = meta.AttachmentById(id)

	shouldUpdate, _, err := input.SetAttrsForDelete(att)

	if err != nil {
		return
	}

	if shouldUpdate {
		err = meta.Put(att)
		return
	}

	err = blob.Delete(id)
	if err != nil && err != mgo.ErrNotFound {
		return
	}

	err = meta.Remove(id)
	if err != nil {
		return
	}
	log.Printf("Delete file id:%s, name:%s, size:%.2f M", att.Id, att.Filename, float32(att.ContentLength)/1024/1024)

	deleted = true

	return

}

func CreateAttachment(input UploadInput, blob BlobStorage, meta MetaStorage, body io.Reader) (att *Attachment, err error) {
	att = &Attachment{}
	err = input.SetAttrsForCreate(att)

	if err != nil {
		return
	}

	filename, contentType, contentId := input.GetFileMeta()

	att.UploadTime = time.Now()
	att.ContentId = contentId

	err = blob.Put(filename, contentType, body, att)
	if err != nil {
		return
	}

	err = meta.Put(att)
	if err != nil {
		return
	}

	return
}

func CopyAttachment(fromBlob BlobStorage, toBlob BlobStorage, toMeta MetaStorage, att *Attachment) (err error) {

	if err = fromBlob.CopyToStorage(att, toBlob); err != nil && err != mgo.ErrNotFound {
		return
	}

	if err = toMeta.Put(att); err != nil {
		return
	}
	return
}
