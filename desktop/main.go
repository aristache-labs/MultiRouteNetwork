package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"slices"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"

	"net-merge/internal/control"
)

//go:embed frontend/dist
var dist embed.FS

func main() {
	api := &API{}
	sub, err := fs.Sub(dist, "frontend/dist")
	if err != nil {
		panic(err)
	}
	app := application.New(application.Options{
		Name:        "MultiRoute",
		Description: "Choose which network adapter each app uses",
		Services: []application.Service{
			application.NewServiceWithOptions(api, application.ServiceOptions{Route: "/api"}),
		},
		Assets: application.AssetOptions{
			Handler:        http.FileServer(http.FS(sub)),
			DisableLogging: true,
		},
	})
	window := app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:            "MultiRoute",
		Hidden:           slices.Contains(os.Args, "--background"),
		Width:            1400,
		Height:           900,
		MinWidth:         960,
		MinHeight:        640,
		URL:              "/",
		BackgroundColour: application.RGBA{Red: 17, Green: 17, Blue: 17, Alpha: 255},
	})
	quit := func() error {
		_, err := api.call(control.Request{Op: "quit"})
		if err != nil && !(control.IsPipeAbsent(err) && !api.runningEngine()) {
			return fmt.Errorf("Could not stop the routing engine: %w", err)
		}
		api.quitting.Store(true)
		app.Quit()
		return nil
	}
	api.window = &windowControl{
		dir:  api.dir(),
		hide: func() { window.Hide() },
		exit: quit,
		prompt: func(message string) {
			data, _ := json.Marshal(message)
			window.Show()
			window.ExecJS("window.dispatchEvent(new CustomEvent('multiroute:close', {detail: " + string(data) + "}))")
		},
	}
	window.RegisterHook(events.Common.WindowClosing, func(event *application.WindowEvent) {
		if api.quitting.Load() {
			return
		}
		event.Cancel()
		go api.window.requestClose()
	})
	menu := app.NewMenu()
	menu.Add("Show").OnClick(func(*application.Context) { window.Show() })
	menu.Add("Exit").OnClick(func(*application.Context) {
		go func() {
			if err := quit(); err != nil {
				api.window.prompt(err.Error())
			}
		}()
	})
	tray := app.SystemTray.New()
	tray.SetMenu(menu)
	tray.SetTooltip("MultiRoute")
	if control.LocalStatus(api.dir()).Settings.StartRouting {
		go func() { _ = api.startRouting() }()
	}
	if err := app.Run(); err != nil {
		panic(err)
	}
}
