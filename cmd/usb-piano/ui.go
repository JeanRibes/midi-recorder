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
	win, err := gtk.WindowNew(gtk.WINDOW_TOPLEVEL)
	if err != nil {
		logger.Error("Unable to create window:", err)
	}
	win.SetTitle("Piano Jean")
	windestroyhandle := win.Connect("destroy", func() {
		//cancel()
		logger.Debug("close win, sending quit event")
		MasterControl <- Message{ev: Quit}
	})

	mainBox, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 1)

	recordBtn, _ := gtk.ButtonNew()
	recordBtn.SetLabel("Record")
	recordBtn.Connect("clicked", func() {
		recordBtn.SetSensitive(false)
		SinkLoop <- Message{ev: Record}
	})

	playBtn, _ := gtk.ButtonNewWithLabel("Play")
	playBtn.Connect("clicked", func() {
		SinkLoop <- Message{ev: PlayPause}
	})

	quantizeBox, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 10)

	quantizeInput, _ := gtk.EntryNew()
	quantizeInput.SetText("120")
	quantizeBtn, _ := gtk.ButtonNewWithLabel("Quantize")
	quantizeBtn.Connect("clicked", func() {
		quantizeBtn.SetSensitive(false)
		txt, err := quantizeInput.GetText()
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
	quantizeBox.Add(quantizeInput)
	quantizeBox.Add(quantizeBtn)

	stepBtn, _ := gtk.ButtonNewWithLabel("Activer Mode steps")
	stepBtn.Connect("clicked", func() {
		SinkLoop <- Message{ev: StepMode}
	})

	loadFileBtn, _ := gtk.ButtonNewWithLabel("Charger depuis fichier")
	loadFileBtn.Connect("clicked", func() {
		d, err := gtk.FileChooserDialogNewWith2Buttons("Charger MIDI", win, gtk.FILE_CHOOSER_ACTION_OPEN, "Ouvrir", gtk.RESPONSE_ACCEPT, "Annuler", gtk.RESPONSE_CANCEL)
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

	saveFileBtn, _ := gtk.ButtonNewWithLabel("Sauvegarder vers fichier")
	saveFileBtn.Connect("clicked", func() {
		d, err := gtk.FileChooserDialogNewWith2Buttons("Enregistrer MIDI", win, gtk.FILE_CHOOSER_ACTION_SAVE, "Sauvegarder", gtk.RESPONSE_ACCEPT, "Annuler", gtk.RESPONSE_CANCEL)
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
	errorDialog := gtk.MessageDialogNew(win, gtk.DIALOG_MODAL, gtk.MESSAGE_ERROR, gtk.BUTTONS_CLOSE, "Erreur")
	errorDialog.Connect("response", func() {
		errorDialog.Hide()
		errors = ""
	})

	reloadBtn, _ := gtk.ButtonNewWithLabel("Reload UI")
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
	comboInPorts, _ := gtk.ComboBoxNewWithModel(listIn)
	rendererIn, _ := gtk.CellRendererTextNew()
	comboInPorts.PackStart(rendererIn, true)
	//comboInPorts.AddAttribute(rendererIn, "number", 0)
	comboInPorts.AddAttribute(rendererIn, "text", 1)
	comboInPorts.SetActive(inP)

	comboOutPorts, _ := gtk.ComboBoxNewWithModel(listOut)
	rendererOut, _ := gtk.CellRendererTextNew()
	comboOutPorts.PackStart(rendererOut, true)
	comboOutPorts.AddAttribute(rendererOut, "text", 1)
	comboOutPorts.SetActive(outP)

	changePortsBtn, _ := gtk.ButtonNewWithLabel("Reconnect MIDI")
	changePortsBtn.Connect("clicked", func() {
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

	banksBox, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 5)
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

	deleteZone, _ := gtk.EventBoxNew()
	deleteImg, err := gtk.ImageNewFromIconName("edit-delete-symbolic", gtk.ICON_SIZE_DIALOG)
	if err != nil {
		he(err)
	}
	deleteZone.Add(deleteImg)

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

	mainBox.Add(reloadBtn)

	mainBox.Add(comboInPorts)
	mainBox.Add(comboOutPorts)
	mainBox.Add(changePortsBtn)

	mainBox.Add(recordBtn)
	mainBox.Add(quantizeBox)
	mainBox.Add(playBtn)
	mainBox.Add(stepBtn)
	mainBox.Add(banksBox)
	mainBox.Add(deleteZone)
	mainBox.Add(loadFileBtn)
	mainBox.Add(saveFileBtn)

	win.Add(mainBox)
	win.ShowAll()

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
						glib.IdleAdd(func() { playBtn.SetLabel("Stop") })
					} else {
						glib.IdleAdd(func() { playBtn.SetLabel("Play") })
					}
				case Quantize:
					glib.IdleAdd(func() { quantizeBtn.SetSensitive(true) })
				case StepMode:
					glib.IdleAdd(func() {
						if msg.boolean {
							stepBtn.SetLabel("Désactiver mode steps")
						} else {
							stepBtn.SetLabel("Activer mode steps")
						}
					})
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
	win.HandlerDisconnect(windestroyhandle)
	win.Destroy()

}
