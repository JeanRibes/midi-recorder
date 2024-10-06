package ui

//go:generate python3 genGoGlade.py

import (
	"context"
	_ "embed"
	"fmt"
	"os"
	"strconv"
	"time"

	. "github.com/JeanRibes/midi/shared"

	charmlog "github.com/charmbracelet/log"
	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/glib"
	"github.com/gotk3/gotk3/gtk"
)

//go:embed ui.glade
var gladeUiXML string

//go:embed ui.css
var stylesheet string

type DragDropSrc byte

const (
	Bank DragDropSrc = iota
	ImportZone
)

func Run(ctx context.Context, cancel func(), inP, outP int, inL []string, inN []int, outL []string, outN []int, SinkUI, SinkLoop, MasterControl chan Message) {
	logger := charmlog.NewWithOptions(os.Stdout, charmlog.Options{
		Level:           charmlog.DebugLevel,
		ReportCaller:    true,
		ReportTimestamp: false,
		Prefix:          "UI",
	})
	logger.Info("start")
	gtk.Init(nil)

	//builder, err := gtk.BuilderNewFromFile("ui.glade") //remplacer par du embed
	builder, err := gtk.BuilderNewFromString(gladeUiXML)
	if err != nil {
		logger.Fatal(err)
	}
	loadUI(builder)

	if err != nil {
		logger.Error("Unable to create window:", err)
	}
	windestroyhandle := mainWin.Connect("destroy", func() {
		//cancel()
		logger.Debug("close win, sending quit event")
		MasterControl <- Message{Type: Quit}
		quitCtx, _ := context.WithDeadline(context.Background(), time.Now().Add(time.Second*5))
		<-quitCtx.Done()
		logger.Warn("took too long to shutdown")
		os.Exit(3)

	})

	recordBtn.SetImage(startrecordImg)
	recordBtn.Connect("clicked", func() {
		recordBtn.SetSensitive(false)
		SinkLoop <- Message{Type: Record}
	})

	playBtn.Connect("clicked", func() {
		SinkLoop <- Message{Type: PlayPause}
	})

	quantizeBtn.Connect("clicked", func() {
		quantizeBtn.SetSensitive(false)
		txt, err := bpmEntry.GetText()

		ticksEntry.GetText()
		if err == nil {
			var i int64
			if i, err = strconv.ParseInt(txt, 10, 32); err == nil {
				SinkLoop <- Message{Type: Quantize, Number: int(i)}
			} else {
				logger.Error(err)
				quantizeBtn.SetSensitive(true)
			}
		} else {
			logger.Error(err)
			quantizeBtn.SetSensitive(true)
		}
	})

	stepsChb.Connect("clicked", func() {
		logger.Debug("set steps to", "mode", stepsChb.GetActive())
		SinkLoop <- Message{Type: StepMode, Boolean: stepsChb.GetActive()}
	})

	reloadBtn.Connect("clicked", func() {
		cancel()
	})
	listIn, _ := gtk.ListStoreNew(glib.TYPE_INT, glib.TYPE_STRING)
	for i, port := range inL {
		iter := listIn.Append()
		listIn.Set(iter,
			[]int{0, 1},
			[]interface{}{inN[i], port},
		)
	}
	listOut, _ := gtk.ListStoreNew(glib.TYPE_INT, glib.TYPE_STRING)
	for i, port := range outL {
		iter := listOut.Append()
		listOut.Set(iter,
			[]int{0, 1},
			[]interface{}{outN[i], port},
		)
	}
	comboInPorts.SetModel(listIn)
	rendererIn, _ := gtk.CellRendererTextNew()
	comboInPorts.PackStart(rendererIn, true)
	//comboInPorts.AddAttribute(rendererIn, "number", 0)
	comboInPorts.AddAttribute(rendererIn, "text", 1)
	comboInPorts.SetActive(inP)

	comboOutPorts.SetModel(listOut)
	rendererOut, _ := gtk.CellRendererTextNew()
	comboOutPorts.PackStart(rendererOut, true)
	comboOutPorts.AddAttribute(rendererOut, "text", 1)
	comboOutPorts.SetActive(outP)

	reconnectMidi.Connect("clicked", func() {
		inIter, _ := comboInPorts.GetActiveIter()
		inVal, _ := listIn.GetValue(inIter, 0)
		inN, _ := inVal.GoValue()

		outIter, _ := comboOutPorts.GetActiveIter()
		outVal, _ := listOut.GetValue(outIter, 0)
		outN, _ := outVal.GoValue()
		MasterControl <- Message{
			Type:    RestartMIDI,
			Number:  inN.(int),
			Number2: outN.(int),
		}
	})

	stepReset.Connect("clicked", func() {
		SinkLoop <- Message{Type: ResetStep}
	})

	targetsList := []gtk.TargetEntry{
		targ(gtk.TargetEntryNew("text/plain", gtk.TARGET_OTHER_WIDGET, 0)),
		//targ(gtk.TargetEntryNew("audio/midi", gtk.TARGET_OTHER_APP, 0)),
		targ(gtk.TargetEntryNew("audio/midi", gtk.TARGET_OTHER_APP, 0)),
		targ(gtk.TargetEntryNew("text/plain", gtk.TARGET_OTHER_APP, 0)),
	}
	banksToggles := []*gtk.CheckButton{}
	banksLabels := map[int]*gtk.Label{}
	ACTION := gdk.ACTION_COPY
	for i := 0; i < NUM_BANKS; i++ {
		bankEventBox, _ := gtk.EventBoxNew() //gtk.LabelNew(fmt.Sprintf("bank \n%d", i))
		bankLabel, _ := gtk.LabelNew(fmt.Sprintf("bank %d", i))
		banksLabels[i] = bankLabel

		playBankToggle, _ := gtk.CheckButtonNewWithLabel("play")
		playBankToggle.Connect("toggled", func() {
			SinkLoop <- Message{Type: BankStateChange, Boolean: playBankToggle.GetActive(), Number: i}
		})

		banksToggles = append(banksToggles, playBankToggle)
		playBankToggle.SetEvents(int(gdk.BUTTON_PRESS_MASK))
		playBankToggle.Connect("button-release-event", func(self *gtk.CheckButton, event *gdk.Event) bool {
			switch gdk.EventButtonNewFromEvent(event).Button() {
			case gdk.BUTTON_SECONDARY:
				if self.GetActive() {
					for _, cb := range banksToggles {
						cb.SetActive(!cb.GetActive())
					}
				} else {
					for j, cb := range banksToggles {
						if j != i {
							cb.SetActive(false)
						}
					}
					banksToggles[i].SetActive(true)
				}
				return true
			case gdk.BUTTON_MIDDLE:
				for _, cb := range banksToggles {
					cb.SetActive(!cb.GetActive())
				}
				return true
			default:
				return false
			}
		})

		if i == 0 {
			bankLabel.SetLabel("buffer")
			playBankToggle.SetActive(true)
		}

		bankBox, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 5)
		bankBox.SetMarginStart(5)

		sc, _ := bankLabel.GetStyleContext()
		sc.AddClass("zone")

		bankBox.Add(bankLabel)
		bankBox.Add(playBankToggle)

		bankEventBox.Add(bankBox)

		banksBox.Add(bankEventBox)
		/*bankBtn.SetMarginBottom(10)
		bankBtn.SetMarginStart(10)
		bankBtn.SetMarginEnd(10)
		bankBtn.SetMarginTop(10)*/

		bankEventBox.DragSourceSet(gdk.BUTTON1_MASK|gdk.BUTTON2_MASK, targetsList, ACTION)
		bankEventBox.DragDestSet(gtk.DEST_DEFAULT_ALL, targetsList, ACTION)
		/*bankBtn.Connect("drag-begin", func(self *gtk.EventBox, context *gdk.DragContext) {
			println("drag-begin")
			//self.GetChild()
		})
		bankBtn.Connect("drag-end", func(self *gtk.EventBox, context *gdk.DragContext) {
			println("drag-end")
		})
		bankBtn.Connect("drag-drop", func(self *gtk.EventBox, context *gdk.DragContext, x, y int, time int) {
			fmt.Printf("drag-drop x=%d,y=%d %s %#v\n", x, y, _btn.GetLabel(), context)
		})*/
		bankEventBox.Connect("drag-data-get", func(self *gtk.EventBox, ctx *gdk.DragContext, data *gtk.SelectionData, info, time int) {
			data.SetData(gdk.SELECTION_PRIMARY, []byte{
				byte(Bank),
				byte(i),
			})
			//data.SetURIs([]string{"/tmp/test/source.txt"})
		})
		bankEventBox.Connect("drag-data-received", func(self *gtk.EventBox, ctx *gdk.DragContext, x, y int, data *gtk.SelectionData, m int, t uint) {
			dst := i
			dragType := DragDropSrc(data.GetData()[0])
			switch dragType {
			case Bank:
				src := int(data.GetData()[1])
				logger.Printf("append bank %d to bank %d\n", src, dst)

				SinkLoop <- Message{
					Type:    BankDragDrop,
					Number:  src,
					Number2: dst,
				}
			case ImportZone:
				filename := importBankBtn.GetFilename()
				logger.Info("appending to bank", "index", i, "path", filename)
				if len(filename) > 0 {
					SinkLoop <- Message{
						Type:   BankImport,
						Number: dst,
						String: filename,
					}
				}
			}

		})
		bankEventBox.Connect("drag-data-delete", func(self *gtk.EventBox, ctx *gdk.DragContext) {
			println("drag-fata-delete")
			println(bankLabel.GetLabel(), "ACK drag-fata-delete")
		})
	}

	deleteZone.DragDestSet(gtk.DEST_DEFAULT_ALL, targetsList, ACTION)
	deleteZone.Connect("drag-data-received", func(self *gtk.EventBox, ctx *gdk.DragContext, x, y int, data *gtk.SelectionData, m int, t uint) {
		dragType := DragDropSrc(data.GetData()[0])
		if dragType != Bank {
			return
		}
		src := int(data.GetData()[1])
		logger.Info("delete bank", "index", src)
		SinkLoop <- Message{
			Type:   BankClear,
			Number: src,
		}
	})

	exportZone.DragDestSet(gtk.DEST_DEFAULT_ALL, targetsList, ACTION)
	exportZone.Connect("drag-data-received", func(self *gtk.EventBox, ctx *gdk.DragContext, x, y int, data *gtk.SelectionData, m int, t uint) {
		dragType := DragDropSrc(data.GetData()[0])
		if dragType != Bank {
			return
		}
		src := int(data.GetData()[1])
		logger.Info("export bank", "src", src)
		d, _ := gtk.FileChooserDialogNewWith2Buttons("Exporter MIDI", mainWin, gtk.FILE_CHOOSER_ACTION_SAVE, "Exporter", gtk.RESPONSE_ACCEPT, "Annuler", gtk.RESPONSE_CANCEL)
		filter, _ := gtk.FileFilterNew()
		filter.AddPattern("*.mid")
		filter.AddPattern("*.midi")
		filter.AddMimeType("audio/midi")
		d.SetFilter(filter)
		d.SetDoOverwriteConfirmation(true)
		glib.IdleAdd(func() {
			response := d.Run()
			if response == gtk.RESPONSE_ACCEPT {
				//SinkLoop <- Message{ev: LoadFromFile, str: d.GetFilename()}
				logger.Info("exporting bank to file", "bank", src, "path", d.GetFilename())
				SinkLoop <- Message{
					Type:   BankExport,
					Number: src,
					String: d.GetFilename(),
				}
			}
			d.Destroy()
		})
	})

	importZone.DragSourceSet(gdk.BUTTON1_MASK, targetsList, ACTION)
	importZone.Connect("drag-data-get", func(self *gtk.EventBox, ctx *gdk.DragContext, data *gtk.SelectionData, info, time int) {
		logger.Debug("import drag")
		binData := []byte{byte(ImportZone)}
		data.SetData(gdk.SELECTION_PRIMARY, append(binData, []byte(importBankBtn.GetFilename())...))
	})

	cutZone.DragDestSet(gtk.DEST_DEFAULT_ALL, targetsList, ACTION)
	cutZone.Connect("drag-data-received", func(self *gtk.EventBox, ctx *gdk.DragContext, x, y int, data *gtk.SelectionData, m int, t uint) {
		dragType := DragDropSrc(data.GetData()[0])
		if dragType != Bank {
			logger.Info("bad drag destination: cutzone")
			return
		}
		src := int(data.GetData()[1])
		logger.Info("cutting bank to buffer", "bank", src)
		SinkLoop <- Message{Type: BankCut, Number: src}
	})

	loadStateBtn.Connect("clicked", func() {
		d, _ := gtk.FileChooserDialogNewWith2Buttons("Charger MIDI", mainWin, gtk.FILE_CHOOSER_ACTION_OPEN, "Ouvrir", gtk.RESPONSE_ACCEPT, "Annuler", gtk.RESPONSE_CANCEL)
		filter, _ := gtk.FileFilterNew()
		filter.AddPattern("*.mid")
		filter.AddPattern("*.midi")
		filter.AddMimeType("audio/midi")
		d.SetFilter(filter)
		d.SetKeepAbove(false)
		d.SetKeepBelow(true)
		response := d.Run()
		if response == gtk.RESPONSE_ACCEPT {
			SinkLoop <- Message{Type: StateImport, String: d.GetFilename()}
		}
		d.Destroy()
	})

	//saveFileBtn, _ := gtk.ButtonNewWithLabel("Sauvegarder vers fichier")
	saveStateBtn.Connect("clicked", func() {
		d, _ := gtk.FileChooserDialogNewWith2Buttons("Enregistrer MIDI", mainWin, gtk.FILE_CHOOSER_ACTION_SAVE, "Sauvegarder", gtk.RESPONSE_ACCEPT, "Annuler", gtk.RESPONSE_CANCEL)
		d.SetDoOverwriteConfirmation(true)
		response := d.Run()
		if response == gtk.RESPONSE_ACCEPT {
			SinkLoop <- Message{Type: StateExport, String: d.GetFilename()}
		}
		d.Destroy()
	})
	exportMultiTrackBtn.Connect("clicked", func() {
		d, _ := gtk.FileChooserDialogNewWith2Buttons("Exporter MIDI", mainWin, gtk.FILE_CHOOSER_ACTION_SAVE, "Exporter", gtk.RESPONSE_ACCEPT, "Annuler", gtk.RESPONSE_CANCEL)
		d.SetDoOverwriteConfirmation(true)
		response := d.Run()
		if response == gtk.RESPONSE_ACCEPT {
			SinkLoop <- Message{Type: ExportMultiTrack, String: d.GetFilename()}
		}
		d.Destroy()
	})

	eraseSessionBtn.Connect("clicked", func() {
		SinkLoop <- Message{Type: ClearState}
	})

	undoNote.Connect("clicked", func() {
		SinkLoop <- Message{Type: NoteUndo}
	})

	/*loadFileBtn2, _ := gtk.FileChooserButtonNew("ouvrir", gtk.FILE_CHOOSER_ACTION_OPEN) //comme en HTML
	loadFileBtn2.Connect("file-set", func() {
		println(loadFileBtn2.GetFilename())
	})
	loadFileBtn2.DragDestSet(gtk.DEST_DEFAULT_ALL, targetsList, gdk.ACTION_COPY)
	mainBox.Add(loadFileBtn2)*/

	prov, _ := gtk.CssProviderNew()
	if err := prov.LoadFromData(stylesheet); err != nil {
		logger.Warn(err)
	}
	screen, _ := gdk.ScreenGetDefault()
	gtk.AddProviderForScreen(screen, prov, 1)
	mainWin.ShowAll()

	go loop(ctx, SinkUI, *logger, banksLabels)
	gtk.Main()
	logger.Info("stop")
	mainWin.HandlerDisconnect(windestroyhandle)
	mainWin.Destroy()

}

func targ(t *gtk.TargetEntry, err error) gtk.TargetEntry {
	return *t
}
