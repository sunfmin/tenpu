package tenpu

import (
	"fmt"
	"github.com/sunfmin/mgodb"
	"io"
	"labix.org/v2/mgo"
	"labix.org/v2/mgo/bson"
	"path"
	"time"
)

var CollectionName = "attachments"

// the time format used for HTTP headers 
const HTTP_TIME_FORMAT = "Mon, 02 Jan 2006 15:04:05 GMT"

func FormatHour(hours string) string {
	d, _ := time.ParseDuration(hours + "h")
	return time.Now().Add(d).Format(HTTP_TIME_FORMAT)
}

func FormatDays(day int) string {
	return FormatHour(fmt.Sprintf("%d", day*24))
}

func FormatDayToSec(day int) string {
	return fmt.Sprintf("%d", day*60*60*24)
}

type Storage interface {
	Put(filename string, contentType string, body io.Reader, attachment *Attachment) (err error)
	Delete(attachment *Attachment) (err error)
	Copy(attachment *Attachment, w io.Writer) (err error)
	Find(collectionName string, query interface{}, result interface{}) (err error)
	Zip(entryId string, attachments []*Attachment, w io.Writer) (err error)
	Database() *mgodb.Database
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

type AttachmentForMigration struct {
	Id            string `bson:"_id"`
	OwnerId       string
	Category      string
	Filename      string
	ContentType   string
	MD5           string
	ContentLength int64
	Error         string
	GroupId       string
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

func (att *Attachment) AddOwnerId(ownerid string, db *mgodb.Database) (err error) {
	for _, id := range att.OwnerId {
		if id == ownerid {
			return
		}
	}
	att.OwnerId = append(att.OwnerId, ownerid)
	err = db.Save(CollectionName, att)
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

func AttachmentById2(id string, db *mgodb.Database) (r *Attachment) {
	db.CollectionDo(CollectionName, func(c *mgo.Collection) {
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

func (dbc *DatabaseClient) AttachmentsCountByOwnerIds(ownerids []string) (r int) {
	dbc.Database.CollectionDo(CollectionName, func(c *mgo.Collection) {
		r, _ = c.Find(bson.M{"ownerid": bson.M{"$in": ownerids}}).Count()
	})
	return
}

func (dbc *DatabaseClient) AttachmentById(id string) (r *Attachment) {
	dbc.Database.CollectionDo(CollectionName, func(c *mgo.Collection) {
		c.Find(bson.M{"_id": id}).One(&r)
	})
	return
}

func (dbc *DatabaseClient) AttachmentByIds(ids []string) (r []*Attachment) {
	dbc.Database.CollectionDo(CollectionName, func(c *mgo.Collection) {
		c.Find(bson.M{"_id": bson.M{"$in": ids}}).All(&r)
	})
	return
}

func (dbc *DatabaseClient) AttachmentsByGroupId(groupId string) (r *Attachment) {
	dbc.Database.CollectionDo(CollectionName, func(c *mgo.Collection) {
		c.Find(bson.M{"groupid": groupId}).All(&r)
	})
	return
}

func (dbc *DatabaseClient) RemoveAttachmentById(id string) (err error) {
	dbc.Database.CollectionDo(CollectionName, func(c *mgo.Collection) {
		c.Remove(bson.M{"_id": id})
	})
	return
}
