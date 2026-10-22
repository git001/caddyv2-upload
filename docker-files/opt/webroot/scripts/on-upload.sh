#!/bin/sh
# Demo handler for the upload_finished event, started by events.handlers.exec.
# It prints all values the event provides, see config/Caddyfile-events
# for the matching argument order:
#   $1  {event.data.filepath}   full path of the uploaded file
#   $2  {event.data.filesize}   file size in bytes
#   $3  {event.data.filename}   file name (from the client or fixed_file_name)
#   $4  {event.data.directory}  directory of the file
#   $5  {event.data.uuiddir}    UUID directory name, empty without create_uuid_dir
#   $6  {event.data.requuid}    http.request.uuid of the upload request
#   $7  {event.id}              unique ID of this event
#   $8  {event.name}            event name (upload_finished)
#   $9  {event.time}            time the event was emitted
#   $10 {event.time_unix}       same as Unix timestamp in milliseconds
#   $11 {event.module}          module that emitted the event (http.handlers.upload)
#
# Exit code 3 rejects the upload when the handler runs in the foreground
# with abort_codes [3] and the upload handler has "on_abort abort".
#
# Environment:
#   SCAN_TIMEOUT  max. seconds for the scan (default 20). Keep it below the
#                 exec timeout (default 30s): when exec kills this script,
#                 the upload is accepted. Here a timeout rejects it instead.
#   SCAN_DELAY    seconds to sleep before the scan, to simulate a slow scanner
#                 (default 0)

file="$1"

echo "on-upload: filepath=$1"
echo "on-upload: filesize=$2"
echo "on-upload: filename=$3"
echo "on-upload: directory=$4"
echo "on-upload: uuiddir=$5"
echo "on-upload: requuid=$6"
echo "on-upload: event.id=$7"
echo "on-upload: event.name=$8"
echo "on-upload: event.time=$9"
echo "on-upload: event.time_unix=${10}"
echo "on-upload: event.module=${11}"
echo "on-upload: sha256=$(sha256sum -- "$file" | cut -d' ' -f1)"

# simple stand-in for a virus scanner which detects the EICAR test string.
# The file path is passed as argument to sh -c, never put into the script text.
timeout "${SCAN_TIMEOUT:-20}" sh -c 'sleep "$1"; grep -q "EICAR-STANDARD-ANTIVIRUS-TEST-FILE" -- "$2"' \
	scan "${SCAN_DELAY:-0}" "$file"
rc=$?

# fail closed: everything except "clean" rejects the upload
case $rc in
	1)   echo "on-upload: clean $file" ;;
	0)   echo "on-upload: rejecting $file (EICAR found)"; exit 3 ;;
	124) echo "on-upload: rejecting $file (scan timeout)"; exit 3 ;;
	*)   echo "on-upload: rejecting $file (scan error $rc)"; exit 3 ;;
esac
