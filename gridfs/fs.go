package gridfs

import (
	"github.com/sunfmin/mgodb"
	"github.com/sunfmin/tenpu"
	"io"
	"launchpad.net/mgo"
	"launchpad.net/mgo/bson"
)

type Storage struct {
}

func (s *Storage) Put(filename string, contentType string, body io.Reader, attachment *tenpu.Attachment) (err error) {
	var f *mgo.GridFile
	mgodb.DatabaseDo(func(db *mgo.Database) {
		f, err = db.GridFS("fs").Create(filename)
		defer f.Close()
		if err != nil {
			panic(err)
		}
		f.SetContentType(contentType)
		io.Copy(f, body)

	})

	attachment.Id = f.Id().(bson.ObjectId).Hex()
	attachment.ContentLength = f.Size()
	attachment.ContentType = f.ContentType()
	attachment.Filename = f.Name()
	attachment.MD5 = f.MD5()
	return
}

func (s *Storage) Copy(attachment *tenpu.Attachment, w io.Writer) (err error) {
	mgodb.DatabaseDo(func(db *mgo.Database) {
		f, err := db.GridFS("fs").OpenId(bson.ObjectIdHex(attachment.Id))
		if err == nil {
			defer f.Close()
			io.Copy(w, f)
		}
	})
	return
}

func NewStorage() (s *Storage) {
	s = &Storage{}
	return
}
