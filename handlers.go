package tenpu

import (
	"encoding/json"
	"fmt"
	"log"
	"mime/multipart"
	"net/http"
	"strings"
	"time"
)

type Result struct {
	Error       string
	Attachments []*Attachment
}

func MakeFileLoader(maker StorageMaker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		storage, meta, input, err := maker.MakeForRead(r)

		id, _, download := input.GetViewMeta()
		if id == "" || err != nil {
			log.Printf("tenpu: attachment id is blank [%s] or err: %v\n", id, err)
			http.NotFound(w, r)
			return
		}

		att := meta.AttachmentById(id)
		if att == nil {
			log.Printf("tenpu: attachment can not been fould by id: [%s]\n", id)
			http.NotFound(w, r)
			return
		}

		log.Printf("Load file id:%s, name:%s, size:%.2f M", id, att.Filename, float32(att.ContentLength)/1024/1024)
		if download {
			filename, _, _ := input.GetFileMeta()
			w.Header().Set("Content-Disposition", "attachment; filename="+filename)
		}

		w.Header().Set("Content-Type", att.ContentType)
		// fix pdf Content-Type
		if strings.ToLower(att.Extname()) == "pdf" {
			w.Header().Set("Content-Type", "application/pdf")
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

func MakeZipFileLoader(maker StorageMaker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		storage, _, input, err := maker.MakeForRead(r)
		var atts []*Attachment
		atts, err = input.LoadAttachments()

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

func MakeDeleter(maker StorageMaker) http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {
		blob, meta, input, err := maker.MakeForRead(r)

		att, deleted, err := DeleteAttachment(input, blob, meta)

		if err != nil {
			writeJson(w, err.Error(), []*Attachment{att})
			return
		}

		if deleted {

		}
		writeJson(w, "", []*Attachment{att})
		return
	}
}

func MakeUploader(maker StorageMaker) http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {

		blob, meta, input, err1 := maker.MakeForUpload(r)
		if err1 != nil {
			writeJson(w, err1.Error(), nil)
			return
		}

		mr, err := r.MultipartReader()

		if err != nil {
			writeJson(w, err.Error(), nil)
			return
		}

		var part *multipart.Part
		var attachments []*Attachment

		for {
			part, err = mr.NextPart()
			if err != nil {
				break
			}

			isFile := input.SetMultipart(part)
			if !isFile {
				continue
			}

			var att *Attachment
			att, err = CreateAttachment(input, blob, meta, part)
			if err != nil {
				att.Error = err.Error()
			}
			log.Printf("Upload file id:%s, name:%s, size:%.2f M", att.Id, att.Filename, float32(att.ContentLength)/1024/1024)
			attachments = append(attachments, att)
		}

		if len(attachments) == 0 {
			writeJson(w, "No attachments uploaded.", nil)
			return
		}

		writeJson(w, "", attachments)
		return
	}
}

func SetCacheControl(w http.ResponseWriter, days int) {
	w.Header().Set("Expires", formatDays(days))
	w.Header().Set("Cache-Control", "max-age="+formatDayToSec(days))
}

func writeJson(w http.ResponseWriter, err string, attachments []*Attachment) {
	r := &Result{
		Error:       err,
		Attachments: attachments,
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	// w.Header().Set("Content-Type", "application/json")
	b, _ := json.Marshal(r)
	w.Write(b)
}

// the time format used for HTTP headers
const httpTimeFormat = "Mon, 02 Jan 2006 15:04:05 GMT"

func formatHour(hours string) string {
	d, _ := time.ParseDuration(hours + "h")
	return time.Now().Add(d).Format(httpTimeFormat)
}

func formatDays(day int) string {
	return formatHour(fmt.Sprintf("%d", day*24))
}

func formatDayToSec(day int) string {
	return fmt.Sprintf("%d", day*60*60*24)
}
