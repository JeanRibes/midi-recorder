package ui

import (
	"strings"
	"time"

	"github.com/charmbracelet/log"
	"github.com/gotk3/gotk3/glib"
	"github.com/gotk3/gotk3/gtk"
)

type Tableau struct {
	treeView  *gtk.TreeView
	listStore *gtk.ListStore
}

const (
	COLONNE_NOM = iota
	COLONNE_DATE
)

func NewWithTreeView(treeView *gtk.TreeView) *Tableau {
	cell1Renderer, _ := gtk.CellRendererTextNew()
	column1, _ := gtk.TreeViewColumnNewWithAttribute("nom", cell1Renderer, "text", COLONNE_NOM)
	treeView.AppendColumn(column1)

	cell2Renderer, _ := gtk.CellRendererTextNew()
	column2, _ := gtk.TreeViewColumnNewWithAttribute("date", cell2Renderer, "text", COLONNE_DATE)
	treeView.AppendColumn(column2)

	listStore, err := gtk.ListStoreNew(glib.TYPE_STRING, glib.TYPE_STRING)
	if err != nil {
		log.Fatal("Unable to create list store:", err)
	}
	treeView.SetModel(listStore)

	return &Tableau{
		treeView:  treeView,
		listStore: listStore,
	}
}

func (tb *Tableau) AddRow(nom string, date time.Time) {
	iter := tb.listStore.Append()

	depuis := date.Format("le 02/01 à 15h04")
	// Set the contents of the list store row that the iterator represents
	tb.listStore.SetValue(iter, COLONNE_NOM, nom)
	/*tb.listStore.Set(iter,
	[]int{COLONNE_NOM, COLONNE_DATE},
	[]interface{}{nom, depuis})*/
	tb.listStore.SetValue(iter, COLONNE_DATE, depuis)
}

func (tb *Tableau) Clear() {
	tb.listStore.Clear()
}

func (tb *Tableau) FromSessions(sessions []string) {
	home := glib.GetHomeDir()
	print("home", home)
	tb.Clear()
	for _, sess := range sessions {
		sess = strings.Replace(sess, home, "~", 1)
		tb.AddRow(sess, time.Now())
	}
}
