package log

import (
	"github.com/op/go-logging"
	syslog "log"
	"os"
	"strings"
)

var format string = "%{level}: [%{time:2006-01-02 15:04:05.000}][%{pid}][%{module}][%{shortfile}(%{shortfunc})][%{message}]"

func init() {
	var servername string
	if i := strings.LastIndex(os.Args[0], "/"); i >= 0 {
		i++
		servername = os.Args[0][i:]
	} else {
		servername = os.Args[0]
	}
	Gen(servername, "./", servername +".log")
}

func Gen(servername, dirname, filename string) {
	InitLog(servername, dirname, filename, format)
}

func InitLog(servername, dirname, filename, logFormat string) {
	os.MkdirAll(dirname, 0777)
	info_log_fp, err := NewFileLogWriter(dirname+"/"+filename, true, 1024*1024*1024)
	if err != nil {
		syslog.Fatalf("open file[%s] failed[%s]", filename, err)
		return
	}

	err_log_fp, err := NewFileLogWriter(dirname+"/"+filename+".wf", true, 1024*1024*1024)
	if err != nil {
		syslog.Fatalf("open file[%s.wf] failed[%s]", filename, err)
		return
	}

	backend_info := logging.NewLogBackend(info_log_fp, "", 0)
	backend_err := logging.NewLogBackend(err_log_fp, "", 0)
	format := logging.MustStringFormatter(logFormat)
	backend_info_formatter := logging.NewBackendFormatter(backend_info, format)
	backend_err_formatter := logging.NewBackendFormatter(backend_err, format)

	backend_info_leveld := logging.AddModuleLevel(backend_info_formatter)
	backend_info_leveld.SetLevel(logging.INFO, "")

	backend_err_leveld := logging.AddModuleLevel(backend_err_formatter)
	backend_err_leveld.SetLevel(logging.WARNING, "")

	logging.SetBackend(backend_info_leveld, backend_err_leveld)

	lg := logging.MustGetLogger(servername)
	lg.ExtraCalldepth += 1
	commonLogger = lg
}
