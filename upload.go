package upload

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"

	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
	"github.com/caddyserver/caddy/v2/caddyconfig/httpcaddyfile"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
	"github.com/dustin/go-humanize"
	"go.uber.org/zap"
)

const (
	Version = "0.8"
)

func init() {
	caddy.RegisterModule(Upload{})
	httpcaddyfile.RegisterHandlerDirective("upload", parseCaddyfile)
}

// Middleware implements an HTTP handler that writes the
// uploaded file  to a file on the disk.
type Upload struct {
	DestDir          string `json:"dest_dir,omitempty"`
	FileFieldName    string `json:"file_field_name,omitempty"`
	MaxFilesize      int64  `json:"max_filesize_int,omitempty"`
	MaxFilesizeH     string `json:"max_filesize,omitempty"`
	MaxFormBuffer    int64  `json:"max_form_buffer_int,omitempty"`
	MaxFormBufferH   string `json:"max_form_buffer,omitempty"`
	ResponseTemplate string `json:"response_template,omitempty"`
	NotifyURL        string `json:"notify_url,omitempty"`
	NotifyMethod     string `json:"notify_method,omitempty"`

	MyTlsSetting struct {
		InsecureSkipVerify bool   `json:"insecure,omitempty"`
		CAPath             string `json:"capath,omitempty"`
	}

	// TODO: Handle notify Body

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

	repl := caddy.NewReplacer()

	if u.DestDir == "" {
		u.logger.Error("Provision",
			zap.String("msg", "no Destination Directory specified (dest_dir)"))
		return fmt.Errorf("no Destination Directory specified (dest_dir)")
	}

	mdall_err := os.MkdirAll(u.DestDir, 0755)
	if mdall_err != nil {
		u.logger.Error("Provision",
			zap.String("msg", "MkdirAll: Error creating destination Directory"),
			zap.Error(mdall_err))
		return mdall_err
	}

	if u.FileFieldName == "" {
		u.logger.Warn("Provision",
			zap.String("msg", "no FileFieldName specified (file_field_name), using the default one 'myFile'"),
		)
		u.FileFieldName = "myFile"
	}

	if u.ResponseTemplate == "" {
		u.logger.Warn("Provision",
			zap.String("msg", "no ResponseTemplate specified (response_template), using the default one"),
		)
		u.ResponseTemplate = "upload-resp-template.txt"
	}

	if u.NotifyURL != "" && u.NotifyMethod == "" {
		u.NotifyMethod = "GET"
	}

	if u.MaxFilesize == 0 && u.MaxFilesizeH != "" {

		MaxFilesizeH := repl.ReplaceAll(u.MaxFilesizeH, "1GB")
		u.MaxFilesizeH = MaxFilesizeH

		size, err := humanize.ParseBytes(u.MaxFilesizeH)
		if err != nil {
			u.logger.Error("Provision ReplaceAll",
				zap.String("msg", "MaxFilesizeH: Error parsing max_filesize"),
				zap.String("MaxFilesizeH", u.MaxFilesizeH),
				zap.Error(err))
			return err
		}
		u.MaxFilesize = int64(size)
	} else {
		if u.MaxFilesize == 0 {
			size, err := humanize.ParseBytes("1GB")
			if err != nil {
				u.logger.Error("Provision int",
					zap.String("msg", "MaxFilesize: Error parsing max_filesize_int"),
					zap.Int64("MaxFilesize", u.MaxFilesize),
					zap.Error(err))
				return err
			}
			u.MaxFilesize = int64(size)
		}
	}

	if u.MaxFormBuffer == 0 && u.MaxFormBufferH != "" {

		MaxFormBufferH := repl.ReplaceAll(u.MaxFormBufferH, "1GB")
		u.MaxFormBufferH = MaxFormBufferH

		size, err := humanize.ParseBytes(u.MaxFormBufferH)
		if err != nil {
			u.logger.Error("Provision ReplaceAll",
				zap.String("msg", "MaxFormBufferH: Error parsing max_form_buffer"),
				zap.String("MaxFormBufferH", u.MaxFormBufferH),
				zap.Error(err))
			return err
		}
		u.MaxFormBuffer = int64(size)
	} else {
		if u.MaxFormBuffer == 0 {
			size, err := humanize.ParseBytes("1GB")
			if err != nil {
				u.logger.Error("Provision int",
					zap.String("msg", "MaxFormBufferH: Error parsing max_form_buffer_int"),
					zap.Int64("MaxFormBuffer", u.MaxFormBuffer),
					zap.Error(err))
				return err
			}
			u.MaxFormBuffer = int64(size)
		}
	}

	u.logger.Info("Current Config",
		zap.String("Version", Version),
		zap.String("dest_dir", u.DestDir),
		zap.Int64("max_filesize_int", u.MaxFilesize),
		zap.String("max_filesize", u.MaxFilesizeH),
		zap.Int64("max_form_buffer_int", u.MaxFormBuffer),
		zap.String("max_form_buffer", u.MaxFormBufferH),
		zap.String("response_template", u.ResponseTemplate),
		zap.String("notify_method", u.NotifyMethod),
		zap.String("notify_url", u.NotifyURL),
		zap.String("capath", u.MyTlsSetting.CAPath),
		zap.Bool("insecure", u.MyTlsSetting.InsecureSkipVerify),
	)

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
			zap.Bool("requuiderr", requuiderr),
			zap.Object("request", caddyhttp.LoggableHTTPRequest{Request: r}))
	}

	repl.Set("http.upload.max_filesize", u.MaxFilesize)

	r.Body = http.MaxBytesReader(w, r.Body, u.MaxFilesize)
	if max_size_err := r.ParseMultipartForm(u.MaxFormBuffer); max_size_err != nil {
		u.logger.Error("ServeHTTP",
			zap.String("requuid", requuid),
			zap.String("message", "The uploaded file is too big. Please choose an file that's less than MaxFilesize."),
			zap.String("MaxFilesize", humanize.Bytes(uint64(u.MaxFilesize))),
			zap.Error(max_size_err),
			zap.Object("request", caddyhttp.LoggableHTTPRequest{Request: r}))
		return caddyhttp.Error(http.StatusRequestEntityTooLarge, max_size_err)
	}

	// FormFile returns the first file for the given file field key
	// it also returns the FileHeader so we can get the Filename,
	// the Header and the size of the file
	file, handler, ff_err := r.FormFile(u.FileFieldName)
	if ff_err != nil {
		u.logger.Error("FormFile Error",
			zap.String("requuid", requuid),
			zap.String("message", "Error Retrieving the File"),
			zap.Error(ff_err),
			zap.Object("request", caddyhttp.LoggableHTTPRequest{Request: r}))
		return caddyhttp.Error(http.StatusInternalServerError, ff_err)
	}
	defer file.Close()

	// Create the file within the DestDir directory

	tempFile, tmpf_err := os.OpenFile(u.DestDir+"/"+handler.Filename, os.O_RDWR|os.O_CREATE, 0755)

	if tmpf_err != nil {
		u.logger.Error("TempFile Error",
			zap.String("requuid", requuid),
			zap.String("message", "Error at TempFile"),
			zap.Error(tmpf_err),
			zap.Object("request", caddyhttp.LoggableHTTPRequest{Request: r}))
		return caddyhttp.Error(http.StatusInternalServerError, tmpf_err)
	}
	defer tempFile.Close()

	// read all of the contents of our uploaded file into a
	// byte array
	//fileBytes, io_err := ioutil.ReadAll(file)
	fileBytes, io_err := io.Copy(tempFile, file)
	if io_err != nil {
		u.logger.Error("Copy Error",
			zap.String("requuid", requuid),
			zap.String("message", "Error at io.Copy"),
			zap.Error(io_err),
			zap.Object("request", caddyhttp.LoggableHTTPRequest{Request: r}))
		return caddyhttp.Error(http.StatusInternalServerError, io_err)
	}
	// write this byte array to our temporary file
	//tempFile.Write(fileBytes)

	u.logger.Info("Successful Upload Info",
		zap.String("requuid", requuid),
		zap.String("Uploaded File", handler.Filename),
		zap.Int64("File Size", handler.Size),
		zap.Int64("written-bytes", fileBytes),
		zap.Any("MIME Header", handler.Header),
		zap.Object("request", caddyhttp.LoggableHTTPRequest{Request: r}))

	repl.Set("http.upload.filename", handler.Filename)
	repl.Set("http.upload.filesize", handler.Size)

	if u.ResponseTemplate != "" {
		r.URL.Path = "/" + u.ResponseTemplate
	}

	if u.NotifyURL != "" {
		errNotify := u.SendNotify(requuid)

		if errNotify != nil {
			u.logger.Error("Notify Error",
				zap.Error(errNotify),
			)
		}
	}

	return next.ServeHTTP(w, r)
}

// UnmarshalCaddyfile implements caddyfile.Unmarshaler.
func (u *Upload) UnmarshalCaddyfile(d *caddyfile.Dispenser) error {

	for d.Next() {
		for d.NextBlock(0) {
			switch d.Val() {

			case "dest_dir":
				if !d.Args(&u.DestDir) {
					return d.ArgErr()
				}
			case "file_field_name":
				if !d.Args(&u.FileFieldName) {
					return d.ArgErr()
				}
			case "max_form_buffer":
				var sizeStr string
				if !d.AllArgs(&sizeStr) {
					return d.ArgErr()
				}
				size, err := humanize.ParseBytes(sizeStr)
				if err != nil {
					return d.Errf("parsing max_form_buffer: %v", err)
				}
				u.MaxFormBuffer = int64(size)
			case "max_form_buffer_int":
				var sizeStr string
				if !d.AllArgs(&sizeStr) {
					return d.ArgErr()
				}
				i, err := strconv.ParseInt(sizeStr, 10, 64)
				if err != nil {
					return d.Errf("parsing max_form_buffer_int: %v", err)
				}
				u.MaxFormBuffer = i
			case "max_filesize":
				var sizeStr string
				if !d.AllArgs(&sizeStr) {
					return d.ArgErr()
				}
				size, err := humanize.ParseBytes(sizeStr)
				if err != nil {
					return d.Errf("parsing max_filesize: %v", err)
				}
				u.MaxFilesize = int64(size)
			case "max_filesize_int":
				var sizeStr string
				if !d.AllArgs(&sizeStr) {
					return d.ArgErr()
				}
				i, err := strconv.ParseInt(sizeStr, 10, 64)
				if err != nil {
					return d.Errf("parsing max_filesize_int: %v", err)
				}
				u.MaxFilesize = i
			case "response_template":
				if !d.Args(&u.ResponseTemplate) {
					return d.ArgErr()
				}
			case "notify_url":
				if !d.Args(&u.NotifyURL) {
					return d.ArgErr()
				}
			case "notify_method":
				if !d.Args(&u.NotifyMethod) {
					return d.ArgErr()
				}
			case "insecure":
				if !d.NextArg() {
					return d.ArgErr()
				}
				u.MyTlsSetting.InsecureSkipVerify = true
			case "capath":
				if !d.Args(&u.MyTlsSetting.CAPath) {
					return d.ArgErr()
				}
			default:
				return d.Errf("unrecognized servers option '%s'", d.Val())
			}
		}
	}
	return nil
}

// parseCaddyfile parses the upload directive. It enables the upload
// of a file:
//
//    upload {
//        dest_dir          <destination directory>
//        max_filesize      <size>
//        response_template [<path to a response template>]
//    }
//
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
