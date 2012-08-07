package tenpu

import (
	"github.com/sunfmin/mgodb"
	"io"
	"labix.org/v2/mgo"
	"labix.org/v2/mgo/bson"
	"path"
)

var CollectionName = "attachments"

type Storage interface {
	Put(filename string, contentType string, body io.Reader, attachment *Attachment) (err error)
	Delete(attachment *Attachment) (err error)
	Copy(attachment *Attachment, w io.Writer) (err error)
	Find(collectionName string, query interface{}, result interface{}) (err error)
	Database() *mgodb.Database
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

func Attachments(ownerid string) (r []*Attachment) {
	mgodb.CollectionDo(CollectionName, func(c *mgo.Collection) {
		c.Find(bson.M{"ownerid": ownerid}).All(&r)
	})
	return
}

func AttachmentsByOwnerIds(ownerids []string) (r []*Attachment) {
	mgodb.CollectionDo(CollectionName, func(c *mgo.Collection) {
		c.Find(bson.M{"ownerid": bson.M{"$in": ownerids}}).All(&r)
	})
	return
}

func AttachmentById(id string) (r *Attachment) {
	mgodb.CollectionDo(CollectionName, func(c *mgo.Collection) {
		c.Find(bson.M{"_id": id}).One(&r)
	})
	return
}

func RemoveAttachmentById(id string) (err error) {
	mgodb.CollectionDo(CollectionName, func(c *mgo.Collection) {
		c.Remove(bson.M{"_id": id})
	})
	return
}

type DatabaseClient struct {
	Database *mgodb.Database
}

func (dbc *DatabaseClient) Attachments(ownerid string) (r []*Attachment) {
	dbc.Database.CollectionDo(CollectionName, func(c *mgo.Collection) {
		c.Find(bson.M{"ownerid": ownerid}).All(&r)
	})
	return
}

func (dbc *DatabaseClient) AttachmentsByOwnerIds(ownerids []string) (r []*Attachment) {
	dbc.Database.CollectionDo(CollectionName, func(c *mgo.Collection) {
		c.Find(bson.M{"ownerid": bson.M{"$in": ownerids}}).All(&r)
	})
	return
}

func (dbc *DatabaseClient) AttachmentById(id string) (r *Attachment) {
	dbc.Database.CollectionDo(CollectionName, func(c *mgo.Collection) {
		c.Find(bson.M{"_id": id}).One(&r)
	})
	return
}

func (dbc *DatabaseClient) RemoveAttachmentById(id string) (err error) {
	dbc.Database.CollectionDo(CollectionName, func(c *mgo.Collection) {
		c.Remove(bson.M{"_id": id})
	})
	return
}
