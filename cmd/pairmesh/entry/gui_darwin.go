// Copyright 2021 PairMesh, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package entry

import (
	"fmt"
	"runtime"

	"github.com/atotto/clipboard"
	"github.com/pairmesh/pairmesh/cmd/pairmesh/resources"
	"github.com/pairmesh/pairmesh/i18n"
	"github.com/progrium/macdriver/cocoa"
	"github.com/progrium/macdriver/core"
	"github.com/progrium/macdriver/objc"
	"go.uber.org/zap"
)

type osApp struct {
	baseApp

	nsApp      cocoa.NSApplication
	menuApp    cocoa.NSMenu
	statusItem cocoa.NSStatusItem
}

func newOSApp() *osApp {
	return &osApp{}
}

func (app *osApp) createTray() error {
	runtime.LockOSThread()

	cocoa.TerminateAfterWindowsClose = false
	app.nsApp = cocoa.NSApp_WithDidLaunch(func(n objc.Object) {
		obj := cocoa.NSStatusBar_System().StatusItemWithLength(cocoa.NSVariableStatusItemLength)
		obj.Retain()
		logo := resources.Logo
		if app.cfg.IsGuest() {
			logo = resources.DisabledLogo
		}
		obj.Button().SetImage(logo)
		menu := cocoa.NSMenu_New()
		obj.SetMenu(menu)
		app.menuApp = menu
		app.statusItem = obj

		app.renderTrayContextMenu()
	})

	return nil
}

func (app *osApp) run() {
	app.nsApp.Run()
}

func (app *osApp) renderTrayContextMenu() {
	app.menuApp.RemoveAllItems()

	summary := app.driver.Summarize()

	if app.cfg.IsGuest() {
		// Login
		login := app.addMenuItem(i18n.L("tray.login"), "handleLogin:", func(_ objc.Object) { app.onOpenLoginWeb() })
		app.menuApp.AddItem(login)
	} else {
		// Current device information.
		profile := summary.Profile
		desc := fmt.Sprintf("%s (%s)", profile.Name, profile.IPv4)
		device := app.addMenuItem(i18n.L("tray.device", desc), "handleSelfAddress:", app.copyAddressToClipboard(profile.Name, profile.IPv4))
		app.menuApp.AddItem(device)

		// Display current network status.
		network := app.addMenuItem(i18n.L("tray.status."+summary.Status), "handleNetworkStatus:", func(object objc.Object) {})
		network.SetEnabled(false)
		app.menuApp.AddItem(network)

		// Enabled devices
		enabled := app.addMenuItem(i18n.L("tray.enable"), "handleEnabled:", func(object objc.Object) {
			if summary.Enabled {
				app.driver.Disable()
			} else {
				app.driver.Enable()
			}
		})
		if summary.Enabled {
			enabled.SetState(1)
		} else {
			enabled.SetState(0)
		}
		app.menuApp.AddItem(enabled)

		app.addSeparator(app.menuApp)

		// Profile
		profileMenu := app.addMenuItem(i18n.L("tray.profile.console"), "handleProfile:", func(_ objc.Object) { app.onOpenConsole() })
		app.menuApp.AddItem(profileMenu)

		// My devices list
		myDevicesMenu, myDevices := app.addMenu(app.menuApp, i18n.L("tray.my_devices"))
		myDevices.SetEnabled(len(summary.Mesh.MyDevices) != 0)
		app.menuApp.AddItem(myDevices)
		for i, d := range summary.Mesh.MyDevices {
			desc := fmt.Sprintf("%s (%s)", d.Name, d.IPv4)
			handleName := fmt.Sprintf("handleDeviceAddress%d:", i)
			item := app.addMenuItem(i18n.L("tray.device", desc), handleName, app.copyAddressToClipboard(d.Name, d.IPv4))
			myDevicesMenu.AddItem(item)
		}

		myNetworksMenu, myNetwork := app.addMenu(app.menuApp, i18n.L("tray.my_networks"))
		myNetwork.SetEnabled(len(summary.Mesh.Networks) != 0)
		app.menuApp.AddItem(myNetwork)
		for j, n := range summary.Mesh.Networks {
			submenu, sub := app.addMenu(myNetworksMenu, n.Name)
			myNetworksMenu.AddItem(sub)
			sub.SetEnabled(len(n.Devices) != 0)
			for i, d := range n.Devices {
				desc := fmt.Sprintf("%s (%s)", d.Name, d.IPv4)
				handleName := fmt.Sprintf("handleDeviceAddress%d_%d:", j, i)
				item := app.addMenuItem(i18n.L("tray.device", desc), handleName, app.copyAddressToClipboard(d.Name, d.IPv4))
				submenu.AddItem(item)
			}
		}
	}

	app.addSeparator(app.menuApp)

	// Auto start
	start := app.addMenuItem(i18n.L("tray.autorun"), "handleAutoRun:", func(_ objc.Object) { app.onAutoStart() })
	if app.auto.IsEnabled() {
		start.SetState(1)
	} else {
		start.SetState(0)
	}
	app.menuApp.AddItem(start)

	if !app.cfg.IsGuest() {
		logout := app.addMenuItem(i18n.L("tray.profile.logout"), "handleLogout:", func(_ objc.Object) { app.onLogout() })
		app.menuApp.AddItem(logout)
	}

	// About
	about := app.addMenuItem(i18n.L("tray.about"), "handleAbout:", func(_ objc.Object) { app.onOpenAbout() })
	app.menuApp.AddItem(about)

	app.addSeparator(app.menuApp)

	// Quit
	quit := app.addMenuItem(i18n.L("tray.exit"), "handleQuit:", func(_ objc.Object) { app.onQuit() })
	app.menuApp.AddItem(quit)
}

func (app *osApp) addSeparator(menu cocoa.NSMenu) {
	sep := cocoa.NSMenuItem_Separator()
	menu.AddItem(sep)
}

func (app *osApp) addMenuItem(title, sel string, action func(object objc.Object)) cocoa.NSMenuItem {
	item := cocoa.NSMenuItem_New()
	item.SetTitle(title)
	item.SetAction(objc.Sel(sel))
	cocoa.DefaultDelegateClass.AddMethod(sel, action)
	return item
}

func (app *osApp) addMenu(parent cocoa.NSMenu, title string) (cocoa.NSMenu, cocoa.NSMenuItem) {
	menu := cocoa.NSMenu_New()
	item := cocoa.NSMenuItem_New()
	item.SetTitle(title)
	item.SetSubmenu(menu)
	return menu, item
}

func (app *osApp) dispose() {
	app.nsApp.Autorelease()
	app.menuApp.Autorelease()
	app.statusItem.Autorelease()
}

func (app *osApp) showMessage(title, message string) {

}

func (app *osApp) showAlbert(title, msg string) bool {
	//alert := cocoa.NSAlert_New()
	//
	//alert.SetIcon(app.alertLogo)
	//alert.SetAlertStyle(cocoa.NSAlertStyleInformational)
	//alert.ShowsSuppressionButton(true)
	//alert.SetTitle(title)
	//alert.SetMessageText(msg)
	//
	//_ = alert.RunModal()
	//state := alert.SuppressionButton().State()
	//return state == suppressed
	return false
}

func (app *osApp) copyAddressToClipboard(name, address string) func(object objc.Object) {
	return func(object objc.Object) {
		err := clipboard.WriteAll(address)
		if err != nil {
			zap.L().Error("Write to clipboard failed", zap.Error(err))
			return
		}

		if app.cfg.OnceAlert {
			return
		}
		core.Dispatch(func() {
			msg := i18n.L("tray.toast.my_devices.tips_format", address)
			suppressed := app.showAlbert(name, msg)
			if suppressed {
				app.cfg.OnceAlert = true
				if err := app.cfg.Save(); err != nil {
					zap.L().Error("Set once alert failed", zap.Error(err))
				}
			}
		})
	}
}

func (app *osApp) setTrayIcon(enabled bool) {
	if enabled {
		app.statusItem.Button().SetImage(resources.DisabledLogo)
	} else {
		app.statusItem.Button().SetImage(resources.Logo)
	}
}
