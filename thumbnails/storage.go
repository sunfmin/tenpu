package thumbnails

import (
	"github.com/sunfmin/mgodb"
	"github.com/sunfmin/tenpu"
	"labix.org/v2/mgo"
	"labix.org/v2/mgo/bson"
)

type Storage struct {
	database       *mgodb.Database
	collectionName string
}

func NewStorage(db *mgodb.Database, collectionName string) (s *Storage) {
	s = &Storage{}
	if db == nil {
		db = mgodb.DefaultDatabase
	}

	if collectionName == "" {
		collectionName = "thumbnails"
	}

	s.database = db
	s.collectionName = collectionName
	return
}

type ThumbnailStorageMaker interface {
	Make(r tenpu.RequestValue) (storage *Storage, err error)
}

type Thumbnail struct {
	Id       bson.ObjectId `bson:"_id"`
	ParentId string
	BodyId   string
	Name     string
	Width    int64
	Height   int64
}

func (tb *Thumbnail) MakeId() interface{} {
	if tb.Id == "" {
		tb.Id = bson.NewObjectId()
	}
	return tb.Id
}

func (s *Storage) ThumbnailByName(parentId string, name string) (r *Thumbnail) {
	s.database.CollectionDo(s.collectionName, func(c *mgo.Collection) {
		c.Find(bson.M{"parentid": parentId, "name": name}).One(&r)
	})
	return
}

func (s *Storage) Put(att *Thumbnail) (err error) {
	err = s.database.Save(s.collectionName, att)
	return
}
