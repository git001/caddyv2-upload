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
			case "root_dir":
				if !d.Args(&u.RootDir) {
					return d.ArgErr()
				}
			case "file_field_name":
				if !d.Args(&u.FileFieldName) {
					return d.ArgErr()
				}
			case "fixed_file_name":
				if !d.Args(&u.FixedFileName) {
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
				if d.NextArg() {
					return d.Err("insecure is a flag and does not take a value")
				}
				u.MyTlsSetting.InsecureSkipVerify = true
			case "capath":
				if !d.Args(&u.MyTlsSetting.CAPath) {
					return d.ArgErr()
				}
			case "create_uuid_dir":
				uuidDirBool, err := parseOptionalBool(d, "create_uuid_dir")
				if err != nil {
					return err
				}
				u.CreateUuidDir = uuidDirBool
			case "dest_dir_field_name":
				if !d.Args(&u.DestDirFieldName) {
					return d.ArgErr()
				}
			case "pass_thru":
				passThruBool, err := parseOptionalBool(d, "pass_thru")
				if err != nil {
					return err
				}
				u.PassThru = passThruBool
			default:
				return d.Errf("unrecognized servers option '%s'", d.Val())
			}
		}
	}
	return nil
}

// parseOptionalBool parses a flag-style option. Without an argument the
// option is enabled; an optional single argument may be true or false.
func parseOptionalBool(d *caddyfile.Dispenser, name string) (bool, error) {
	if !d.NextArg() {
		return true, nil
	}
	b, err := strconv.ParseBool(d.Val())
	if err != nil {
		return false, d.Errf("parsing %s: %v", name, err)
	}
	if d.NextArg() {
		return false, d.ArgErr()
	}
	return b, nil
}

// parseCaddyfile parses the upload directive. It enables the upload
// of a file:
//
//	upload {
//		capath				<CA Path>
//		create_uuid_dir		[true | false]
//		dest_dir			<destination directory>
//		dest_dir_field_name	<form field name>
//		file_field_name		<form field name>
//		fixed_file_name		<file name>
//		insecure
//		max_filesize		<Humanized size>
//		max_filesize_int	<size>
//		max_form_buffer		<Humanized size>
//		max_form_buffer_int	<size>
//		notify_method		<method>
//		notify_url			<URL>
//		pass_thru			[true | false]
//		response_template	[<path to a response template>]
//		root_dir			<root directory>
//	}
func parseCaddyfile(h httpcaddyfile.Helper) (caddyhttp.MiddlewareHandler, error) {
	var u Upload
	err := u.UnmarshalCaddyfile(h.Dispenser)
	return u, err
}
