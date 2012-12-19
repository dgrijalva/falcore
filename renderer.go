package falcore

import (
	"io"
	"io/ioutil"
	"os"
	"template"
	"bytes"
	"net/http"
	"mime"
	"strings"
	"encoding/json"
)

// Base interface for renderers
type Renderer interface {
	Render(req *Request, data interface{}) (io.ReadCloser, string, os.Error)
}

// Response generating helper
func RenderResponse(req *Request, status int, headers http.Header, renderer Renderer, data interface{})(res *http.Response, err os.Error) {
	if body, typ, error := renderer.Render(req, data); error == nil {
		res = new(http.Response)
		res.StatusCode = status
		res.ProtoMajor = 1
		res.ProtoMinor = 1
		res.Request = req.HttpRequest
		res.Header = make(http.Header)
		res.Body = body
		res.ContentLength = -1
		if headers != nil {
			res.Header = headers
		}
		res.Header.Set("Content-Type", typ)
	} else {
		err = error
	}
	return
}

// Choose a renderer based on Accept header
type FormatSelector struct {
	types map[string]Renderer
	defaultType Renderer
}
type FormatSelectError os.Error

// Supply a default formatter or nil
func NewFormatSelector(defaultType Renderer) *FormatSelector {
	return &FormatSelector{types: make(map[string]Renderer), defaultType: defaultType}
}

type acceptType struct {
	typ string
	params map[string]string
}

func (r *FormatSelector) Add(typ string, rr Renderer) {
	r.types[typ] = rr
}

func (r *FormatSelector) Render(req *Request, data interface{}) (io.ReadCloser, string, os.Error) {
	
	// Get preferred content types
	var types []acceptType
	if req.HttpRequest.Header.Get("Accept") != "" {
		typeStrings := strings.Split(req.HttpRequest.Header.Get("Accept"), ",")
		types = make([]acceptType, len(typeStrings))
		for i, t := range typeStrings  {
			typ, params := mime.ParseMediaType(t)
			types[i] = acceptType{typ, params}
		}
	}
	
	// Look for renderer that matches preferred type
	for _, typ := range types {
		if f, ok := r.types[typ.typ]; ok {
			return f.Render(req, data)
		}
	}
	
	// Use the default type
	if r.defaultType != nil {
		return r.defaultType.Render(req, data)
	}
	
	return nil, "", FormatSelectError(os.NewError("No matching response type available"))
}

// Render using a standard library template
type TemplateRenderer struct {
	Format string
	Template *template.Template
}

func NewTemplateRenderer(t string, format string)(r *TemplateRenderer, err os.Error) {
	tmpl := template.New("")
	if _, err = tmpl.Parse(t); err == nil {
		r = &TemplateRenderer{format, tmpl}
	}
	return
}

func (r *TemplateRenderer) Render(req *Request, data interface{}) (io.ReadCloser, string, os.Error) {
	body := new(bytes.Buffer)
	var err os.Error
	if err = r.Template.Execute(body, data); err == nil {
		return ioutil.NopCloser(body), r.Format, nil
	}

	// Error
	return nil, "", err
}

// Render JSON
type JSONRenderer struct {
	
}

func (r *JSONRenderer) Render(req *Request, data interface{}) (io.ReadCloser, string, os.Error) {
	body := new(bytes.Buffer)
	var err os.Error
	jsonE := json.NewEncoder(body)
	if err = jsonE.Encode(data); err == nil {
		return ioutil.NopCloser(body), "application/json", nil
	}
	
	return nil, "", err
}