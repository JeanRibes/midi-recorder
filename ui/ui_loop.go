package ui

import (
	"context"
	"fmt"

	. "github.com/JeanRibes/midi-recorder/shared"
	"github.com/charmbracelet/log"
	"github.com/gotk3/gotk3/glib"
	"github.com/gotk3/gotk3/gtk"
)

func loop(ctx context.Context, SinkUI chan Message, logger log.Logger, banksLabels map[int]*gtk.Label) {
	errors := ""
	errorDialog := gtk.MessageDialogNew(mainWin, gtk.DIALOG_MODAL, gtk.MESSAGE_ERROR, gtk.BUTTONS_CLOSE, "Erreur")
	errorDialog.Connect("response", func() {
		errorDialog.Hide()
		errors = ""
	})

	for {
		select {
		case <-ctx.Done():
			logger.Debug("chan Done, quitting")
			gtk.MainQuit()
			return
		case msg := <-SinkUI:
			switch msg.Type {
			case Record:
				if msg.Boolean {
					glib.IdleAdd(func() {
						recordBtn.SetLabel("Arrêter rec")
						recordBtn.SetSensitive(true)
						recordBtn.SetImage(stoprecordImg)
					})
				} else {
					glib.IdleAdd(func() {
						recordBtn.SetSensitive(true)
						recordBtn.SetLabel("Enregistrer")
						recordBtn.SetImage(startrecordImg)
					})
				}
			case PlayPause:
				if msg.Boolean {
					glib.IdleAdd(func() {
						playBtn.SetImage(pauseImg)
					})
				} else {
					glib.IdleAdd(func() {
						playBtn.SetImage(playImg)
					})
				}
			case Quantize:
				glib.IdleAdd(func() { quantizeBtn.SetSensitive(true) })
			case StepMode:
				stepsChb.SetActive(msg.Boolean)
				logger.Debug("set stepsChb to", "bool", msg.Boolean)
			case Error:
				if len(errors) == 0 {
					errors = msg.String
				} else {
					errors += "\n\n" + msg.String
				}
				glib.IdleAdd(func() {
					errorDialog.FormatSecondaryText(errors)
					errorDialog.Show()
				})
			case BankLengthNotify:
				bank := msg.Number
				length := msg.Number2
				glib.IdleAdd(func() {
					banksLabels[bank].SetLabel(BankName(bank) + fmt.Sprintf("\n%d notes", length))
					/*if bank == 0 {
						banksLabels[bank].SetLabel(fmt.Sprintf("buffer \n%d notes", length))
					} else {
						banksLabels[bank].SetLabel(fmt.Sprintf("bank %d\n%d notes", bank, length))
					}*/
				})
			}
		}
	}
}
