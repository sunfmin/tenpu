package tenpu

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/sunfmin/mgodb"
	"io"
	"launchpad.net/mgo/bson"
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
		mgodb.FindOne(CollectionName, bson.M{"_id": id}, &att)
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

func MakeUploader(ownerName string, category string, storage Storage) http.HandlerFunc {
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

		hasError := ""
		for _, att := range attachments {
			if att.Error != "" {
				hasError = "Some attachment has error"
			} else {
				mgodb.Save(CollectionName, att)
			}
		}
		writeJson(w, hasError, attachments)
		return
	}
}
