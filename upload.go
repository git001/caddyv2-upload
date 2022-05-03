package upload

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"

	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
	"github.com/caddyserver/caddy/v2/caddyconfig/httpcaddyfile"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
	"go.uber.org/zap"
)

func init() {
	caddy.RegisterModule(Upload{})
	httpcaddyfile.RegisterHandlerDirective("upload", parseCaddyfile)
}

// Middleware implements an HTTP handler that writes the
// uploaded file  to a file on the disk.
type Upload struct {
	DestDir          string `json:"dest_dir,omitempty"`
	MaxFilesize      int64  `json:"max_filesize,omitempty"`
	ResponseTemplate string `json:"response_template,omitempty"`

	ctx    caddy.Context
	logger *zap.Logger
}

// CaddyModule returns the Caddy module information.
func (Upload) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID:  "http.handlers.upload",
		New: func() caddy.Module { return new(Upload) },
	}
}

// Provision implements caddy.Provisioner.
func (u *Upload) Provision(ctx caddy.Context) error {
	u.ctx = ctx
	u.logger = ctx.Logger(u)

	if u.DestDir == "" {
		u.logger.Error("Provision",
			zap.String("msg", "no Destinaton Directory specified (dest_dir)"))
		return fmt.Errorf("no Destinaton Directory specified (dest_dir)")
	}

	if u.ResponseTemplate == "" {
		u.logger.Warn("Provision",
			zap.String("msg", "no ResponseTemplate specified (response_template)"))
	}

	mdall_err := os.MkdirAll(u.DestDir, 0755)
	if mdall_err != nil {
		u.logger.Error("Provision",
			zap.String("msg", "MkdirAll: Error creating destination Directory"),
			zap.Error(mdall_err))
		return mdall_err
	}

	u.logger.Info("Current Config",
		zap.String("Destinaton Directory (dest_dir)", u.DestDir),
		zap.Int64("Max filesize in bytes (max_filesize)", u.MaxFilesize),
		zap.String("Response Template (response_template)", u.ResponseTemplate))

	return nil
}

// Validate implements caddy.Validator.
func (u *Upload) Validate() error {
	// TODO: Do I need this func
	return nil
}

// ServeHTTP implements caddyhttp.MiddlewareHandler.
func (u Upload) ServeHTTP(w http.ResponseWriter, r *http.Request, next caddyhttp.Handler) error {

	repl := r.Context().Value(caddy.ReplacerCtxKey).(*caddy.Replacer)

	requuid, requuiderr := repl.GetString("http.request.uuid")
	if !requuiderr {
		requuid = "0"
		u.logger.Error("http.request.uuid",
			zap.Bool("requuiderr", requuiderr))
	}

	r.Body = http.MaxBytesReader(w, r.Body, u.MaxFilesize)
	if max_size_err := r.ParseMultipartForm(u.MaxFilesize); max_size_err != nil {
		u.logger.Error("ServeHTTP",
			zap.String("Request uuid", requuid),
			zap.String("message", "The uploaded file is too big. Please choose an file that's less than MaxFilesize."),
			zap.Int64("MaxFilesize in Bytes", u.MaxFilesize),
			zap.Error(max_size_err))
		return caddyhttp.Error(http.StatusRequestEntityTooLarge, max_size_err)
	}

	// FormFile returns the first file for the given key `myFile`
	// it also returns the FileHeader so we can get the Filename,
	// the Header and the size of the file
	file, handler, ff_err := r.FormFile("myFile")
	if ff_err != nil {
		u.logger.Error("FormFile Error",
			zap.String("Request uuid", requuid),
			zap.String("message", "Error Retrieving the File"),
			zap.Error(ff_err))
		return caddyhttp.Error(http.StatusInternalServerError, ff_err)
	}
	defer file.Close()

	// Create the file within the DestDir directory

	tempFile, tmpf_err := os.OpenFile(u.DestDir+"/"+handler.Filename, os.O_RDWR|os.O_CREATE, 0755)

	if tmpf_err != nil {
		u.logger.Error("TempFile Error",
			zap.String("Request uuid", requuid),
			zap.String("message", "Error at TempFile"),
			zap.Error(tmpf_err))
		return caddyhttp.Error(http.StatusInternalServerError, tmpf_err)
	}
	defer tempFile.Close()

	// read all of the contents of our uploaded file into a
	// byte array
	fileBytes, io_err := ioutil.ReadAll(file)
	if io_err != nil {
		u.logger.Error("ReadAll Error",
			zap.String("Request uuid", requuid),
			zap.String("message", "Error at ReadAll"),
			zap.Error(io_err))
		return caddyhttp.Error(http.StatusInternalServerError, io_err)
	}
	// write this byte array to our temporary file
	tempFile.Write(fileBytes)

	u.logger.Info("Successfull Upload Info",
		zap.String("Request uuid", requuid),
		zap.String("Uploaded File", handler.Filename),
		zap.Int64("File Size", handler.Size),
		zap.Any("MIME Header", handler.Header))

	repl.Set("http.upload.filename", handler.Filename)
	repl.Set("http.upload.filesize", handler.Size)

	if u.ResponseTemplate != "" {
		r.URL.Path = "/" + u.ResponseTemplate
	}

	return next.ServeHTTP(w, r)
}

// UnmarshalCaddyfile implements caddyfile.Unmarshaler.
// TODO: make Caddyfile config robust
func (u *Upload) UnmarshalCaddyfile(d *caddyfile.Dispenser) error {
	for d.Next() {
		if !d.Args(&u.DestDir) {
			return d.ArgErr()
		}

		if !d.NextArg() {
			return d.ArgErr()
		}

		MaxFilesize, err := strconv.ParseInt(d.Val(), 10, 64)
		if err != nil {
			return d.Errf("bad max file size number '%s': %v", d.Val(), err)
		}
		u.MaxFilesize = MaxFilesize
	}
	return nil
}

// parseCaddyfile unmarshals tokens from h into a new Middleware.
func parseCaddyfile(h httpcaddyfile.Helper) (caddyhttp.MiddlewareHandler, error) {
	var u Upload
	err := u.UnmarshalCaddyfile(h.Dispenser)
	return u, err
}

// Interface guards
var (
	_ caddy.Provisioner           = (*Upload)(nil)
	_ caddy.Validator             = (*Upload)(nil)
	_ caddyhttp.MiddlewareHandler = (*Upload)(nil)
	_ caddyfile.Unmarshaler       = (*Upload)(nil)
)
