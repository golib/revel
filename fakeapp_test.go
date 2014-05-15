package revel

import (
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
	"reflect"
)

type Hotel struct {
	HotelId          int
	Name, Address    string
	City, State, Zip string
	Country          string
	Price            int
}

type Hotels struct {
	*Controller
}

func (c Hotels) Index() Result {
	bookings := []*Hotel{
		&Hotel{1, "A Hotel", "300 Main St.", "New York", "NY", "10010", "USA", 300},
		&Hotel{2, "B Hotel", "200 Main St.", "San Francisco", "SF", "30010", "USA", 420},
	}

	return c.Render(bookings)
}

func (c Hotels) Show(id int) Result {
	title := "View Hotel"
	hotel := &Hotel{id, "A Hotel", "300 Main St.", "New York", "NY", "10010", "USA", 300}
	return c.Render(title, hotel)
}

func (c Hotels) Book(id int) Result {
	hotel := &Hotel{id, "A Hotel", "300 Main St.", "New York", "NY", "10010", "USA", 300}
	return c.RenderJson(hotel)
}

func (c Hotels) Plaintext() Result {
	return c.RenderText("Hello, World!")
}

func (c Hotels) Error() Result {
	c.RenderArgs["error"] = true
	c.RenderArgs["NilPointer"] = nil

	return c.Render()
}

func (c Hotels) Panic() Result {
	c.RenderArgs["panic"] = true

	return c.Render()
}

type Static struct {
	*Controller
}

func (c Static) Serve(prefix, filepath string) Result {
	var basePath, dirName string

	if !path.IsAbs(dirName) {
		basePath = BasePath
	}

	fname := path.Join(basePath, prefix, filepath)
	file, err := os.Open(fname)
	if os.IsNotExist(err) {
		return c.NotFound("")
	} else if err != nil {
		WARN.Printf("Problem opening file (%s): %s ", fname, err)
		return c.NotFound("This was found but not sure why we couldn't open it.")
	}

	return c.RenderFile(file, "")
}

func fakeTestApp() {
	Init("prod", "github.com/golib/revel/testdata/testapp", "")

	// Disable logging.
	TRACE = log.New(ioutil.Discard, "", 0)
	INFO = TRACE
	WARN = TRACE
	ERROR = TRACE

	runStartupHooks()

	// Template render panic helper
	TemplateHelpers["triggerPanic"] = func() error {
		panic("Render panic test")
		return nil
	}

	MainRouter = NewRouter("")
	if routeBytes, err := ioutil.ReadFile(filepath.Join(BasePath, "conf", "routes")); err == nil {
		MainRouter.Routes, _ = parseRoutes("", "", string(routeBytes), false)
		MainRouter.updateTree()
	}
	RevelTemplateLoader = NewTemplateLoader("default", []string{RevelTemplatePath})
	RevelTemplateLoader.Refresh()
	MainTemplateLoader = NewTemplateLoader("default", []string{ViewsPath})
	MainTemplateLoader.Refresh()

	RegisterController((*Hotels)(nil),
		[]*MethodType{
			&MethodType{
				Name: "Index",
				Args: []*MethodArg{},
				RenderArgNames: map[int][]string{
					30: []string{
						"bookings",
					},
				},
			},
			&MethodType{
				Name: "Show",
				Args: []*MethodArg{
					&MethodArg{Name: "id", Type: reflect.TypeOf((*int)(nil))},
				},
				RenderArgNames: map[int][]string{
					36: []string{
						"title",
						"hotel",
					},
				},
			},
			&MethodType{
				Name: "Book",
				Args: []*MethodArg{
					&MethodArg{Name: "id", Type: reflect.TypeOf((*int)(nil))},
				},
				RenderArgNames: map[int][]string{
					163: []string{
						"title",
						"hotel",
					},
				},
			},
			&MethodType{
				Name: "Plaintext",
			},
			&MethodType{
				Name: "Error",
			},
			&MethodType{
				Name: "Panic",
			},
		})

	RegisterController((*Static)(nil),
		[]*MethodType{
			&MethodType{
				Name: "Serve",
				Args: []*MethodArg{
					&MethodArg{Name: "prefix", Type: reflect.TypeOf((*string)(nil))},
					&MethodArg{Name: "filepath", Type: reflect.TypeOf((*string)(nil))},
				},
				RenderArgNames: map[int][]string{},
			},
		})
}
