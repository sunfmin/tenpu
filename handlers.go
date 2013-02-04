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

func MakeFileLoader(identifierName string, maker StorageMaker, download bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		storage, meta, err := maker.Make(r)
		id := r.URL.Query().Get(identifierName)
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

func MakeDeleter(groupId string, maker StorageMaker) http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {
		storage, meta, err := maker.Make(r)

		id := r.FormValue("Id")
		ownerId := r.FormValue("OwnerId")

		if id == "" {
			err = errors.New("id required.")
			writeJson(w, err.Error(), []*Attachment{})
			return
		}

		att, err := deleteAttachment(id, ownerId, groupId, storage, meta)

		if err != nil {
			writeJson(w, err.Error(), []*Attachment{att})
			return
		}

		writeJson(w, "", []*Attachment{att})
		return
	}
}

func deleteAttachment(id string, ownerId string, groupId string, storage BlobStorage, meta MetaStorage) (att *Attachment, err error) {
	att = meta.AttachmentById(id)
	if len(att.OwnerId) > 1 {
		groupids := []string{}
		ownids := []string{}
		for _, oid := range att.OwnerId {
			if oid == ownerId {
				continue
			}
			ownids = append(ownids, oid)
		}
		att.OwnerId = ownids

		if groupId == "" {
			groupids = att.GroupId
		} else {
			for _, gid := range att.GroupId {
				if gid == groupId {
					continue
				}
				groupids = append(groupids, gid)
			}
		}
		att.GroupId = groupids
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

func MakeTheUploader(ownerName string, category string, clear bool, maker StorageMaker, groupName string) http.HandlerFunc {
	return makeUploader(ownerName, category, clear, maker, groupName)
}

func MakeUploader(ownerName string, category string, maker StorageMaker) http.HandlerFunc {
	return makeUploader(ownerName, category, false, maker, "")
}

func MakeClearUploader(ownerName string, category string, maker StorageMaker) http.HandlerFunc {
	return makeUploader(ownerName, category, true, maker, "")
}

func makeUploader(ownerName string, category string, clear bool, maker StorageMaker, groupName string) http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {
		storage, meta, err1 := maker.Make(r)
		if err1 != nil {
			panic(err1)
		}

		mr, err := r.MultipartReader()

		if err != nil {
			panic(err)
		}

		var ownerId string
		var groupId string
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
				if groupName != "" && part.FormName() == groupName {
					groupId = formValue(part)
				}
				continue
			}

			if ownerId == "" {
				writeJson(w, fmt.Sprintf("ownerId required, Please put a hidden field in form called `%s`", ownerName), nil)
				return
			}
			att := &Attachment{}
			att.Category = category
			att.OwnerId = []string{ownerId}
			att.GroupId = []string{groupId}
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

		ats := meta.AttachmentsByOwnerIds([]string{ownerId})
		if err != nil {
			writeJson(w, err.Error(), ats)
			return
		}

		writeJson(w, "", ats)
		return
	}
}
