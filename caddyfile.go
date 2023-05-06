package upload

import (
	"strconv"

	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
	"github.com/caddyserver/caddy/v2/caddyconfig/httpcaddyfile"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
	"github.com/dustin/go-humanize"
)

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
			case "create_uuid_dir":
				var uuidDirStr string
				if !d.AllArgs(&uuidDirStr) {
					return d.ArgErr()
				}
				uuidDirBool, err := strconv.ParseBool(uuidDirStr)
				if err != nil {
					return d.Errf("parsing create_uuid_dir: %v", err)
				}
				u.CreateUuidDir = uuidDirBool
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
//	upload {
//		capath				<CA Path>
//		create_uuid_dir		true | false
//		dest_dir			<destination directory>
//		insecure
//		max_filesize		<Humanized size>
//		max_filesize_int	<size>
//		response_template	[<path to a response template>]
//	}
func parseCaddyfile(h httpcaddyfile.Helper) (caddyhttp.MiddlewareHandler, error) {
	var u Upload
	err := u.UnmarshalCaddyfile(h.Dispenser)
	return u, err
}
