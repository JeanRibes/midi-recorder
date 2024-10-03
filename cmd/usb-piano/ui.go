package main

import (
	"context"
	"fmt"
	"os"
	"strconv"

	"github.com/charmbracelet/log"
	charmlog "github.com/charmbracelet/log"

	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/glib"
	"github.com/gotk3/gotk3/gtk"
)

func ui(ctx context.Context, cancel func(), inP, outP int, inL []string, inN []int, outL []string, outN []int) {
	logger := charmlog.NewWithOptions(os.Stdout, charmlog.Options{
		Level: charmlog.DebugLevel,
		//ReportCaller:    true,
		ReportTimestamp: false,
		Prefix:          "UI",
	})
	logger.Info("start")
	gtk.Init(nil)

	builder, err := gtk.BuilderNewFromFile("ui.glade") //remplacer par du embed
	he(err)

	_mainWin, _ := builder.GetObject("mainWin")
	mainWin := _mainWin.(*gtk.Window)
	_reconnectMidi, _ := builder.GetObject("reconnectMidi")
	reconnectMidi := _reconnectMidi.(*gtk.Button)
	_reloadBtn, _ := builder.GetObject("reloadBtn")
	reloadBtn := _reloadBtn.(*gtk.Button)
	_saveState, _ := builder.GetObject("saveState")
	saveState := _saveState.(*gtk.Button)
	_comboInPorts, _ := builder.GetObject("comboInPorts")
	comboInPorts := _comboInPorts.(*gtk.ComboBox)
	_comboOutPorts, _ := builder.GetObject("comboOutPorts")
	comboOutPorts := _comboOutPorts.(*gtk.ComboBox)
	_bpmEntry, _ := builder.GetObject("bpmEntry")
	bpmEntry := _bpmEntry.(*gtk.Entry)
	_ticksEntry, _ := builder.GetObject("ticksEntry")
	ticksEntry := _ticksEntry.(*gtk.Entry)
	_quantizeBtn, _ := builder.GetObject("quantizeBtn")
	quantizeBtn := _quantizeBtn.(*gtk.Button)
	_recordBtn, _ := builder.GetObject("recordBtn")
	recordBtn := _recordBtn.(*gtk.Button)
	_playBtn, _ := builder.GetObject("playBtn")
	playBtn := _playBtn.(*gtk.Button)
	_stepsChb, _ := builder.GetObject("stepsChb")
	stepsChb := _stepsChb.(*gtk.CheckButton)
	_stepReset, _ := builder.GetObject("stepReset")
	stepReset := _stepReset.(*gtk.Button)
	_banksBox, _ := builder.GetObject("banksBox")
	banksBox := _banksBox.(*gtk.Box)
	_deleteZone, _ := builder.GetObject("deleteZone")
	deleteZone := _deleteZone.(*gtk.EventBox)

	_play, _ := builder.GetObject("play")
	play := _play.(*gtk.Image)
	_pause, _ := builder.GetObject("pause")
	pause := _pause.(*gtk.Image)

	if err != nil {
		logger.Error("Unable to create window:", err)
	}
	windestroyhandle := mainWin.Connect("destroy", func() {
		//cancel()
		logger.Debug("close win, sending quit event")
		MasterControl <- Message{ev: Quit}
	})

	recordBtn.Connect("clicked", func() {
		recordBtn.SetSensitive(false)
		SinkLoop <- Message{ev: Record}
	})

	playBtn.Connect("clicked", func() {
		SinkLoop <- Message{ev: PlayPause}
	})

	quantizeBtn.Connect("clicked", func() {
		quantizeBtn.SetSensitive(false)
		txt, err := bpmEntry.GetText()

		ticksEntry.GetText()
		if err == nil {
			var i int64
			if i, err = strconv.ParseInt(txt, 10, 32); err == nil {
				SinkLoop <- Message{ev: Quantize, number: int(i)}
			} else {
				log.Error(err)
				quantizeBtn.SetSensitive(true)
			}
		} else {
			log.Error(err)
			quantizeBtn.SetSensitive(true)
		}
	})

	stepsChb.Connect("clicked", func() {
		SinkLoop <- Message{ev: StepMode}
	})

	loadFileBtn, _ := gtk.ButtonNewWithLabel("Charger depuis fichier")
	loadFileBtn.Connect("clicked", func() {
		d, err := gtk.FileChooserDialogNewWith2Buttons("Charger MIDI", mainWin, gtk.FILE_CHOOSER_ACTION_OPEN, "Ouvrir", gtk.RESPONSE_ACCEPT, "Annuler", gtk.RESPONSE_CANCEL)
		he(err)
		filter, _ := gtk.FileFilterNew()
		filter.AddPattern("*.mid")
		filter.AddPattern("*.midi")
		filter.AddMimeType("audio/midi")
		d.SetFilter(filter)
		d.SetKeepAbove(false)
		d.SetKeepBelow(true)
		response := d.Run()
		if response == gtk.RESPONSE_ACCEPT {
			SinkLoop <- Message{ev: LoadFromFile, str: d.GetFilename()}
		}
		d.Destroy()
	})

	//saveFileBtn, _ := gtk.ButtonNewWithLabel("Sauvegarder vers fichier")
	saveState.Connect("clicked", func() {
		d, err := gtk.FileChooserDialogNewWith2Buttons("Enregistrer MIDI", mainWin, gtk.FILE_CHOOSER_ACTION_SAVE, "Sauvegarder", gtk.RESPONSE_ACCEPT, "Annuler", gtk.RESPONSE_CANCEL)
		he(err)
		d.SetDoOverwriteConfirmation(true)
		response := d.Run()
		println(response)
		if response == gtk.RESPONSE_ACCEPT {
			SinkLoop <- Message{ev: SaveToFile, str: d.GetFilename()}
		}
		d.Destroy()
	})

	errors := ""
	errorDialog := gtk.MessageDialogNew(mainWin, gtk.DIALOG_MODAL, gtk.MESSAGE_ERROR, gtk.BUTTONS_CLOSE, "Erreur")
	errorDialog.Connect("response", func() {
		errorDialog.Hide()
		errors = ""
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
		inIter, err := comboInPorts.GetActiveIter()
		he(err)
		inVal, err := listIn.GetValue(inIter, 0)
		he(err)
		inN, err := inVal.GoValue()
		he(err)

		outIter, _ := comboOutPorts.GetActiveIter()
		outVal, _ := listOut.GetValue(outIter, 0)
		outN, _ := outVal.GoValue()
		MasterControl <- Message{
			ev:     RestartMIDI,
			number: inN.(int),
			port2:  outN.(int),
		}
	})

	stepReset.Connect("clicked", func() {
		SinkLoop <- Message{ev: ResetStep}
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
			SinkLoop <- Message{ev: BankStateChange, boolean: playBankToggle.GetActive(), number: i}
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
		bankBox.Add(bankLabel)
		bankBox.Add(playBankToggle)
		frame, _ := gtk.FrameNew("")
		frame.SetBorderWidth(1)
		bankBox.SetBorderWidth(5)

		frame.Add(bankBox)
		bankEventBox.Add(frame)

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
			data.SetData(gdk.SELECTION_PRIMARY, []byte{byte(i)})
			//data.SetURIs([]string{"/tmp/test/source.txt"})
		})
		bankEventBox.Connect("drag-data-received", func(self *gtk.EventBox, ctx *gdk.DragContext, x, y int, data *gtk.SelectionData, m int, t uint) {
			dst := i
			src := int(data.GetData()[0])
			logger.Printf("append bank %d to bank %d\n", src, dst)

			SinkLoop <- Message{
				ev:     BankDragDrop,
				number: src,
				port2:  dst,
			}
		})
		bankEventBox.Connect("drag-data-delete", func(self *gtk.EventBox, ctx *gdk.DragContext) {
			println("drag-fata-delete")
			println(bankLabel.GetLabel(), "ACK drag-fata-delete")
		})

	}

	deleteZone.DragDestSet(gtk.DEST_DEFAULT_ALL, targetsList, ACTION)
	deleteZone.Connect("drag-data-received", func(self *gtk.EventBox, ctx *gdk.DragContext, x, y int, data *gtk.SelectionData, m int, t uint) {
		src := int(data.GetData()[0])
		logger.Info("delete bank", "index", src)
		SinkLoop <- Message{
			ev:     BankClear,
			number: src,
		}
	})

	/*loadFileBtn2, _ := gtk.FileChooserButtonNew("ouvrir", gtk.FILE_CHOOSER_ACTION_OPEN) //comme en HTML
	loadFileBtn2.Connect("file-set", func() {
		println(loadFileBtn2.GetFilename())
	})
	loadFileBtn2.DragDestSet(gtk.DEST_DEFAULT_ALL, targetsList, gdk.ACTION_COPY)
	mainBox.Add(loadFileBtn2)*/

	mainWin.ShowAll()

	go func() {
		for {
			select {
			case <-ctx.Done():
				logger.Debug("chan Done, quitting")
				gtk.MainQuit()
				return
			case msg := <-SinkUI:
				switch msg.ev {
				case Record:
					if msg.boolean {
						glib.IdleAdd(func() {
							recordBtn.SetLabel("Stop Recording (or press sustain)")
							recordBtn.SetSensitive(true)
						})
					} else {
						glib.IdleAdd(func() {
							recordBtn.SetSensitive(true)
							recordBtn.SetLabel("Record")
						})
					}
				case PlayPause:
					if msg.boolean {
						glib.IdleAdd(func() {
							playBtn.SetImage(pause)
						})
					} else {
						glib.IdleAdd(func() {
							playBtn.SetImage(play)
						})
					}
				case Quantize:
					glib.IdleAdd(func() { quantizeBtn.SetSensitive(true) })
				case StepMode:
					// checkbox
					continue
				case Error:
					if len(errors) == 0 {
						errors = msg.str
					} else {
						errors += "\n\n" + msg.str
					}
					glib.IdleAdd(func() {
						errorDialog.FormatSecondaryText(errors)
						errorDialog.Show()
					})
				case BankLengthNotify:
					bank := msg.number
					length := msg.port2
					glib.IdleAdd(func() {
						if bank == 0 {
							banksLabels[bank].SetLabel(fmt.Sprintf("buffer \n%d notes", length))
						} else {
							banksLabels[bank].SetLabel(fmt.Sprintf("bank %d\n%d notes", bank, length))
						}
					})
				}
			}
		}
	}()
	gtk.Main()
	logger.Info("stop")
	mainWin.HandlerDisconnect(windestroyhandle)
	mainWin.Destroy()

}
