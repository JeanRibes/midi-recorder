package main

type Event int

const (
	Quit Event = iota
	Record
	PlayPause
	Quantize
	StepMode
	LoadFromFile
	SaveToFile
	Error
)

type Message struct {
	ev      Event
	number  int
	boolean bool
	str     string
}

var BusFromUItoLoop chan Message
var BusFromLoopToUI chan Message

func init() {
	BusFromUItoLoop = make(chan Message, 10)
	BusFromLoopToUI = make(chan Message, 10)
}
