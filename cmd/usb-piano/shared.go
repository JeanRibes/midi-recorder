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
	RestartMIDI
	BankStateChange
	BankDragDrop
	BankClear
	BankLengthNotify
	ResetStep
)

type Message struct {
	ev      Event
	number  int
	boolean bool
	str     string
	port2   int
}

var SinkLoop chan Message
var SinkUI chan Message
var MasterControl chan Message

func init() {
	SinkLoop = make(chan Message, 10)
	SinkUI = make(chan Message, 10)
}

var LoopDied bool = false
