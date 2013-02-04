package tenpu

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"time"
)

// the time format used for HTTP headers 
const HTTP_TIME_FORMAT = "Mon, 02 Jan 2006 15:04:05 GMT"

func formatHour(hours string) string {
	d, _ := time.ParseDuration(hours + "h")
	return time.Now().Add(d).Format(HTTP_TIME_FORMAT)
}

func formatDays(day int) string {
	return formatHour(fmt.Sprintf("%d", day*24))
}

func formatDayToSec(day int) string {
	return fmt.Sprintf("%d", day*60*60*24)
}

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

func MakeFileLoader(viewer AttachmentViewer, maker StorageMaker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		storage, meta, err := maker.Make(r)

		id, download := viewer.ViewId(r)
		if id == "" || err != nil {
			http.NotFound(w, r)
			return
		}

		att := meta.AttachmentById(id)
		if att == nil {
			http.NotFound(w, r)
			return
		}

		if download {
			w.Header().Set("Content-Type", "application/octet-stream")
		} else {
			w.Header().Set("Content-Type", att.ContentType)
		}
		w.Header().Set("Content-Length", fmt.Sprintf("%d", att.ContentLength))
		SetCacheControl(w, 30)
		err = storage.Copy(att, w)

		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		return
	}
}

func SetCacheControl(w http.ResponseWriter, days int) {
	w.Header().Set("Expires", formatDays(days))
	w.Header().Set("Cache-Control", "max-age="+formatDayToSec(days))
}

func MakeZipFileLoader(loader AttachmentsLoader, maker StorageMaker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		storage, _, err := maker.Make(r)
		var atts []*Attachment
		atts, err = loader.LoadAttachments(r)

		if atts == nil {
			http.NotFound(w, r)
			return
		}
		// w.Header().Set("Content-Type", "application/zip")
		// w.Header().Set("Content-Length", fmt.Sprintf("%d", att.ContentLength))
		// w.Header().Set("Expires", formatDays(30))
		// w.Header().Set("Cache-Control", "max-age="+formatDayToSec(30))

		err = storage.Zip(atts, w)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		return
	}
}

func MakeDeleter(deleter AttachmentDeleter, maker StorageMaker) http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {
		storage, meta, err := maker.Make(r)
		id := r.FormValue("Id")

		if id == "" {
			err = errors.New("id required.")
			writeJson(w, err.Error(), []*Attachment{})
			return
		}

		att, err := deleteAttachment(r, id, deleter, storage, meta)

		if err != nil {
			writeJson(w, err.Error(), []*Attachment{att})
			return
		}

		writeJson(w, "", []*Attachment{att})
		return
	}
}

func deleteAttachment(r *http.Request, id string, deleter AttachmentDeleter, storage BlobStorage, meta MetaStorage) (att *Attachment, err error) {
	att = meta.AttachmentById(id)

	shouldUpdate, _, err := deleter.UpdateAttrsOrDelete(att, r)

	if err != nil {
		return
	}

	if shouldUpdate {
		meta.Put(att)
		return
	}

	err = storage.Delete(att)
	if err != nil {
		return
	}
	meta.Remove(id)
	return
}

func MakeUploader(initializer AttachmentInitializer, maker StorageMaker) http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {
		storage, meta, err1 := maker.Make(r)
		if err1 != nil {
			panic(err1)
		}

		mr, err := r.MultipartReader()

		if err != nil {
			panic(err)
		}

		var part *multipart.Part
		var attachments []*Attachment

		var metaInfo = make(map[string]string)

		for {
			part, err = mr.NextPart()
			if err != nil {
				break
			}

			if part.FileName() == "" {
				metaInfo[part.FormName()] = formValue(part)
				continue
			}

			att := &Attachment{}
			err = initializer.Fill(att, metaInfo)
			if err != nil {
				writeJson(w, err.Error(), nil)
				return
			}

			att.UploadTime = time.Now()
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
				err = meta.Put(att)
			}
		}

		ats := meta.AttachmentsByOwnerIds(attachments[0].OwnerId)
		if err != nil {
			writeJson(w, err.Error(), ats)
			return
		}

		writeJson(w, "", ats)
		return
	}
}
