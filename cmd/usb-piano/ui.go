package main

import (
	"log"
	"strconv"

	"github.com/gotk3/gotk3/glib"
	"github.com/gotk3/gotk3/gtk"
)

func ui() {
	gtk.Init(nil)
	win, err := gtk.WindowNew(gtk.WINDOW_TOPLEVEL)
	if err != nil {
		log.Fatal("Unable to create window:", err)
	}
	win.SetTitle("Step-Recorder")
	win.Connect("destroy", func() {
		BusFromUItoLoop <- Message{ev: Quit}
		gtk.MainQuit()
	})

	mainBox, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 1)

	recordBtn, _ := gtk.ButtonNew()
	recordBtn.SetLabel("Record")
	recordBtn.Connect("clicked", func() {
		recordBtn.SetSensitive(false)
		BusFromUItoLoop <- Message{ev: Record}
	})

	playBtn, _ := gtk.ButtonNewWithLabel("Play")
	playBtn.Connect("clicked", func() {
		playBtn.SetSensitive(false)
		BusFromUItoLoop <- Message{ev: PlayPause}
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
				BusFromUItoLoop <- Message{ev: Quantize, number: int(i)}
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
		BusFromUItoLoop <- Message{ev: StepMode}
	})

	loadFileBtn, _ := gtk.ButtonNewWithLabel("Charger depuis fichier")
	loadFileBtn.Connect("clicked", func() {
		d, err := gtk.FileChooserDialogNewWith2Buttons("Charger MIDI", win, gtk.FILE_CHOOSER_ACTION_OPEN, "Ouvrir", gtk.RESPONSE_ACCEPT, "Annuler", gtk.RESPONSE_CANCEL)
		he(err)
		response := d.Run()
		if response == gtk.RESPONSE_ACCEPT {
			BusFromUItoLoop <- Message{ev: LoadFromFile, str: d.GetFilename()}
		}
		d.Destroy()
	})

	saveFileBtn, _ := gtk.ButtonNewWithLabel("Sauvegarder vers fichier")
	saveFileBtn.Connect("clicked", func() {
		d, err := gtk.FileChooserDialogNewWith2Buttons("Enregistrer MIDI", win, gtk.FILE_CHOOSER_ACTION_SAVE, "Sauvegarder", gtk.RESPONSE_ACCEPT, "Annuler", gtk.RESPONSE_CANCEL)
		he(err)
		response := d.Run()
		if response == gtk.RESPONSE_ACCEPT {
			BusFromUItoLoop <- Message{ev: SaveToFile, str: d.GetFilename()}
		}
		d.Destroy()
	})

	mainBox.Add(recordBtn)
	mainBox.Add(quantizeBox)
	mainBox.Add(playBtn)
	mainBox.Add(stepBtn)
	mainBox.Add(loadFileBtn)
	mainBox.Add(saveFileBtn)

	win.Add(mainBox)
	win.SetDefaultSize(800, 300)
	win.ShowAll()

	go func() {
		for {
			msg := <-BusFromLoopToUI
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
				glib.IdleAdd(func() {
					d := gtk.MessageDialogNew(win, gtk.DIALOG_MODAL, gtk.MESSAGE_ERROR, gtk.BUTTONS_CLOSE, "erreur :"+msg.str)
					d.Connect("response", d.Destroy)
					d.Run()
				})

			}
		}
	}()
	gtk.Main()

}
