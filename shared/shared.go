package shared

import "fmt"

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
	StepBack
	BankImport
	BankExport
	BankCut
	NoteUndo
	ExportMultiTrack
	ClearState
)

type Message struct {
	Type    Event
	Number  int
	Boolean bool
	String  string
	Number2 int
}

var LoopDied bool = false
var BPM = float64(120)

const NUM_BANKS = 15

func BankName(bank int) string {
	switch bank {
	case 0:
		return "buffer"
	case 1:
		return "soprane"
	case 2:
		return "alto"
	case 3:
		return "ténor"
	case 4:
		return "basse"
	default:
		return fmt.Sprintf("piste %d", bank)
	}
}
