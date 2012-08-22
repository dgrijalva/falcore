package filter

import (
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// A falcore RequestFilter for serving static files
// from the filesystem.
type FileFilter struct {
	// File system base path for serving files
	BasePath string
	// Prefix in URL path
	PathPrefix string
}

func (f *FileFilter) FilterRequest(req *Request) (res *http.Response) {
	// Clean asset path
	asset_path := filepath.Clean(filepath.FromSlash(req.HttpRequest.URL.Path))

	// Resolve PathPrefix
	if strings.HasPrefix(asset_path, f.PathPrefix) {
		asset_path = asset_path[len(f.PathPrefix):]
	} else {
		Debug("%v doesn't match prefix %v", asset_path, f.PathPrefix)
		res = SimpleResponse(req.HttpRequest, 404, nil, "Not found.")
		return
	}

	// Resolve FSBase
	if f.BasePath != "" {
		asset_path = filepath.Join(f.BasePath, asset_path)
	} else {
		Error("file_filter requires a BasePath")
		return SimpleResponse(req.HttpRequest, 500, nil, "Server Error\n")
	}
	// Open File
	if file, err := os.Open(asset_path); err == nil {
		// Make sure it's an actual file
		if stat, err := file.Stat(); err == nil && stat.Mode()&os.ModeType == 0 {
			res = &http.Response{
				Request:       req.HttpRequest,
				StatusCode:    200,
				Proto:         "HTTP/1.1",
				ProtoMajor:    1,
				ProtoMinor:    1,
				Body:          file,
				Header:        make(http.Header),
				ContentLength: stat.Size(),
			}
			if ct := mime.TypeByExtension(filepath.Ext(asset_path)); ct != "" {
				res.Header.Set("Content-Type", ct)
			}
		} else {
			file.Close()
		}
	} else {
		Finest("Can't open %v: %v", asset_path, err)
	}
	return
}