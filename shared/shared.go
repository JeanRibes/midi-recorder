package shared

type Event int

const (
	Quit Event = iota
	Record
	PlayPause
	Quantize
	StepMode
	StateImport
	StateExport
	Error
	RestartMIDI
	BankStateChange
	BankDragDrop
	BankClear
	BankLengthNotify
	ResetStep
	BankImport
	BankExport
	BankCut
)

type Message struct {
	Type    Event
	Number  int
	Boolean bool
	String  string
	Number2 int
}

var SinkLoop chan Message
var SinkUI chan Message
var MasterControl chan Message

func init() {
	SinkLoop = make(chan Message, 10)
	SinkUI = make(chan Message, 10)
}

var LoopDied bool = false
var BPM = float64(120)
