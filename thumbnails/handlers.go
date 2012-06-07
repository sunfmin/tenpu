package thumbnails

import (
	"bytes"
	"code.google.com/p/graphics-go/graphics"
	"fmt"
	"github.com/sunfmin/mgodb"
	"github.com/sunfmin/tenpu"
	"image"
	_ "image/gif"
	"image/jpeg"
	_ "image/png"
	"io"
	"launchpad.net/mgo/bson"
	"log"
	"net/http"
)

var CollectionName = "thumbnails"

type ThumbnailSpec struct {
	Name   string
	Width  int
	Height int
}

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

type Thumbnail struct {
	Id       bson.ObjectId `bson:"_id"`
	ParentId string
	BodyId   string
	Name     string
	Width    int64
	Height   int64
}

func (tb *Thumbnail) MakeId() interface{} {
	if tb.Id == "" {
		tb.Id = bson.NewObjectId()
	}
	return tb.Id
}

type Configuration struct {
	IdentifierName     string
	ThumbnailParamName string
	Storage            tenpu.Storage
	ThumbnailSpecs     []*ThumbnailSpec
}

func MakeLoader(config *Configuration) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.URL.Query().Get(config.IdentifierName)
		if id == "" {
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

		var att *tenpu.Attachment
		mgodb.FindOne(tenpu.CollectionName, bson.M{"_id": id}, &att)
		if att == nil {
			http.NotFound(w, r)
			return
		}

		var thumb *Thumbnail
		mgodb.FindOne(CollectionName, bson.M{"attachmentid": id, "name": thumbName}, &thumb)

		if thumb == nil {
			var buf bytes.Buffer
			config.Storage.Copy(att, &buf)
			thumbAtt := &tenpu.Attachment{}

			body, width, height, err := resize(&buf, spec)

			if err != nil {
				log.Printf("tenpu/thumbnails: %+v", err)
				http.NotFound(w, r)
				return
			}

			config.Storage.Put(att.Filename, att.ContentType, body, thumbAtt)

			mgodb.Save(tenpu.CollectionName, thumbAtt)

			thumb = &Thumbnail{
				Name:     thumbName,
				ParentId: id,
				BodyId:   thumbAtt.Id,
				Width:    int64(width),
				Height:   int64(height),
			}
			mgodb.Save(CollectionName, thumb)
		}

		thumbAttachment := tenpu.AttachmentById(thumb.BodyId)
		if thumbAttachment == nil {
			log.Printf("tenpu/thumbnails: Can't find body attachment by %+v", thumb)
			http.NotFound(w, r)
			return
		}

		w.Header().Set("Content-Type", thumbAttachment.ContentType)
		w.Header().Set("Content-Length", fmt.Sprintf("%d", thumbAttachment.ContentLength))

		err := config.Storage.Copy(thumbAttachment, w)

		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		return
	}
}

func resize(from *bytes.Buffer, spec *ThumbnailSpec) (to io.Reader, w int, h int, err error) {

	src, name, err := image.Decode(from)
	if err != nil {
		return
	}

	w, h = spec.CalculateRect(src.Bounds())
	dst := image.NewRGBA(image.Rect(0, 0, w, h))

	var buf bytes.Buffer
	if err = graphics.Thumbnail(dst, src); err != nil {
		log.Println(err)
	}
	if name == "jpeg" {
		jpeg.Encode(&buf, dst, &jpeg.Options{95})
	}
	to = &buf
	return
}
