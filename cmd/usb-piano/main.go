package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"

	"gitlab.com/gomidi/midi/v2"
	"gitlab.com/gomidi/midi/v2/drivers"
	rtmididrv "gitlab.com/gomidi/midi/v2/drivers/rtmididrv"
)

var BPM = float64(120)

func main() {
	for _, port := range midi.GetInPorts() {
		//log.Println(port.Number(), port, port.Number(), reflect.TypeOf(port.Underlying()))
		log.Printf("%#v\n", port)
	}
	log.Println("---")
	for _, port := range midi.GetOutPorts() {
		log.Printf("%#v\n", port)
		//log.Println(port.Number(), port, port.Number(), reflect.TypeOf(port.Underlying()))
	}

	inPort := flag.String("input", "LPK25 mk2 MIDI 1", "MIDI input port name")
	outPort := flag.String("output", "Synth input port", "MIDI output port name")

	flag.Parse()

	in, err := midi.FindInPort(*inPort)
	if err != nil {
		fmt.Println("can't find input, opening one")
		in, err = drv.OpenVirtualIn("step-recorder")
		he(err)
	}

	out, err := midi.FindOutPort(*outPort)
	if err != nil {
		fmt.Println("can't find output")
		out, err = drv.OpenVirtualOut("step-recorder")
		he(err)
	}

	he(in.Open())
	he(out.Open())

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
			println("\ninterrupt")
			MasterControl <- Message{ev: Quit}
		case m := <-MasterControl:
			switch m.ev {
			case Quit:
				log.Println("main quit")
				mainCancel()
				break masterLoop
			case RestartMIDI:
				inN := m.number
				outN := m.port2
				log.Printf("reconnect %d %d\n", inN, outN)
				in, err = midi.InPort(inN)
				if err != nil {
					log.Println(err)
					continue
				}
				out, err = midi.OutPort(outN)
				if err != nil {
					log.Println(err)
					continue
				}
				log.Println("reconnect input:", in.String())
				log.Println("reconnect output:", out.String())
				cancelLoop()
				he(drv.Close())
				midi.CloseDriver()

			}
		case <-uiCtx.Done():
			// UI closed, on relance
			uiCtx, cancelUi = context.WithCancel(mainCtx)
			//time.Sleep(time.Second)
			log.Println("mc: restart UI")
			inL, inN, outL, outN := listPorts()
			go ui(uiCtx, cancelUi, in.Number(), out.Number(), inL, inN, outL, outN)
		case <-loopCtx.Done():
			loopCtx, cancelLoop = context.WithCancel(mainCtx)
			log.Println("mc: restart Loop")

			go loop(loopCtx, cancelLoop, in, out)
		}
	}

	he(drv.Close())
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
