package thumbnails

import (
	"github.com/sunfmin/mgodb"
	"github.com/sunfmin/tenpu"
	"labix.org/v2/mgo"
	"labix.org/v2/mgo/bson"
	// "log"
	"net/http"
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
	Make(r *http.Request) (storage *Storage, err error)
}

type Thumbnail struct {
	Id bson.ObjectId `bson:"_id"`
	// ParentId : original file's attachment id
	ParentId string
	// BodyId : thumbnail file's attachment id
	BodyId string
	Name   string
	Width  int64
	Height int64
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

func (s *Storage) ThumbnailByParentId(parentId string) (r []*Thumbnail) {
	s.database.CollectionDo(s.collectionName, func(c *mgo.Collection) {
		c.Find(bson.M{"parentid": parentId}).All(&r)
	})
	return
}

func (s *Storage) Put(att *Thumbnail) (err error) {
	err = s.database.Save(s.collectionName, att)
	return
}

func (s *Storage) RemoveAll(parentId string) (err error) {
	s.database.CollectionDo(s.collectionName, func(c *mgo.Collection) {
		_, err = c.RemoveAll(bson.M{"parentid": parentId})
	})
	return
}

func (s *Storage) DeleteThumbnails(parentAttId string, blob tenpu.BlobStorage, meta tenpu.MetaStorage) (err error) {
	thumbs := s.ThumbnailByParentId(parentAttId)
	// log.Println("Delete thumbnail num:", len(thumbs))
	var thumbAttIds []string
	for _, thumb := range thumbs {
		thumbAttIds = append(thumbAttIds, thumb.BodyId)
	}

	for _, thumbAttId := range thumbAttIds {

		err = blob.Delete(thumbAttId)
		if err != nil && err != mgo.ErrNotFound {
			return
		}

		err = meta.Remove(thumbAttId)
		if err != nil {
			return
		}
	}
	err = s.RemoveAll(parentAttId)
	return
}
