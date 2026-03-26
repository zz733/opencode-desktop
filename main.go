package main

import (
	"embed"
	"runtime"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/menu"
	"github.com/wailsapp/wails/v2/pkg/menu/keys"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/mac"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	// Create an instance of the app structure
	app := NewApp()

	// Windows 使用无边框窗口（自定义标题栏），Mac 使用系统标题栏
	frameless := runtime.GOOS == "windows"

	// 初始化系统菜单
	AppMenu := menu.NewMenu()

	if runtime.GOOS == "darwin" {
		AppMenu.Append(menu.AppMenu())
		
		// 在 Mac 的系统菜单栏创建一个名为 "文件" 的菜单，并放入 "打开目录"
		fileMenu := AppMenu.AddSubmenu("文件")
		fileMenu.AddText("打开目录 / Open Directory", keys.CmdOrCtrl("o"), func(_ *menu.CallbackData) {
			app.OpenFolder()
		})

		AppMenu.Append(menu.EditMenu())
		AppMenu.Append(menu.WindowMenu())
	} else {
		// Windows 的菜单逻辑在 TitleBar.vue 中实现
	}

	// Create application with options
	err := wails.Run(&options.App{
		Title:             "OpenCode Desktop",
		Width:             1440,
		Height:            900,
		MinWidth:          1024,
		MinHeight:         768,
		WindowStartState:  options.Maximised,
		HideWindowOnClose: true,
		Menu:              AppMenu,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 25, G: 22, B: 29, A: 1},
		OnStartup:        app.startup,
		OnShutdown:       app.shutdown,
		Bind: []interface{}{
			app,
		},
		Mac: &mac.Options{
			TitleBar: &mac.TitleBar{
				TitlebarAppearsTransparent: true,
				HideTitle:                  true,
				HideTitleBar:               false,
				FullSizeContent:            true,
				UseToolbar:                 false,
			},
			WebviewIsTransparent: true,
			WindowIsTranslucent:  false,
			About: &mac.AboutInfo{
				Title:   "OpenCode Desktop",
				Message: "AI 编程助手",
			},
		},
		Windows: &windows.Options{
			WebviewIsTransparent:              false,
			WindowIsTranslucent:               false,
			Theme:                             windows.Dark,
			DisableFramelessWindowDecorations: false,
		},
		Frameless:                frameless,
		EnableDefaultContextMenu: false,
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
