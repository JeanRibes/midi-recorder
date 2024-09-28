package main

import (
	"context"
	"log"
	"strconv"

	"github.com/gotk3/gotk3/glib"
	"github.com/gotk3/gotk3/gtk"
)

func ui(ctx context.Context, cancel func(), inP, outP int, inL []string, inN []int, outL []string, outN []int) {
	gtk.Init(nil)
	win, err := gtk.WindowNew(gtk.WINDOW_TOPLEVEL)
	if err != nil {
		log.Fatal("Unable to create window:", err)
	}
	win.SetTitle("Step-Recorder")
	windestroyhandle := win.Connect("destroy", func() {
		//cancel()
		log.Println("ui: close win, sending quit event")
		MasterControl <- Message{ev: Quit}
	})

	mainBox, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 1)

	recordBtn, _ := gtk.ButtonNew()
	recordBtn.SetLabel("Record")
	recordBtn.Connect("clicked", func() {
		recordBtn.SetSensitive(false)
		SinkUI <- Message{ev: Record}
	})

	playBtn, _ := gtk.ButtonNewWithLabel("Play")
	playBtn.Connect("clicked", func() {
		playBtn.SetSensitive(false)
		SinkUI <- Message{ev: PlayPause}
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
				SinkUI <- Message{ev: Quantize, number: int(i)}
			} else {
				println(err.Error())
				quantizeBtn.SetSensitive(true)
			}
		} else {
			println(err.Error())
			quantizeBtn.SetSensitive(true)
		}
	})
	quantizeBox.Add(quantizeInput)
	quantizeBox.Add(quantizeBtn)

	stepBtn, _ := gtk.ButtonNewWithLabel("Activer Mode steps")
	stepBtn.Connect("clicked", func() {
		SinkUI <- Message{ev: StepMode}
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
			SinkUI <- Message{ev: LoadFromFile, str: d.GetFilename()}
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
			SinkUI <- Message{ev: SaveToFile, str: d.GetFilename()}
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
		println(inN[i], port)
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

	mainBox.Add(reloadBtn)

	mainBox.Add(comboInPorts)
	mainBox.Add(comboOutPorts)
	mainBox.Add(changePortsBtn)

	mainBox.Add(recordBtn)
	mainBox.Add(quantizeBox)
	mainBox.Add(playBtn)
	mainBox.Add(stepBtn)
	mainBox.Add(loadFileBtn)
	mainBox.Add(saveFileBtn)

	win.Add(mainBox)
	win.ShowAll()

	go func() {
		for {
			select {
			case <-ctx.Done():
				log.Println("ui: chan Done, quitting")
				gtk.MainQuit()
				return
			case msg := <-SinkLoop:
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
					glib.IdleAdd(func() { playBtn.SetSensitive(true) })
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

				}
			}
		}
	}()
	gtk.Main()
	log.Println("ui: exited")
	win.HandlerDisconnect(windestroyhandle)
	win.Destroy()

}
