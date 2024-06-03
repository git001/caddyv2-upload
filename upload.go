package upload

import (
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
	"github.com/caddyserver/caddy/v2/caddyconfig/httpcaddyfile"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
	"github.com/dustin/go-humanize"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

const (
	Version = "0.17"
)

func init() {
	caddy.RegisterModule(Upload{})
	httpcaddyfile.RegisterHandlerDirective("upload", parseCaddyfile)
}

// Middleware implements an HTTP handler that writes the
// uploaded file  to a file on the disk.
type Upload struct {
	DestDir          string `json:"dest_dir,omitempty"`
	RootDir          string `json:"root_dir,omitempty"`
	FileFieldName    string `json:"file_field_name,omitempty"`
	FixedFileName    string `json:"fixed_file_name,omitempty"`
	MaxFilesize      int64  `json:"max_filesize_int,omitempty"`
	MaxFilesizeH     string `json:"max_filesize,omitempty"`
	MaxFormBuffer    int64  `json:"max_form_buffer_int,omitempty"`
	MaxFormBufferH   string `json:"max_form_buffer,omitempty"`
	ResponseTemplate string `json:"response_template,omitempty"`
	NotifyURL        string `json:"notify_url,omitempty"`
	NotifyMethod     string `json:"notify_method,omitempty"`
	CreateUuidDir    bool   `json:"create_uuid_dir,omitempty"`
	DestDirFieldName string `json:"dest_dir_field_name,omitempty"`

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

	if u.DestDir == "" && u.DestDirFieldName == "" {
		u.logger.Error("Provision",
			zap.String("msg", "no Destination Directory specified (dest_dir or dest_dir_field_name)"))
		return fmt.Errorf("no Destination Directory specified (dest_dir or dest_dir_field_name)")
	}

	if u.RootDir == "" {
		u.logger.Info("Provision",
			zap.String("msg", "no Root Directory specified (root_dir), will use {http.vars.root}"))
	}

	if u.FileFieldName == "" {
		u.logger.Warn("Provision",
			zap.String("msg", "no FileFieldName specified (file_field_name), using the default one 'myFile'"),
		)
		u.FileFieldName = "myFile"
	}

	if u.FixedFileName == "" {
		u.logger.Info("Provision",
			zap.String("msg", "no fixed file name specified (fixed_file_name), will use file name from header"))
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
		zap.String("root_dir", u.RootDir),
		zap.String("fixed_file_name", u.FixedFileName),
		zap.Int64("max_filesize_int", u.MaxFilesize),
		zap.String("max_filesize", u.MaxFilesizeH),
		zap.Int64("max_form_buffer_int", u.MaxFormBuffer),
		zap.String("max_form_buffer", u.MaxFormBufferH),
		zap.String("response_template", u.ResponseTemplate),
		zap.String("notify_method", u.NotifyMethod),
		zap.String("notify_url", u.NotifyURL),
		zap.Bool("CreateUuidDir", u.CreateUuidDir),
		zap.String("capath", u.MyTlsSetting.CAPath),
		zap.Bool("insecure", u.MyTlsSetting.InsecureSkipVerify),
		zap.String("dest_dir_field_name", u.DestDirFieldName),
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

	// cleanup uploaded files see
	// https://github.com/git001/caddyv2-upload/issues/13
	defer r.MultipartForm.RemoveAll()

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

	if u.RootDir == "" {
		var rootDirErr bool
		u.RootDir, rootDirErr = repl.GetString("http.vars.root")

		if !rootDirErr {
			u.logger.Error("Provision",
				zap.Bool("rootDirErr", rootDirErr),
				zap.String("rootDir", u.RootDir),
				zap.String("message", "Variable {http.vars.root} not defined"))
			return caddyhttp.Error(http.StatusInternalServerError, fmt.Errorf("can't find root dir"))
		}
	}

	if u.DestDirFieldName != "" {
		u.DestDir = r.FormValue(u.DestDirFieldName)
	}

	concatDir := caddyhttp.SanitizedPathJoin(u.RootDir, u.DestDir)

	if u.CreateUuidDir {
		uuidDir := uuid.New().String()

		// It's very unlikely that the uuidDir already exists, but just in case
		for {
			if _, err := os.Stat(caddyhttp.SanitizedPathJoin(concatDir, uuidDir)); os.IsNotExist(err) {
				break
			} else {
				uuidDir = uuid.New().String()
			}
		}

		concatDir = caddyhttp.SanitizedPathJoin(concatDir, uuidDir)
		repl.Set("http.upload.uuiddir", uuidDir)
	}

	if err := os.MkdirAll(concatDir, 0755); err != nil {
		u.logger.Error("UUID directory creation error",
			zap.String("requuid", requuid),
			zap.String("message", "Failed to create "+concatDir),
			zap.Error(err),
			zap.Object("request", caddyhttp.LoggableHTTPRequest{Request: r}))
		return caddyhttp.Error(http.StatusInternalServerError, err)
	}

	// Create the file within the DestDir directory

	var filename string = handler.Filename
	if u.FixedFileName != "" {
		filename = u.FixedFileName
	}

	tempFile, tmpf_err := os.OpenFile(caddyhttp.SanitizedPathJoin(concatDir, filename), os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0755)

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
	fileBytes, io_err := io.Copy(tempFile, file)
	if io_err != nil {
		u.logger.Error("Copy Error",
			zap.String("requuid", requuid),
			zap.String("message", "Error at io.Copy"),
			zap.Error(io_err),
			zap.Object("request", caddyhttp.LoggableHTTPRequest{Request: r}))
		return caddyhttp.Error(http.StatusInternalServerError, io_err)
	}

	u.logger.Info("Successful Upload Info",
		zap.String("requuid", requuid),
		zap.String("Uploaded File", filename),
		zap.Int64("File Size", handler.Size),
		zap.Int64("written-bytes", fileBytes),
		zap.String("UploadDir", concatDir),
		zap.Any("MIME Header", handler.Header),
		zap.Object("request", caddyhttp.LoggableHTTPRequest{Request: r}))

	repl.Set("http.upload.filename", filename)
	repl.Set("http.upload.filesize", handler.Size)
	repl.Set("http.upload.directory", concatDir)

	if u.NotifyURL != "" {
		errNotify := u.SendNotify(requuid)

		if errNotify != nil {
			u.logger.Error("Notify Error",
				zap.Error(errNotify),
			)
		}
	}

	if u.ResponseTemplate != "" {

		fileRespTemplate, fRTErr := os.Open(caddyhttp.SanitizedPathJoin(u.RootDir, u.ResponseTemplate))
		if fRTErr != nil {
			u.logger.Error("File Response Template open Error",
				zap.String("requuid", requuid),
				zap.Any("fileRespTemplate", fileRespTemplate),
				zap.String("message", "Error at os.Open"),
				zap.Error(fRTErr),
				zap.Object("request", caddyhttp.LoggableHTTPRequest{Request: r}))
			return caddyhttp.Error(http.StatusInternalServerError, fRTErr)
		}
		defer fileRespTemplate.Close()

		// get information about the file
		info, fRTSTErr := fileRespTemplate.Stat()
		if fRTSTErr != nil {
			u.logger.Error("File Response Template Stat Error",
				zap.String("requuid", requuid),
				zap.Any("fileRespTemplate", fileRespTemplate),
				zap.String("message", "Error at fileRespTemplate.Stat"),
				zap.Error(fRTSTErr),
				zap.Object("request", caddyhttp.LoggableHTTPRequest{Request: r}))
			return caddyhttp.Error(http.StatusInternalServerError, fRTSTErr)
		}

		http.ServeContent(w, r, info.Name(), info.ModTime(), fileRespTemplate)
		return nil
	}

	return nil
}

// Interface guards
var (
	_ caddy.Provisioner           = (*Upload)(nil)
	_ caddy.Validator             = (*Upload)(nil)
	_ caddyhttp.MiddlewareHandler = (*Upload)(nil)
	_ caddyfile.Unmarshaler       = (*Upload)(nil)
)
