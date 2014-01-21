package main

import (
	"fmt"
	"github.com/conformal/gotk3/gtk"
	"io/ioutil"
	"log"
	"path"
	"runtime"
	"strings"
	"sync"
)

const (
	dict_path = "./dictionaries"
)

type Application struct {
	active_dicts map[string]bool
	ch           chan string
	done_flag    chan int
	wg           sync.WaitGroup
	result       string

	// Widgets
	win           *gtk.Window
	dict_list     *gtk.Box
	search_entry  *gtk.Entry
	search_button *gtk.Button
	text_view     *gtk.TextView
	status_bar    *gtk.Statusbar
}

func NewApplication() *Application {
	app := new(Application)
	app.ch = make(chan string)
	app.done_flag = make(chan int)

	return app
}

func (app *Application) GetDictList(path string) []string {
	var dict_files []string
	files, err := ioutil.ReadDir(path)
	if err != nil {
		log.Fatal(err)
	}
	for _, f := range files {
		dict_files = append(dict_files, f.Name())
	}
	return dict_files
}

func (app *Application) Search(text string, ch chan string, done_flag chan int) {
	id := app.status_bar.GetContextId("Search status")
	app.status_bar.Pop(id)
	app.status_bar.Push(id, "Поиск...")
	//
	// Reader from channel
	go func(ch chan string, done_flag chan int) {
		result := ""
		for {
			select {
			case s := <-ch:
				log.Println(s)
				result += s + "\n\n"
			case <-done_flag:
				log.Println("Done!")
				buf, err := app.text_view.GetBuffer()
				if err != nil {
					log.Fatal(err)
				}
				// _, end := buf.GetBounds()
				buf.SetText(result)
				app.status_bar.Pop(id)
				app.status_bar.Push(id, "Поиск завершен.")
				return
			}
		}
	}(app.ch, app.done_flag)
	//

	searchstring := strings.TrimSpace(text)
	if len(searchstring) < 1 || len(app.active_dicts) == 0 {
		return
	}
	log.Println(searchstring)
	log.Println(app.active_dicts)
	for d, _ := range app.active_dicts {
		data, err := ioutil.ReadFile(path.Join(dict_path, d))
		if err != nil {
			log.Fatal(err)
		}
		app.wg.Add(1)
		go func(data []byte, ch chan string, d string) {
			defer app.wg.Done()
			for _, entry := range strings.Split(string(data), "\n") {
				if strings.HasPrefix(strings.ToLower(entry), strings.ToLower(searchstring)+" ") {
					ch <- fmt.Sprintf("%s:\n%s", d, entry)
				}
			}
		}(data, ch, d)
	}
	app.wg.Wait()
	app.done_flag <- 1
}

func (app *Application) Run() {

	gtk.Init(nil)

	app.active_dicts = make(map[string]bool)

	builder, err := gtk.BuilderNew()
	if err != nil {
		log.Fatal(err)
	}
	builder.AddFromFile("dictsearch.glade")
	w, err := builder.GetObject("win")
	if err != nil {
		log.Fatal(err)
	}

	app.win = w.(*gtk.Window)
	app.win.Connect("destroy", func() {
		gtk.MainQuit()
	})

	// Box, containing list of dictionary chaeckbuttons
	dict_list_box, err := builder.GetObject("dict_list_box")
	if err != nil {
		log.Fatal(err)
	}
	app.dict_list = dict_list_box.(*gtk.Box)

	// Search Entry
	s_e, err := builder.GetObject("search_entry")
	if err != nil {
		log.Fatal(err)
	}
	app.search_entry = s_e.(*gtk.Entry)
	app.search_entry.Connect("activate", func() {
		text, err := app.search_entry.GetText()
		if err != nil {
			log.Fatal(err)
		}
		go app.Search(text, app.ch, app.done_flag)
	})

	// Search Button
	s_b, err := builder.GetObject("search_button")
	if err != nil {
		log.Fatal(err)
	}
	app.search_button = s_b.(*gtk.Button)
	app.search_button.Connect("clicked", func() {
		text, err := app.search_entry.GetText()
		if err != nil {
			log.Fatal(err)
		}
		go app.Search(text, app.ch, app.done_flag)
	})

	// Text View
	tv, err := builder.GetObject("textview")
	if err != nil {
		log.Fatal(err)
	}
	app.text_view = tv.(*gtk.TextView)
	app.text_view.SetEditable(false)
	app.text_view.SetWrapMode(gtk.WRAP_WORD)

	// Adding dictionary checkbuttons
	for i, dic := range app.GetDictList(dict_path) {
		dict_button, err := gtk.CheckButtonNew()
		if err != nil {
			log.Fatal(err)
		}
		label := dic
		dict_button.SetLabel(label)
		name := fmt.Sprintf("checkbutton%d", i)
		dict_button.SetName(name)
		dict_button.SetFocusOnClick(false)
		dict_button.Connect("toggled", func() {
			if dict_button.GetActive() {
				app.active_dicts[label] = true
			} else {
				delete(app.active_dicts, label)
			}
		})
		app.dict_list.Add(dict_button)
	}

	// StatusBar
	stb, err := builder.GetObject("statusbar")
	if err != nil {
		log.Fatal(err)
	}
	app.status_bar = stb.(*gtk.Statusbar)

	app.search_entry.GrabFocus()

	app.win.ShowAll()
	gtk.Main()

}

func main() {
	numcpu := runtime.NumCPU()
	runtime.GOMAXPROCS(numcpu)

	app := NewApplication()
	app.Run()
}
