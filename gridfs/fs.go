package gridfs

import (
	"archive/zip"
	// "bytes"
	_ "code.google.com/p/go.image/bmp"
	"github.com/sunfmin/mgodb"
	"github.com/sunfmin/tenpu"
	"image"
	_ "image/jpeg"
	_ "image/png"
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
		if attachment.Id != "" {
			f.SetId(bson.ObjectIdHex(attachment.Id))
		}
		f.SetContentType(contentType)
		_, err = io.Copy(f, body)
	})

	if err == io.ErrUnexpectedEOF {
		attId := f.Id().(bson.ObjectId)
		s.Delete(attId.Hex())
		return
	}

	if attachment.Id == "" {
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
	}

	return
}

func (s *Storage) CopyToStorage(attachment *tenpu.Attachment, toBlob tenpu.BlobStorage) (err error) {
	session := s.database.GetOrDialSession().Copy()
	defer session.Close()
	db := session.DB(s.database.DatabaseName)
	reader, err := db.GridFS("fs").OpenId(bson.ObjectIdHex(attachment.Id))
	if err == nil {
		defer reader.Close()
	} else {
		// log.Println("open file error: ", attachment.Id, attachment.Filename)
		return
	}
	err = toBlob.Put(attachment.Filename, attachment.ContentType, reader, attachment)

	return
}

// Have Session problem
// func (s *Storage) Get(attachment *tenpu.Attachment) (r io.Reader, err error) {

// 	s.database.DatabaseDo(func(db *mgo.Database) {
// 		r, err = db.GridFS("fs").OpenId(bson.ObjectIdHex(attachment.Id))
// 	})
// 	return
// }

func (s *Storage) Copy(attachment *tenpu.Attachment, w io.Writer) (err error) {
	s.database.DatabaseDo(func(db *mgo.Database) {
		f, err := db.GridFS("fs").OpenId(bson.ObjectIdHex(attachment.Id))
		if err == nil {
			defer f.Close()
			io.Copy(w, f)
		} else {
			// log.Println("open file error: ", attachment.Id, attachment.Filename)
		}
	})
	return
}

func (s *Storage) Zip(attachments []*tenpu.Attachment, w io.Writer) (err error) {
	// Create a buffer to write our archive to.
	// buf := new(bytes.Buffer)

	// Create a new zip archive.
	zipfile := zip.NewWriter(w)

	attNameMap := make(map[string]bool)
	attMD5Map := make(map[string]bool)

	// Add some files to the archive.
	for _, att := range attachments {

		if _, ok := attNameMap[att.Filename]; ok {
			if _, ok := attMD5Map[att.MD5+att.Filename]; ok {
				continue
			} else {
				att.Filename = att.MD5 + "_" + att.Filename
				attMD5Map[att.MD5+att.Filename] = true
			}
		} else {
			attNameMap[att.Filename] = true
			attMD5Map[att.MD5+att.Filename] = true
		}
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

func (s *Storage) Delete(attachmentId string) (err error) {
	s.database.DatabaseDo(func(db *mgo.Database) {
		err = db.GridFS("fs").RemoveId(bson.ObjectIdHex(attachmentId))
	})
	return
}
