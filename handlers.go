package tenpu

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"labix.org/v2/mgo/bson"
	"mime/multipart"
	"net/http"
)

type Result struct {
	Error       string
	Attachments []*Attachment
}

func writeJson(w http.ResponseWriter, err string, attachments []*Attachment) {
	r := &Result{
		Error:       err,
		Attachments: attachments,
	}
	w.Header().Set("Content-Type", "application/json")
	b, _ := json.Marshal(r)
	w.Write(b)
}

func formValue(p *multipart.Part) string {
	var b bytes.Buffer
	io.CopyN(&b, p, int64(1<<20)) // Copy max: 1 MiB
	return b.String()
}

func MakeFileLoader(identifierName string, storage Storage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.URL.Query().Get(identifierName)
		if id == "" {
			http.NotFound(w, r)
			return
		}
		var att *Attachment
		storage.Find(CollectionName, bson.M{"_id": id}, &att)
		if att == nil {
			http.NotFound(w, r)
			return
		}

		w.Header().Set("Content-Type", att.ContentType)
		w.Header().Set("Content-Length", fmt.Sprintf("%d", att.ContentLength))

		err := storage.Copy(att, w)

		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		return
	}
}

func MakeDeleter(AttachmentIdName string, storage Storage) http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {
		id := r.FormValue("Id")
		var err error

		if id == "" {
			err = errors.New("id required.")
			writeJson(w, err.Error(), []*Attachment{})
			return
		}

		att, err := deleteAttachment(id, storage)

		if err != nil {
			writeJson(w, err.Error(), []*Attachment{att})
			return
		}

		writeJson(w, "", []*Attachment{att})
		return
	}
}

func deleteAttachment(id string, storage Storage) (att *Attachment, err error) {
	dbc := DatabaseClient{Database: storage.Database()}
	att = dbc.AttachmentById(id)
	err = storage.Delete(att)
	if err != nil {
		return
	}

	err = dbc.RemoveAttachmentById(id)
	return
}

func MakeTheUploader(ownerName string, category string, clear bool, storage Storage) http.HandlerFunc {
	return makeUploader(ownerName, category, clear, storage)
}

func MakeUploader(ownerName string, category string, storage Storage) http.HandlerFunc {
	return makeUploader(ownerName, category, false, storage)
}

func MakeClearUploader(ownerName string, category string, storage Storage) http.HandlerFunc {
	return makeUploader(ownerName, category, true, storage)
}

func makeUploader(ownerName string, category string, clear bool, storage Storage) http.HandlerFunc {
	if storage == nil {
		panic("storage must be provided.")
	}

	return func(w http.ResponseWriter, r *http.Request) {
		mr, err := r.MultipartReader()

		if err != nil {
			panic(err)
		}

		var ownerId string
		var part *multipart.Part
		var attachments []*Attachment

		for {
			part, err = mr.NextPart()
			if err != nil {
				break
			}

			if part.FileName() == "" {
				if part.FormName() == ownerName {
					ownerId = formValue(part)
				}
				continue
			}

			if ownerId == "" {
				writeJson(w, fmt.Sprintf("ownerId required, Please put a hidden field in form called `%s`", ownerName), nil)
				return
			}
			att := &Attachment{}
			att.Category = category
			att.OwnerId = ownerId
			err = storage.Put(part.FileName(), part.Header["Content-Type"][0], part, att)
			if err != nil {
				att.Error = err.Error()
			}
			attachments = append(attachments, att)
		}
		if len(attachments) == 0 {
			writeJson(w, "No attachments uploaded.", nil)
			return
		}

		for _, att := range attachments {
			if att.Error != "" {
				err = errors.New("Some attachment has error")
			} else {
				storage.Database().Save(CollectionName, att)
			}
		}

		if clear {
			dbc := DatabaseClient{Database: storage.Database()}
			ats := dbc.Attachments(ownerId)
			for i := len(ats) - 1; i >= 0; i -= 1 {
				found := false
				for _, newAt := range attachments {
					if ats[i].Id == newAt.Id {
						found = true
						break
					}
				}
				if found {
					continue
				}
				for _, newAt := range attachments {
					if newAt.OwnerId == ats[i].OwnerId {
						_, err = deleteAttachment(ats[i].Id, storage)
					}
				}
			}
		}

		dbc := DatabaseClient{Database: storage.Database()}
		ats := dbc.Attachments(ownerId)
		if err != nil {
			writeJson(w, err.Error(), ats)
			return
		}

		writeJson(w, "", ats)
		return
	}
}
