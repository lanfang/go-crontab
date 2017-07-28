package toplevel

import (
	"fmt"
	"github.com/lanfang/go-lib/log"
	"os"
	"os/signal"
	"runtime/pprof"
	"sync/atomic"
	"syscall"
)

var g_cpupro_file string = ""
var g_cpupro_fp *os.File
var g_mempro_file string = ""
var g_cpupro_roll int64

func InitCPUProfile() error {
	if g_cpupro_file == "" {
		return nil
	}

	roll := atomic.AddInt64(&g_cpupro_roll, 1)
	fileName := fmt.Sprintf("%s.%05d", g_cpupro_file, roll)

	var err error
	g_cpupro_fp, err = os.OpenFile(fileName, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return err
	}

	pprof.StartCPUProfile(g_cpupro_fp)

	return nil
}

type SignalHandler func(sig os.Signal, arg interface{})

type SignalSet struct {
	sig_map map[os.Signal]SignalHandler
}

func NewSignalSet() *SignalSet {
	signal_set := new(SignalSet)
	signal_set.sig_map = make(map[os.Signal]SignalHandler)
	return signal_set
}

func (signal_set *SignalSet) Register(sig os.Signal, handler SignalHandler) {
	if _, found := signal_set.sig_map[sig]; !found {
		signal_set.sig_map[sig] = handler
	}
}

func (signal_set *SignalSet) Handle(sig os.Signal, arg interface{}) {
	if _, found := signal_set.sig_map[sig]; found {
		signal_set.sig_map[sig](sig, arg)
	} else {
		log.Warning("No handler available for signal:%+v, Ignore", sig)
	}
}

func handleMemProfile() error {
	if g_mempro_file == "" {
		return nil
	}

	fp, err := os.OpenFile(g_mempro_file, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		log.Warning("failed to open memory profile file[%s]", g_mempro_file)
		return err
	}

	pprof.WriteHeapProfile(fp)
	fp.Close()

	return nil
}

func handleSigDumpMem(sig os.Signal, arg interface{}) {
	handleMemProfile()
	if g_cpupro_file != "" {
		pprof.StopCPUProfile()
		if g_cpupro_fp != nil {
			g_cpupro_fp.Close()
			g_cpupro_fp = nil
		}

		InitCPUProfile()
	}
}

func StartHandleSignal() {
	signal_set := NewSignalSet()

	signal_set.Register(syscall.SIGUSR1, handleSigDumpMem)

	for {
		signal_chan := make(chan os.Signal)
		signal.Notify(signal_chan, syscall.SIGUSR1)

		sig := <-signal_chan
		signal_set.Handle(sig, nil)
	}
}
