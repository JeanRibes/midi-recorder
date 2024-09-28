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
	port2   int
}

var SinkUI chan Message
var SinkLoop chan Message
var MasterControl chan Message

func init() {
	SinkUI = make(chan Message, 10)
	SinkLoop = make(chan Message, 10)
}
