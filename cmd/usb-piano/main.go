package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"time"

	"gitlab.com/gomidi/midi/v2"
	"gitlab.com/gomidi/midi/v2/drivers"
	rtmididrv "gitlab.com/gomidi/midi/v2/drivers/rtmididrv"
	"gitlab.com/gomidi/midi/v2/smf"
	"gitlab.com/gomidi/quantizer/lib/quantizer"
)

var BPM = float64(120)

func main() {
	inPort := flag.String("input", "LPK25 mk2 MIDI 1", "MIDI input port name")
	outPort := flag.String("output", "Synth input port", "MIDI output port name")

	flag.Parse()
	defer midi.CloseDriver()
	drv := drivers.Get().(*rtmididrv.Driver)
	in, err := midi.FindInPort(*inPort)
	if err != nil {
		fmt.Println("can't find input, opening one")
		in, err = drv.OpenVirtualIn("step-recorder")
		he(err)
	}
	println("input:", in.String())

	out, err := midi.FindOutPort(*outPort)
	if err != nil {
		fmt.Println("can't find output")
		out, err = drv.OpenVirtualOut("step-recorder")
		he(err)
	}
	println("output:", out.String())

	s := smf.New()

	go ui()

	send, err := midi.SendTo(out)
	he(err)
	/*send(midi.NoteOn(0, 70, 80))
	time.Sleep(100 * time.Millisecond)
	he(send(midi.NoteOff(0, 70)))*/

	main := smf.Track{}
	main.Add(0, smf.MetaTempo(BPM))

	var absmillisec int32 = 0
	TICKS := s.TimeFormat.(smf.MetricTicks)

	shoudStartRecording := false
	isRecording := false
	var recordingKey uint8

	isSteps := false
	stepIndex := 0
	stepNotes := []uint8{}
	stop, err := midi.ListenTo(in, func(msg midi.Message, absms int32) {
		var ch, key, vel uint8
		switch {
		case msg.GetNoteOn(&ch, &key, &vel) || msg.GetNoteEnd(&ch, &key):
			on := msg.IsOneOf(midi.NoteOnMsg)

			if isSteps {
				note := stepNotes[stepIndex]
				if on {
					send(midi.NoteOn(ch, note, vel))
				} else {
					send(midi.NoteOn(ch, note, vel))
					stepIndex += 1
					if stepIndex > len(stepNotes)-1 {
						stepIndex = 0
					}
				}
			}
			if shoudStartRecording && on {
				// START record
				shoudStartRecording = false
				isRecording = true
				recordingKey = key
				main = smf.Track{}
				main.Add(0, smf.MetaTempo(BPM))
				println("start rec")
				BusFromLoopToUI <- Message{ev: RecordStart, number: int(key)}
				absmillisec = 0
				return
			}
			if isRecording {
				if key == recordingKey && on {
					//STOP RECORD
					BusFromLoopToUI <- Message{ev: RecordStop}
					main.Close(0)
					isRecording = false
				} else {
					send(msg)
					deltams := absms - absmillisec
					absmillisec = absms
					delta := TICKS.Ticks(BPM, time.Duration(deltams)*time.Millisecond)
					main.Add(delta, msg)
				}
			} else {
				send(msg)
			}
		case msg.GetControlChange(&ch, &key, &vel):
			println(ch, key, vel)
			if isRecording && vel == 127 {
				//STOP record
				BusFromLoopToUI <- Message{ev: RecordStop}
				main.Close(0)
				isRecording = false
			}
			if shoudStartRecording && vel == 127 {
				// START record
				shoudStartRecording = false
				isRecording = true
				recordingKey = 0
				main = smf.Track{}
				main.Add(0, smf.MetaTempo(BPM))
				println("start rec")
				BusFromLoopToUI <- Message{ev: RecordStart, number: 0}
				absmillisec = 0
			}
		}
	})
	he(err)
	quit := make(chan struct{}, 1)

	go func() {
		signalCh := make(chan os.Signal, 1)
		signal.Notify(signalCh, os.Interrupt)
		<-signalCh
		println("interrupt")
		BusFromUItoLoop <- Message{ev: Quit}
	}()
	go func() {
		for {
			msg := <-BusFromUItoLoop
			switch msg.ev {
			case RecordStart:
				println("loop: record SYN")
				time.Sleep(time.Second)
				shoudStartRecording = true
			case PlayPause:
				go func() {
					var bf bytes.Buffer
					tmpFile := smf.New()
					tmpFile.Add(main)
					_, err = tmpFile.WriteTo(&bf)
					he(err)
					player := smf.ReadTracksFrom(&bf)
					he(player.Play(out))
					BusFromLoopToUI <- Message{ev: PlayPause}
					println("end play")
				}()
			case Quit:
				println("quit")
				stop()
				quit <- struct{}{}
			case Quantize:
				go func() {
					fmt.Printf("quantize at %d BPM\n", msg.number)
					var bf bytes.Buffer
					tmpFile := smf.New()
					main[0].Message = smf.MetaTempo(float64(msg.number))
					tmpFile.Add(main)
					_, err = tmpFile.WriteTo(&bf)
					he(quantizer.Quantize(&bf, &bf))
					main = smf.ReadTracksFrom(&bf).SMF().Tracks[0]
					BusFromLoopToUI <- Message{ev: Quantize}
					println("quantize done")
				}()
			case StepMode:
				isSteps = !isSteps
				println("set bool mode to", isSteps)
				BusFromLoopToUI <- Message{ev: StepMode, boolean: isSteps}
			}
		}
	}()
	<-quit

	/*
		main.SendTo(TICKS, smf.TempoChanges{},
			func(m midi.Message, timestampms int32) {
				he(send(m))
				deltams := timestampms - absmillisec
				absmillisec = timestampms
				// time.Sleep(TICKS.Duration(BPM, uint32(deltams)) / time.Millisecond)
				// println(deltams)
				println(timestampms)
				_ = deltams
				//t := time.Duration((deltams * int32(time.Millisecond)) / 2)
				//time.Sleep(t)
			})
	*/

}

func he(err error) {
	if err != nil {
		panic(err)
	}
}

func trackToSteps(tr *smf.Track) []uint8 {
	steps := []uint8{}
	var ch, key, vel uint8
	for _, ev := range *tr {
		if ev.Message.GetNoteOn(&ch, &key, &vel) {
			steps = append(steps, key)
		}
	}
	return steps
}
