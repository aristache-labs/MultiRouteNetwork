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

	"MultiRouteNetwork/internal/control"
	"MultiRouteNetwork/internal/icon"
	"MultiRouteNetwork/internal/state"
)

//go:embed frontend/dist
var dist embed.FS

// Set at build time with -X main.version=...
var version = "0.5.0"

func main() {
	if len(os.Args) > 1 && os.Args[1] == "version" {
		fmt.Println("MultiRoute", version)
		return
	}
	api := &API{}
	sub, err := fs.Sub(dist, "frontend/dist")
	if err != nil {
		panic(err)
	}
	app := application.New(application.Options{
		Name:        "MultiRoute",
		Icon:        icon.AppPNG(64),
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
		if err != nil && !control.IsPipeAbsent(err) && !control.IsAuthorization(err) && !control.IsEndpointDown(err) {
			return fmt.Errorf("Could not stop the routing engine: %w", err)
		}
		api.quitting.Store(true)
		app.Quit()
		return nil
	}
	api.window = &windowControl{
		dir:  state.PreferenceDir(),
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
	installDesktopTray(app, api, func() { window.Show() }, quit)
	if control.LocalStatus(api.dir()).Settings.StartRouting {
		go func() { _ = api.startRouting() }()
	}
	if err := app.Run(); err != nil {
		panic(err)
	}
}
