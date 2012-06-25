package tenpu

import (
	"github.com/sunfmin/mgodb"
	"io"
	"labix.org/v2/mgo"
	"labix.org/v2/mgo/bson"
)

var CollectionName = "attachments"

type Client struct {
	Storage Storage
}

type Storage interface {
	Put(filename string, contentType string, body io.Reader, attachment *Attachment) (err error)
	Copy(attachment *Attachment, w io.Writer) (err error)
}

type Attachment struct {
	Id            string `bson:"_id"`
	OwnerId       string
	Category      string
	Filename      string
	ContentType   string
	MD5           string
	ContentLength int64
	Error         string
}

func (att *Attachment) MakeId() interface{} {
	return att.Id
}

func Attachments(ownerid string) (r []*Attachment) {
	mgodb.CollectionDo(CollectionName, func(c *mgo.Collection) {
		c.Find(bson.M{"ownerid": ownerid}).All(&r)
	})
	return
}

func AttachmentById(id string) (r *Attachment) {
	mgodb.CollectionDo(CollectionName, func(c *mgo.Collection) {
		c.Find(bson.M{"_id": id}).One(&r)
	})
	return
}
