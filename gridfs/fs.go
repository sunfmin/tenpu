package gridfs

import (
	"archive/zip"
	// "bytes"
	"github.com/sunfmin/mgodb"
	"github.com/sunfmin/tenpu"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"labix.org/v2/mgo"
	"labix.org/v2/mgo/bson"
	"log"
)

type Storage struct {
	database *mgodb.Database
}

// func (s *Storage) Find(collectionName string, query interface{}, result interface{}) (err error) {
// 	return s.database.FindOne(collectionName, query, result)
// }

func (s *Storage) Put(filename string, contentType string, body io.Reader, attachment *tenpu.Attachment) (err error) {
	var f *mgo.GridFile

	s.database.DatabaseDo(func(db *mgo.Database) {
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
	if attachment.IsImage() {
		s.database.DatabaseDo(func(db *mgo.Database) {
			f, err := db.GridFS("fs").OpenId(bson.ObjectIdHex(attachment.Id))
			if err == nil {
				config, _, err := image.DecodeConfig(f)
				f.Close()
				if err == nil {
					attachment.Width = config.Width
					attachment.Height = config.Height
				}
			}
		})
	}
	return
}

func (s *Storage) Copy(attachment *tenpu.Attachment, w io.Writer) (err error) {
	s.database.DatabaseDo(func(db *mgo.Database) {
		f, err := db.GridFS("fs").OpenId(bson.ObjectIdHex(attachment.Id))
		if err == nil {
			defer f.Close()
			io.Copy(w, f)
		}
	})
	return
}

func (s *Storage) Zip(attachments []*tenpu.Attachment, w io.Writer) (err error) {
	// Create a buffer to write our archive to.
	// buf := new(bytes.Buffer)

	// Create a new zip archive.
	zipfile := zip.NewWriter(w)

	// Add some files to the archive.
	for _, att := range attachments {
		f, _ := zipfile.Create(att.Filename)
		if err != nil {
			log.Println(err)
			return
		}
		err = s.Copy(att, f)
		if err != nil {
			log.Println(err)
			return
		}
	}

	// Make sure to check the error on Close.
	err = zipfile.Close()
	if err != nil {
		log.Println(err)
		return
	}

	// _, err = buf.WriteTo(w)
	return
}

func NewStorage(db *mgodb.Database) (s *Storage) {
	s = &Storage{}
	if db == nil {
		db = mgodb.DefaultDatabase
	}
	s.database = db
	return
}

func (s *Storage) Delete(attachment *tenpu.Attachment) (err error) {
	s.database.DatabaseDo(func(db *mgo.Database) {
		err = db.GridFS("fs").RemoveId(bson.ObjectIdHex(attachment.Id))
	})
	return
}
