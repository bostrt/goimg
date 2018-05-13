package main

import (
	"context"
	"encoding/binary"
	"fmt"
	"html/template"
	"log"
	"net/http"

	"github.com/unrolled/logger"

	"github.com/GeertJohan/go.rice"
	"github.com/julienschmidt/httprouter"

	"github.com/teris-io/shortid"
)

var (
	maxUploadSize int64  = 10 * 1024 * 1024 // 2 mb
	AppCookie     string = "goimg"
)

type Page struct {
	UUID   string
	Title  string
	Images []string
	Image  *Image
	Owned  bool
}

// Server ...
type Server struct {
	config    Config
	templates *Templates
	router    *httprouter.Router

	imageDao *ImageDao
	fs       *FS

	// Logger
	logger *logger.Logger

	// Stats/Metrics
	// counters *Counters
	// stats    *stats.Stats
}

// NewServer ...
func NewServer(imageDao *ImageDao, fs *FS, config Config) *Server {
	server := &Server{
		config:    config,
		router:    httprouter.New(),
		templates: NewTemplates("base"),
		imageDao:  imageDao,
		fs:        fs,

		// Logger
		logger: logger.New(logger.Options{
			Prefix:               "goimg",
			RemoteAddressHeaders: []string{"X-Forwarded-For"},
			OutputFlags:          log.LstdFlags,
		}),

		// Stats/Metrics
		// counters: NewCounters(),
		// stats:    stats.New(),
	}

	// Templates
	box := rice.MustFindBox("templates")

	viewTemplate := template.New("view")
	template.Must(viewTemplate.Parse(box.MustString("view.html")))
	template.Must(viewTemplate.Parse(box.MustString("base.html")))

	uploadTemplate := template.New("upload")
	template.Must(uploadTemplate.Parse(box.MustString("upload.html")))
	template.Must(uploadTemplate.Parse(box.MustString("base.html")))

	aboutTemplate := template.New("about")
	template.Must(aboutTemplate.Parse(box.MustString("about.html")))
	template.Must(aboutTemplate.Parse(box.MustString("base.html")))

	notFoundTemplate := template.New("notfound")
	template.Must(notFoundTemplate.Parse(box.MustString("404.html")))
	template.Must(notFoundTemplate.Parse(box.MustString("base.html")))

	recentTemplate := template.New("recent")
	template.Must(recentTemplate.Parse(box.MustString("recent.html")))
	template.Must(recentTemplate.Parse(box.MustString("base.html")))

	server.templates.Add("view", viewTemplate)
	server.templates.Add("upload", uploadTemplate)
	server.templates.Add("about", aboutTemplate)
	server.templates.Add("notfound", notFoundTemplate)
	server.templates.Add("recent", recentTemplate)

	server.initRoutes()

	return server
}

func (s *Server) Index(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	s.render("upload", w, nil)
}

func (s *Server) ViewRecent(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	data := &Page{
		Images: s.imageDao.ListRecent(),
	}
	s.render("recent", w, data)
}

func (s *Server) About(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	s.render("about", w, nil)
}

func (s *Server) NotFound(w http.ResponseWriter, _ *http.Request, _ httprouter.Params) {
	w.WriteHeader(http.StatusNotFound)
	s.render("notfound", w, nil)
}

func (s *Server) Upload(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	id, _ := shortid.Generate()
	deleteKey, _ := shortid.Generate()
	cookie := r.Context().Value(AppCookie).(string)

	// validate file size
	r.Body = http.MaxBytesReader(w, r.Body, maxUploadSize)
	if err := r.ParseMultipartForm(maxUploadSize); err != nil {
		fmt.Println(err)
		return
	}

	// parse and validate file and post parameters
	file, _, err := r.FormFile("file")
	if err != nil {
		fmt.Println(err)
		return
	}
	defer file.Close()

	// Save to disk
	newPath, thumbPath := s.fs.Save(file, id)

	image := NewImage(r.FormValue("owner"), id, newPath, thumbPath, r.FormValue("private") != "", r.FormValue("expire"), deleteKey, cookie)

	s.imageDao.Save(image)

	if err != nil {
		return
	}

	// done
	http.Redirect(w, r, fmt.Sprintf("/view/%s", image.UUID), http.StatusFound)
}

func (s *Server) ViewImage(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	UUID := params.ByName("UUID")
	cookie := r.Context().Value(AppCookie).(string)
	image, err := s.imageDao.Load(UUID)
	if err != nil || image == nil {
		s.NotFound(w, nil, nil)
		return
	}
	data := &Page{
		Title: "View",
		UUID:  UUID,
		Image: image,
		Owned: cookie == image.cookie,
	}
	s.render("view", w, data)
}

// GetImage -- Retrieve an image given its UUID
func (s *Server) GetImage(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	UUID := params.ByName("UUID")
	thumbnail := r.URL.Query().Get("thumbnail")
	image, err := s.imageDao.Load(UUID)
	if err != nil || image == nil {
		s.NotFound(w, nil, nil)
		return
	}

	// Check file is present before trying to serve. If it does not, ServeFile will
	// do some redirects trying to help.
	orig, thumb := s.fs.Ensure(image)

	if thumbnail != "" && thumb {
		http.ServeFile(w, r, image.thumbPath)
	} else if orig {
		http.ServeFile(w, r, image.path)
	} else {
		// Something is wrong here.
		s.NotFound(w, nil, nil)
	}
}

// DeleteImage - Delete an image given its UUID and valid delete key.
func (s *Server) DeleteImage(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	UUID := params.ByName("UUID")
	deleteKey := params.ByName("key")
	image, err := s.imageDao.Load(UUID)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	if image.Delete != deleteKey {
		w.WriteHeader(http.StatusForbidden)
		return
	}
	err = s.imageDao.Delete(image)
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(http.StatusNotFound)
		return
	}
	err = s.fs.Delete(image)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

}

func (s *Server) render(name string, w http.ResponseWriter, ctx interface{}) {
	buf, err := s.templates.Exec(name, ctx)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	_, err = buf.WriteTo(w)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *Server) initRoutes() {
	s.router.ServeFiles(
		"/css/*filepath",
		rice.MustFindBox("static/css").HTTPBox(),
	)

	s.router.ServeFiles(
		"/js/*filepath",
		rice.MustFindBox("static/js").HTTPBox(),
	)

	s.router.GET("/", s.Index)
	// UI
	s.router.POST("/upload", s.Upload)
	s.router.GET("/recent", s.ViewRecent)
	s.router.GET("/about", s.About)
	s.router.GET("/404", s.NotFound)
	s.router.GET("/view/:UUID", s.ViewImage)
	// API
	s.router.GET("/i/:UUID", s.GetImage)
	s.router.GET("/d/:UUID/:key", s.DeleteImage)
}

// ListenAndServe ...
func (s *Server) ListenAndServe() {
	log.Fatal(
		http.ListenAndServe(
			cfg.bind,
			s.logger.Handler(
				cookies(s.router),
			),
		),
	)
}

func cookies(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var cookie *http.Cookie
		cookie, err := r.Cookie(AppCookie)
		if err != nil {
			// Set cookie
			val, _ := shortid.Generate()
			cookie = &http.Cookie{Name: AppCookie, Value: val}
			http.SetCookie(w, cookie)
		}
		// Store value in requst context for later
		ctx := context.WithValue(r.Context(), cookie.Name, cookie.Value)
		h.ServeHTTP(w, r.WithContext(ctx))
	})
}

func itob(v int) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, uint64(v))
	return b
}
