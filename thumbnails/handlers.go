package thumbnails

import (
	"bytes"
	"fmt"
	"github.com/sunfmin/resize"
	"github.com/sunfmin/tenpu"
	"image"
	_ "image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
)

type ThumbnailSpec struct {
	Name   string
	Width  int
	Height int
}

var DefaultThumbnailBuf []byte

func (ts *ThumbnailSpec) CalculateRect(rect image.Rectangle) (w int, h int) {
	if ts.Width == 0 && ts.Height == 0 {
		panic("tenpu/thumbnails: must provide width, or height for thumbnails.")
	}

	if ts.Height == 0 {
		w = ts.Width
		h = int((float64(ts.Width) / float64(rect.Dx())) * float64(rect.Dy()))
		return
	}

	if ts.Width == 0 {
		h = ts.Height
		w = int((float64(ts.Height) / float64(rect.Dy())) * float64(rect.Dx()))
		return
	}

	if (float64(ts.Width)/float64(rect.Dx()))*float64(rect.Dy()) > float64(ts.Height) {
		h = ts.Height
		w = int((float64(ts.Height) / float64(rect.Dy())) * float64(rect.Dx()))
		return
	}

	w = ts.Width
	h = int((float64(ts.Width) / float64(rect.Dx())) * float64(rect.Dy()))
	return
}

type Configuration struct {
	IdentifierName     string
	ThumbnailParamName string
	Maker              tenpu.StorageMaker
	ThumbnailStorage   *Storage
	ThumbnailSpecs     []*ThumbnailSpec
	DefaultThumbnail   string
}

func MakeLoader(config *Configuration) http.HandlerFunc {
	if DefaultThumbnailBuf == nil && len(config.DefaultThumbnail) > 0 {
		fileHandler, err := os.Open(config.DefaultThumbnail)
		if err != nil {
			panic(err)
		}

		DefaultThumbnailBuf, err = ioutil.ReadAll(fileHandler)
		if err != nil {
			panic(err)
		}
		fileHandler.Close()
	}

	return func(w http.ResponseWriter, r *http.Request) {
		storage, meta, err1 := config.Maker.Make(r)

		id := r.URL.Query().Get(config.IdentifierName)
		if id == "" || err1 != nil {
			http.NotFound(w, r)
			return
		}
		thumbName := r.URL.Query().Get(config.ThumbnailParamName)
		if thumbName == "" {
			http.NotFound(w, r)
			return
		}

		var spec *ThumbnailSpec
		for _, ts := range config.ThumbnailSpecs {
			if ts.Name == thumbName {
				spec = ts
				break
			}
		}

		if spec == nil {
			log.Println("tenpu/thumbnails: Can't find thumbnail spec %+v in %+v", thumbName, config.ThumbnailSpecs)
			http.NotFound(w, r)
			return
		}

		var att = meta.AttachmentById(id)
		if att == nil {
			http.NotFound(w, r)
			return
		}

		thumb := config.ThumbnailStorage.ThumbnailByName(id, thumbName)

		if thumb == nil {
			var err error
			thumb, err = resizeAndStore(storage, meta, config.ThumbnailStorage, att, spec, thumbName, id)
			if err != nil {
				log.Printf("tenpu/thumbnails: %+v", err)
			}

			if thumb == nil {
				w.Header().Set("Content-Type", "image/png")
				io.Copy(w, bytes.NewBuffer(DefaultThumbnailBuf))
				return
			}
		}

		thumbAttachment := meta.AttachmentById(thumb.BodyId)
		if thumbAttachment == nil {
			log.Printf("tenpu/thumbnails: Can't find body attachment by %+v", thumb)
			http.NotFound(w, r)
			return
		}

		w.Header().Set("Content-Type", thumbAttachment.ContentType)
		w.Header().Set("Content-Length", fmt.Sprintf("%d", thumbAttachment.ContentLength))
		tenpu.SetCacheControl(w, 30)

		err := storage.Copy(thumbAttachment, w)

		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		return
	}
}

func resizeAndStore(storage tenpu.BlobStorage, meta tenpu.MetaStorage, thumbnailStorage *Storage, att *tenpu.Attachment, spec *ThumbnailSpec, thumbName string, id string) (thumb *Thumbnail, err error) {

	var buf bytes.Buffer
	storage.Copy(att, &buf)

	if buf.Len() == 0 {
		return
	}
	thumbAtt := &tenpu.Attachment{}

	body, width, height, err := resizeThumbnail(&buf, spec)

	if err != nil {
		return
	}

	err = storage.Put(att.Filename, att.ContentType, body, thumbAtt)
	if err != nil {
		return
	}

	err = meta.Put(thumbAtt)
	if err != nil {
		return
	}

	thumb = &Thumbnail{
		Name:     thumbName,
		ParentId: id,
		BodyId:   thumbAtt.Id,
		Width:    int64(width),
		Height:   int64(height),
	}
	err = thumbnailStorage.Put(thumb)
	return
}

func resizeThumbnail(from *bytes.Buffer, spec *ThumbnailSpec) (to io.Reader, w int, h int, err error) {

	src, name, err := image.Decode(from)
	if err != nil {
		return
	}
	srcB := src.Bounds()

	w, h = spec.CalculateRect(srcB)

	if w >= srcB.Dx() || h >= srcB.Dy() {
		w, h = srcB.Dx(), srcB.Dy()
	}

	rect := image.Rect(0, 0, srcB.Dx(), srcB.Dy())
	dst := resize.Resize(src, rect, w, h)

	var buf bytes.Buffer
	switch name {
	case "jpeg":
		jpeg.Encode(&buf, dst, &jpeg.Options{95})
	case "png":
		png.Encode(&buf, dst)
	case "gif":
		jpeg.Encode(&buf, dst, &jpeg.Options{95})
	}

	to = &buf
	return
}
