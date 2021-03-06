// gui.go - simple GTK3 GUI for onionize.
//
// To the extent possible under law, Ivan Markin waived all copyright
// and related or neighboring rights to this module of onionize, using the creative
// commons "cc0" public domain dedication. See LICENSE or
// <http://creativecommons.org/publicdomain/zero/1.0/> for full details.
// +build gui

package main

import (
	"log"
	"net/url"
	"os"

	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/glib"
	"github.com/gotk3/gotk3/gtk"
	"github.com/nogoegst/onionize"
	"rsc.io/qr"
)

const applicationTitle = "onionize"

var win *gtk.Window

func guiMain(paramsCh chan<- onionize.Parameters, linkChan <-chan url.URL, errChan <-chan error) {
	gtk.Init(nil)

	var err error
	win, err = gtk.WindowNew(gtk.WINDOW_TOPLEVEL)
	if err != nil {
		log.Fatal("Unable to create window:", err)
	}
	win.SetTitle(applicationTitle)
	win.SetIconName("folder-publicshare")
	win.Connect("destroy", func() {
		gtk.MainQuit()
	})
	win.SetBorderWidth(5)
	win.SetDefaultSize(1, 1)
	win.SetResizable(false)

	grid, err := gtk.GridNew()
	if err != nil {
		log.Fatal("Unable to create grid:", err)
	}
	grid.SetOrientation(gtk.ORIENTATION_VERTICAL)
	grid.SetRowSpacing(12)
	grid.SetColumnSpacing(12)

	slugChkBox, err := gtk.CheckButtonNewWithLabel("slug")
	if err != nil {
		log.Fatal(err)
	}
	slugChkBox.SetActive(true)
	slugChkBox.SetHAlign(gtk.ALIGN_CENTER)
	grid.Attach(slugChkBox, 0, 1, 1, 1)

	passphraseEntry, err := gtk.EntryNew()
	if err != nil {
		log.Fatal("Unable to create entry:", err)
	}
	passphraseEntry.SetHExpand(true)
	passphraseEntry.SetPlaceholderText("identity passphrase")
	passphraseEntry.SetInputPurpose(gtk.INPUT_PURPOSE_PASSWORD)
	passphraseEntry.SetVisibility(false)

	grid.Attach(passphraseEntry, 1, 1, 1, 1)

	combo, err := gtk.ComboBoxTextNew()
	if err != nil {
		log.Fatal(err)
	}
	combo.AppendText("file")
	combo.AppendText("directory")
	combo.AppendText("zip")
	combo.SetActive(0)
	grid.Attach(combo, 0, 0, 1, 1)
	var fchooserBtn *gtk.FileChooserButton

	updateFileChooser := func(pathtype string) {
		var err error
		switch pathtype {
		case "directory":
			fchooserBtn, err = gtk.FileChooserButtonNew("Select a path", gtk.FILE_CHOOSER_ACTION_SELECT_FOLDER)
			if err != nil {
				log.Fatal(err)
			}
		case "file":
			fchooserBtn, err = gtk.FileChooserButtonNew("Select a path", gtk.FILE_CHOOSER_ACTION_OPEN)
			if err != nil {
				log.Fatal(err)
			}
		case "zip":
			fchooserBtn, err = gtk.FileChooserButtonNew("Select a path", gtk.FILE_CHOOSER_ACTION_OPEN)
			if err != nil {
				log.Fatal(err)
			}
			ffilter, err := gtk.FileFilterNew()
			if err != nil {
				log.Fatal(err)
			}
			ffilter.AddPattern("*.zip")
			fchooserBtn.AddFilter(ffilter)
		}
		fchooserBtn.SetHExpand(false)
		w, err := grid.GetChildAt(1, 0)
		if err == nil {
			w.Destroy()
		}
		grid.Attach(fchooserBtn, 1, 0, 1, 1)
		grid.ShowAll()
		win.Resize(1, 1)
	}
	combo.Connect("changed", func() {
		activeText := combo.GetActiveText()
		updateFileChooser(activeText)
	})
	updateFileChooser("file")

	doBtn, err := gtk.ButtonNewWithLabel("onionize")
	if err != nil {
		log.Fatal("Unable to create button:", err)
	}

	fadeOut := func() {
		fchooserBtn.SetSensitive(false)
		doBtn.SetSensitive(false)
		doBtn.SetLabel("onionizing...")
		combo.SetSensitive(false)
		slugChkBox.SetSensitive(false)
		passphraseEntry.SetSensitive(false)
		grid.ShowAll()
	}

	fadeIn := func() {
		fchooserBtn.SetSensitive(true)
		doBtn.SetSensitive(true)
		doBtn.SetLabel("onionize")
		combo.SetSensitive(true)
		slugChkBox.SetSensitive(true)
		passphraseEntry.SetSensitive(true)
		grid.ShowAll()
	}

	doBtn.Connect("clicked", func() {
		path := fchooserBtn.GetFilename()
		if path == "" {
			return
		}
		passphrase, err := passphraseEntry.GetText()
		if err != nil {
			log.Fatalf("Unable to get passphrase: %v", err)
		}
		fadeOut()
		p := onionize.Parameters{
			Debug:           debug,
			ControlPath:     "default://",
			ControlPassword: "",
			Pathspec:        path,
			Zip:             "zip" == combo.GetActiveText(),
			Slug:            slugChkBox.GetActive(),
			Passphrase:      passphrase,
		}
		paramsCh <- p

	})
	grid.Attach(doBtn, 0, 2, 2, 1)

	urlEntry, err := gtk.EntryNew()
	if err != nil {
		log.Fatal("Unable to create entry:", err)
	}
	urlEntry.SetHExpand(true)
	go func() {
		for {
			select {
			case link := <-linkChan:
				_, err = glib.IdleAdd(func() {
					linkString := link.String()
					urlEntry.SetText(linkString)
					doBtn.Destroy()
					grid.Attach(urlEntry, 0, 2, 2, 1)
					urlEntry.SelectRegion(0, len(linkString))

					qrcode, err := qr.Encode(linkString, qr.L)
					if err != nil {
						log.Fatal(err)
					}
					pbl, err := gdk.PixbufLoaderNewWithType("png")
					if err != nil {
						log.Fatalf("Failed to create a pixbuf: %v", err)
					}
					_, err = pbl.Write(qrcode.PNG())
					if err != nil {
						log.Fatalf("Failed to write to pixbuf: %v", err)
					}
					qrPixbuf, err := pbl.GetPixbuf()
					if err != nil {
						log.Fatalf("Failed to get pixbuf: %v", err)
					}
					qrCodeWidget, err := gtk.ImageNewFromPixbuf(qrPixbuf)
					if err != nil {
						log.Fatalf("Failed to create qrcode widget: %v", err)
					}
					grid.Attach(qrCodeWidget, 0, 3, 2, 1)
					grid.ShowAll()
				})
				if err != nil {
					log.Fatal(err)
				}
			case err := <-errChan:
				errDialog := gtk.MessageDialogNew(win, gtk.DIALOG_MODAL, gtk.MESSAGE_ERROR, gtk.BUTTONS_CLOSE, err.Error())
				_, err = glib.IdleAdd(func() {
					errDialog.Run()
					errDialog.Destroy()
					fadeIn()
				})
				if err != nil {
					log.Fatal(err)
				}
			}
		}
	}()

	win.Add(&grid.Container.Widget)
	win.ShowAll()

	gtk.Main()
	os.Exit(0)
}
