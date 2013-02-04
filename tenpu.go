package tenpu

import (
	"io"
	"path"
	"time"
)

type BlobStorage interface {
	Put(filename string, contentType string, body io.Reader, attachment *Attachment) (err error)
	Delete(attachment *Attachment) (err error)
	Copy(attachment *Attachment, w io.Writer) (err error)
	// Find(collectionName string, query interface{}, result interface{}) (err error)
	Zip(attachments []*Attachment, w io.Writer) (err error)
}

type MetaStorage interface {
	Put(input *Attachment) (err error)
	Remove(id string) (err error)
	Attachments(ownerid string) (r []*Attachment)
	AttachmentsByOwnerIds(ownerids []string) (r []*Attachment)
	AttachmentsCountByOwnerIds(ownerids []string) (r int)
	AttachmentById(id string) (r *Attachment)
	AttachmentByIds(ids []string) (r []*Attachment)
	AttachmentsByGroupId(groupId string) (r *Attachment)
}

type RequestValue interface {
	FormValue(key string) string
}

type StorageMaker interface {
	Make(r RequestValue) (blob BlobStorage, meta MetaStorage, err error)
}

type AttachmentsLoader interface {
	LoadAttachments(r RequestValue) (atts []*Attachment, err error)
}

type AttachmentViewer interface {
	ViewId(r RequestValue) (id string, download bool)
}

type AttachmentDeleter interface {
	UpdateAttrsOrDelete(att *Attachment, r RequestValue) (shouldUpdate bool, shouldDelete bool, err error)
}

type AttachmentInitializer interface {
	Fill(att *Attachment, metaInfo map[string]string) (err error)
}

type Attachment struct {
	Id            string `bson:"_id"`
	OwnerId       []string
	Category      string
	Filename      string
	ContentType   string
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
	case "image/png", "image/jpeg", "image/gif":
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
