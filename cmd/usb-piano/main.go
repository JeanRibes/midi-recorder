package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"time"

	charmlog "github.com/charmbracelet/log"
	"gitlab.com/gomidi/midi/v2"
	"gitlab.com/gomidi/midi/v2/drivers"
	rtmididrv "gitlab.com/gomidi/midi/v2/drivers/rtmididrv"
)

var BPM = float64(120)

func main() {
	logger := charmlog.NewWithOptions(os.Stdout, charmlog.Options{
		Level: charmlog.DebugLevel,
		//ReportCaller:    true,
		ReportTimestamp: false,
		Prefix:          "main",
	})
	logger.Info("start")
	for _, port := range midi.GetInPorts() {
		logger.Debug("input port", "number", port.Number(), "name", port.String())
	}
	for _, port := range midi.GetOutPorts() {
		//log.Printf("%#v\n", port)
		logger.Debug("output port", "number", port.Number(), "name", port.String())
	}

	inPort := flag.String("input", "LPK25 mk2 MIDI 1", "MIDI input port name")
	outPort := flag.String("output", "Synth input port", "MIDI output port name")

	flag.Parse()

	out, err := midi.FindOutPort(*outPort)
	if err != nil {
		logger.Warn("can't find output, opening one")
		out, err = drv.OpenVirtualOut("step-recorder")
		he(err)
	}

	in, err := midi.FindInPort(*inPort)
	if err != nil {
		logger.Warn("can't find input, opening one")
		in, err = drv.OpenVirtualIn("step-recorder")
		he(err)
	}

	state := NewState()

	TMPFILE := os.Getenv("XDG_RUNTIME_DIR") + "/usb-piano.mid"
	if err := state.LoadFromFile(TMPFILE); err != nil {
		logger.Error(err)
	}

	//he(in.Open()) // crée une double connexion ! (agression auditive !!!)
	//he(out.Open())

	mainCtx, mainCancel := context.WithCancel(context.Background())

	uiCtx, cancelUi := context.WithCancel(mainCtx)
	loopCtx, cancelLoop := context.WithCancel(mainCtx)
	cancelUi()
	cancelLoop()
	//go ui(uiCtx, cancelUi)
	//go loop(loopCtx, cancelLoop, in, out)

	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, os.Interrupt)

	MasterControl = make(chan Message, 10)

masterLoop:
	for {
		select {
		case <-signalCh:
			logger.Debug("\ninterrupt")
			MasterControl <- Message{ev: Quit}
		case m := <-MasterControl:
			switch m.ev {
			case Quit:
				logger.Info("shutting down")
				mainCancel()
				break masterLoop
			case RestartMIDI:
				inN := m.number
				outN := m.port2
				logger.Info("reconnect", "input", inN, "output", outN)
				cancelLoop()
				for {
					if !out.IsOpen() {
						break
					}
					time.Sleep(time.Millisecond) //race condition
				}
				in, err = midi.InPort(inN)
				if err != nil {
					logger.Error(err)
					continue
				}
				out, err = midi.OutPort(outN)
				if err != nil {
					logger.Error(err)
					continue
				}
				logger.Info("reconnect", "input", in.String())
				logger.Info("reconnect", "output", out.String())
				he(drv.Close())
				midi.CloseDriver()

			}
		case <-uiCtx.Done():
			// UI closed, on relance
			uiCtx, cancelUi = context.WithCancel(mainCtx)
			//time.Sleep(time.Second)
			logger.Info("mc: restart UI")
			inPortsNames, inPortsNumbers, outPortsNames, outPortsNumbers := listPorts()
			inN := 0
			outN := 0
			if in != nil && out != nil {
				inN = in.Number()
				outN = out.Number()
			}
			go ui(uiCtx, cancelUi, inN, outN, inPortsNames, inPortsNumbers, outPortsNames, outPortsNumbers)
		case <-loopCtx.Done():
			loopCtx, cancelLoop = context.WithCancel(mainCtx)
			logger.Info("mc: restart Loop")
			if !LoopDied {
				go loop(loopCtx, cancelLoop, in, out, state)
			}
		}
	}

	if err := state.SaveToFile(TMPFILE); err != nil {
		logger.Error(err)
	}

	drv.Close()
	midi.CloseDriver()
}

func listPorts() (inL []string, inN []int, outL []string, outN []int) {
	for _, port := range midi.GetInPorts() {
		inL = append(inL, port.String())
		inN = append(inN, port.Number())
	}
	for _, port := range midi.GetOutPorts() {
		outL = append(outL, port.String())
		outN = append(outN, port.Number())
	}
	return
}

var drv *rtmididrv.Driver = drivers.Get().(*rtmididrv.Driver)

func he(err error) {
	if err != nil {
		//log.Fatal("fatale, ", err.Error())
		panic(err)
	}
}
