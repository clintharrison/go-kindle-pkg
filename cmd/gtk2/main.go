package main

/*
#cgo pkg-config: gtk+-2.0
#include <gtk/gtk.h>

void set_ythickness(GtkStyle *style, int ythickness) {
    style->ythickness = ythickness;
}
*/
import "C"

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"

	"github.com/clintharrison/go-kindle-pkg/pkg/repository"
	"github.com/mattn/go-gtk/glib"
	"github.com/mattn/go-gtk/gtk"
	"github.com/pingcap/errors"
)

type AppState struct {
	repository *repository.Repository

	listStore *gtk.ListStore
}

func do() error {
	s := AppState{} //nolint:exhaustruct
	f, err := os.Open("/tmp/repo.json")
	if err != nil {
		return errors.Wrapf(err, "os.Open(%q)", "/tmp/repo.json")
	}
	defer f.Close()
	data, err := io.ReadAll(f)
	if err != nil {
		return errors.Wrapf(err, "io.ReadAll(%q)", "/tmp/repo.json")
	}
	err = json.Unmarshal(data, &s.repository)
	if err != nil {
		return errors.Wrapf(err, "json.Unmarshal()")
	}

	gtk.Init(&os.Args)
	window := gtk.NewWindow(gtk.WINDOW_TOPLEVEL)
	window.SetTitle("L:A_N:application_ID:org.kindlemodding.example-gtk-application_PC:T")
	window.Connect("destroy", func(_ *glib.CallbackContext) {
		fmt.Println("goodbye!")
		gtk.MainQuit()
	})

	vbox := gtk.NewVBox(false, 1)
	window.Add(vbox)

	// swin := gtk.NewScrolledWindow(nil, nil)
	// vbox.Add(swin)

	listStore := gtk.NewListStore(gtk.TYPE_STRING)
	s.listStore = listStore

	listView := gtk.NewTreeView()
	vbox.Add(listView)

	listView.SetModel(s.listStore)
	listView.AppendColumn(gtk.NewTreeViewColumnWithAttributes("Package name", gtk.NewCellRendererText(), "text", 0))
	for _, pkg := range []string{"Package 1", "Package 2", "Package 3"} {
		var iter gtk.TreeIter
		s.listStore.Append(&iter)
		s.listStore.SetValue(&iter, 0, pkg)
	}

	vboxButtons := gtk.NewVBox(true, 0)
	quitButton := gtk.NewButtonWithLabel("Quit")
	quitButton.Connect("clicked", func(_ *glib.CallbackContext) {
		gtk.MainQuit()
	})
	vboxButtons.Add(quitButton)
	previewButton := gtk.NewButtonWithLabel("Preview")
	previewButton.Connect("clicked", func(_ *glib.CallbackContext) {
		slog.Info("preview clicked")
	})
	vboxButtons.Add(previewButton)
	installButton := gtk.NewButtonWithLabel("Install")
	installButton.Connect("clicked", func(_ *glib.CallbackContext) {
		slog.Info("install clicked")
	})
	vboxButtons.Add(installButton)
	padding := uint(5) //nolint:mnd
	vboxButtons.SetChildPacking(quitButton, true, true, padding, gtk.PACK_START)
	vboxButtons.SetChildPacking(previewButton, true, true, padding, gtk.PACK_START)
	vboxButtons.SetChildPacking(installButton, true, true, padding, gtk.PACK_START)
	vbox.Add(vboxButtons)

	vbox.SetChildPacking(listView, true, true, 0, gtk.PACK_START)
	vbox.SetChildPacking(vboxButtons, false, true, 0, gtk.PACK_END)

	window.ShowAll()

	gtk.Main()

	return nil
}

func main() {
	err := do()
	if err != nil {
		// fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}
