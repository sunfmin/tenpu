package mgometa

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
		collectionName = "attachments"
	}

	s.database = db
	s.collectionName = collectionName
	return
}

func (s *Storage) Put(att *tenpu.Attachment) (err error) {
	err = s.database.Save(s.collectionName, att)
	return
}

func (s *Storage) Attachments(ownerid string) (r []*tenpu.Attachment) {
	s.database.CollectionDo(s.collectionName, func(c *mgo.Collection) {
		c.Find(bson.M{"ownerid": ownerid}).All(&r)
	})
	return
}

func (s *Storage) AttachmentsByOwnerIds(ownerids []string) (r []*tenpu.Attachment) {
	s.database.CollectionDo(s.collectionName, func(c *mgo.Collection) {
		c.Find(bson.M{"ownerid": bson.M{"$in": ownerids}}).All(&r)
	})
	return
}

func (s *Storage) AttachmentsCountByOwnerIds(ownerids []string) (r int) {
	s.database.CollectionDo(s.collectionName, func(c *mgo.Collection) {
		r, _ = c.Find(bson.M{"ownerid": bson.M{"$in": ownerids}}).Count()
	})
	return
}

func (s *Storage) AttachmentById(id string) (r *tenpu.Attachment) {
	s.database.CollectionDo(s.collectionName, func(c *mgo.Collection) {
		c.Find(bson.M{"_id": id}).One(&r)
	})
	return
}

func (s *Storage) AttachmentByIds(ids []string) (r []*tenpu.Attachment) {
	s.database.CollectionDo(s.collectionName, func(c *mgo.Collection) {
		c.Find(bson.M{"_id": bson.M{"$in": ids}}).All(&r)
	})
	return
}

func (s *Storage) AttachmentsByGroupId(groupId string) (r *tenpu.Attachment) {
	s.database.CollectionDo(s.collectionName, func(c *mgo.Collection) {
		c.Find(bson.M{"groupid": groupId}).All(&r)
	})
	return
}

func (s *Storage) Remove(id string) (err error) {
	s.database.CollectionDo(s.collectionName, func(c *mgo.Collection) {
		c.Remove(bson.M{"_id": id})
	})
	return
}
