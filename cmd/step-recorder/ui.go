package main

import (
	"fmt"
	"log"
	"time"

	"github.com/gotk3/gotk3/glib"
	"github.com/gotk3/gotk3/gtk"
)

func chooseCallbak(i int, win *gtk.Window, name string, action gtk.FileChooserAction, result func(string, int)) func(*gtk.Button) {
	return func(v *gtk.Button) {
		d, err := gtk.FileChooserDialogNewWith2Buttons(fmt.Sprintf("charge %d", i), win, action, name, gtk.RESPONSE_ACCEPT, "Annuler", gtk.RESPONSE_CANCEL)
		he(err)
		response := d.Run()
		if response == gtk.RESPONSE_ACCEPT {
			println(i, d.GetFilename())
			result(d.GetFilename(), i)
		}
		d.Destroy()
	}
}

func ui(banks *[]*Recording) {
	gtk.Init(nil)

	// Create a new toplevel window, set its title, and connect it to the
	// "destroy" signal to exit the GTK main loop when it is destroyed.
	win, err := gtk.WindowNew(gtk.WINDOW_TOPLEVEL)
	if err != nil {
		log.Fatal("Unable to create window:", err)
	}
	win.SetTitle("Step-Recorder")
	win.Connect("destroy", func() {
		gtk.MainQuit()
	})
	banks_indicators := []*gtk.Label{}

	main_box, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 10)

	mv, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 5)
	for i, _ := range *banks {
		hv, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 10)
		l, _ := gtk.LabelNew(fmt.Sprintf("bank %d", i))
		hv.Add(l)
		fcb, _ := gtk.ButtonNewWithLabel("charger")
		fcb.Connect("clicked", chooseCallbak(i, win, "Charger", gtk.FILE_CHOOSER_ACTION_OPEN, func(s string, j int) {
			fmt.Printf("load %s %d\n", s, j)
			LoadFile(s, (*banks)[j])
			println(len(*(*banks)[j]))
		}))
		hv.Add(fcb)

		fsb, _ := gtk.ButtonNewWithLabel("enregistrer")
		fsb.Connect("clicked", chooseCallbak(i, win, "Enregistrer", gtk.FILE_CHOOSER_ACTION_SAVE, func(s string, j int) {
			fmt.Printf("save %s %d\n", s, j)
			Save((*banks)[j], s)
		}))
		hv.Add(fsb)

		bi, _ := gtk.LabelNew("len: 0")
		hv.Add(bi)
		banks_indicators = append(banks_indicators, bi)

		mv.Add(hv)
	}

	go func() {
		for {
			time.Sleep(2 * time.Second)
			glib.IdleAdd(func() {
				for i, bi := range banks_indicators {
					bank := (*banks)[i]
					bi.SetLabel(fmt.Sprintf("len: %d", len(*bank)/2))
				}
			})
		}
	}()

	main_box.Add(mv)

	armureLabel, _ := gtk.LabelNew("armure: pas changé")
	armureLabel.Connect("clicked", func(self *gtk.Label) {
		armureLabel.SetText("armure:" + armure_shift.String())
	})

	scale, err := gtk.ScaleNewWithRange(gtk.ORIENTATION_HORIZONTAL, 0, 26, 1)
	scale.Connect("change-value", func(self *gtk.Scale, scrolltype gtk.ScrollType, value float64) {
		armure := Armure(value)
		if armure >= DO_MAJEUR && armure <= MI_MINEUR {
			if armure != armure_shift {
				fmt.Printf("ancien: %d, nouveau: %d, s: %s → %s\n",
					armure, armure_shift, armure.String(), armure_shift.String())
			}
			armure_shift = armure
			armureLabel.SetText("armure:" + armure_shift.String())
		}
	})
	he(err)
	main_box.Add(scale)
	main_box.Add(armureLabel)

	win.Add(main_box)

	// Set the default window size.
	win.SetDefaultSize(800, 600)

	// Recursively show all widgets contained in this window.
	win.ShowAll()

	// Begin executing the GTK main loop.  This blocks until
	// gtk.MainQuit() is run.
	gtk.Main()

}
