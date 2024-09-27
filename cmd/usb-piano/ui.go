package main

import (
	"fmt"
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
		BusFromUItoLoop <- Message{ev: RecordStart}
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

	mainBox.Add(recordBtn)
	mainBox.Add(quantizeBox)
	mainBox.Add(playBtn)
	mainBox.Add(stepBtn)

	win.Add(mainBox)
	win.SetDefaultSize(800, 300)
	win.ShowAll()

	go func() {
		for {
			msg := <-BusFromLoopToUI
			switch msg.ev {
			case RecordStart:
				glib.IdleAdd(func() {
					recordBtn.SetLabel(fmt.Sprintf("Recording, press %d to end", msg.number))
				})
			case RecordStop:
				glib.IdleAdd(func() {
					recordBtn.SetSensitive(true)
					recordBtn.SetLabel("Record")
				})
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
			}
		}
	}()
	gtk.Main()

}
