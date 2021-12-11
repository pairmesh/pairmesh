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

	"github.com/atotto/clipboard"
	"github.com/pairmesh/pairmesh/i18n"
	"github.com/pairmesh/pairmesh/node/driver"
	"github.com/pairmesh/pairmesh/node/resources"
	"github.com/progrium/macdriver/cocoa"
	"github.com/progrium/macdriver/core"
	"github.com/progrium/macdriver/objc"
	"go.uber.org/atomic"
	"go.uber.org/zap"
)

type osApp struct {
	baseApp

	initialized atomic.Bool
	nsApp       cocoa.NSApplication
	menuApp     cocoa.NSMenu
	statusItem  cocoa.NSStatusItem

	login      cocoa.NSMenuItem
	device     cocoa.NSMenuItem
	status     cocoa.NSMenuItem
	enable     cocoa.NSMenuItem
	console    cocoa.NSMenuItem
	myDevices  cocoa.NSMenuItem
	myNetworks cocoa.NSMenuItem
	start      cocoa.NSMenuItem
	logout     cocoa.NSMenuItem
	about      cocoa.NSMenuItem
	quit       cocoa.NSMenuItem

	// Separators which are showed only when login status
	seps []cocoa.NSMenuItem
}

func newOSApp() *osApp {
	return &osApp{
		baseApp: baseApp{events: make(chan struct{}, 4)},
	}
}

func (app *osApp) createTray() error {
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

		// Guest status
		app.login = app.addMenuItemWithAction(i18n.L("tray.login"), "handleLogin:", func(_ objc.Object) { app.onOpenLoginWeb() })

		// Login status
		app.device = app.addMenuItem(i18n.L("tray.unknown"))
		app.status = app.addMenuItem(i18n.L("tray.unknown"))
		app.status.SetEnabled(false)
		app.enable = app.addMenuItem(i18n.L("tray.enable"))

		app.seps = append(app.seps, app.addSeparator())

		app.console = app.addMenuItemWithAction(i18n.L("tray.profile.console"), "handleProfile:", func(_ objc.Object) { app.onOpenConsole() })
		app.myDevices = app.addMenu(app.menuApp, i18n.L("tray.my_devices"))
		app.myNetworks = app.addMenu(app.menuApp, i18n.L("tray.my_networks"))

		app.seps = append(app.seps, app.addSeparator())

		// General menu item
		app.start = app.addMenuItemWithAction(i18n.L("tray.autorun"), "handleAutoRun:", func(_ objc.Object) { app.onAutoStart() })
		if app.auto.IsEnabled() {
			app.start.SetState(1)
		} else {
			app.start.SetState(0)
		}

		app.logout = app.addMenuItemWithAction(i18n.L("tray.profile.logout"), "handleLogout:", func(_ objc.Object) { app.onLogout() })
		app.about = app.addMenuItemWithAction(i18n.L("tray.about"), "handleAbout:", func(_ objc.Object) { app.onOpenAbout() })

		app.addSeparator()

		app.quit = app.addMenuItemWithAction(i18n.L("tray.exit"), "handleQuit:", func(_ objc.Object) { app.onQuit() })

		app.initialized.Store(true)
		app.render(app.driver.Summarize())
	})

	return nil
}

func (app *osApp) run() {
	app.nsApp.Run()
}

func (app *osApp) render(summary *driver.Summary) {
	if !app.initialized.Load() {
		return
	}

	isGuest := app.cfg.IsGuest()

	// Guest status menu items
	app.login.SetHidden(!isGuest)

	// Login status menu items
	app.device.SetHidden(isGuest)
	app.status.SetHidden(isGuest)
	app.console.SetHidden(isGuest)
	app.enable.SetHidden(isGuest)
	app.myDevices.SetHidden(isGuest)
	app.myNetworks.SetHidden(isGuest)
	app.logout.SetHidden(isGuest)
	// Hidden some separators
	for _, sep := range app.seps {
		sep.SetHidden(isGuest)
	}

	if !isGuest {
		profile := summary.Profile

		// Current device information.
		desc := fmt.Sprintf("%s (%s)", profile.Name, profile.IPv4)
		app.device.SetTitle(desc)
		app.replaceHandler(app.device, "handleSelfAddress:", app.copyAddressToClipboard(profile.Name, profile.IPv4))

		// Display current network status.
		app.status.SetTitle(i18n.L("tray.status." + summary.Status))

		// Enabled devices
		if summary.Enabled {
			app.enable.SetState(1)
		} else {
			app.enable.SetState(0)
		}
		app.replaceHandler(app.enable, "handleEnabled:", func(object objc.Object) { app.switchDriverEnable(summary.Enabled) })

		// My devices list
		app.myDevices.SetEnabled(len(summary.Mesh.MyDevices) != 0)
		myDevicesMenu := app.myDevices.Submenu()
		devicesShowCount := myDevicesMenu.ItemArray().Count()
		for i, d := range summary.Mesh.MyDevices {
			deviceName := i18n.L("tray.device", fmt.Sprintf("%s (%s)", d.Name, d.IPv4))
			handleName := fmt.Sprintf("handleDeviceAddress%d:", i)
			if i < int(devicesShowCount) {
				device := myDevicesMenu.ItemAtIndex_(core.NSInteger(i))
				device.SetTitle(deviceName)
				device.SetAction(objc.Sel(handleName))
				cocoa.DefaultDelegateClass.AddMethod(handleName, app.copyAddressToClipboard(d.Name, d.IPv4))
			} else {
				item := cocoa.NSMenuItem_New()
				item.SetTitle(deviceName)
				item.SetAction(objc.Sel(handleName))
				cocoa.DefaultDelegateClass.AddMethod(handleName, app.copyAddressToClipboard(d.Name, d.IPv4))
				myDevicesMenu.AddItem(item)
			}
		}
		app.removeExtraItem(myDevicesMenu, len(summary.Mesh.MyDevices))

		// My networks list
		app.myNetworks.SetEnabled(len(summary.Mesh.Networks) != 0)
		myNetworksMenu := app.myNetworks.Submenu()
		networksShowCount := int(myNetworksMenu.ItemArray().Count())
		for j, n := range summary.Mesh.Networks {
			if j < networksShowCount {
				network := myNetworksMenu.ItemAtIndex_(core.NSInteger(j))
				network.SetTitle(n.Name)
				network.SetEnabled(len(n.Devices) != 0)
				submenu := network.Submenu()
				devicesShowCount := int(submenu.ItemArray().Count())
				for i, d := range n.Devices {
					deviceName := i18n.L("tray.device", fmt.Sprintf("%s (%s)", d.Name, d.IPv4))
					handleName := fmt.Sprintf("handleDeviceAddress%d_%d:", j, i)
					if i < devicesShowCount {
						device := submenu.ItemAtIndex_(core.NSInteger(i))
						device.SetTitle(deviceName)
						device.SetAction(objc.Sel(handleName))
						cocoa.DefaultDelegateClass.AddMethod(handleName, app.copyAddressToClipboard(d.Name, d.IPv4))
					} else {
						item := cocoa.NSMenuItem_New()
						item.SetTitle(deviceName)
						item.SetAction(objc.Sel(handleName))
						cocoa.DefaultDelegateClass.AddMethod(handleName, app.copyAddressToClipboard(d.Name, d.IPv4))
						submenu.AddItem(item)
					}
				}
				app.removeExtraItem(submenu, len(n.Devices))
			} else {
				network := app.addMenu(myNetworksMenu, n.Name)
				network.SetEnabled(len(n.Devices) != 0)
				submenu := network.Submenu()
				for i, d := range n.Devices {
					deviceName := i18n.L("tray.device", fmt.Sprintf("%s (%s)", d.Name, d.IPv4))
					handleName := fmt.Sprintf("handleDeviceAddress%d_%d:", j, i)
					item := cocoa.NSMenuItem_New()
					item.SetTitle(deviceName)
					item.SetAction(objc.Sel(handleName))
					cocoa.DefaultDelegateClass.AddMethod(handleName, app.copyAddressToClipboard(d.Name, d.IPv4))
					submenu.AddItem(item)
				}
			}
		}
		app.removeExtraItem(myNetworksMenu, len(summary.Mesh.Networks))
	}

	// Auto start
	if app.auto.IsEnabled() {
		app.start.SetState(1)
	} else {
		app.start.SetState(0)
	}
}

func (app *osApp) removeExtraItem(menu cocoa.NSMenu, expectedLength int) {
	menuItemCount := int(menu.ItemArray().Count())
	if expectedLength >= menuItemCount {
		return
	}
	for i := menuItemCount - 1; i >= expectedLength; i-- {
		menu.RemoveItemAtIndex_(core.NSInteger(i))
	}
}

func (app *osApp) replaceHandler(item cocoa.NSMenuItem, sel string, action func(object objc.Object)) {
	item.SetAction(objc.Sel(sel))
	cocoa.DefaultDelegateClass.AddMethod(sel, action)
}

func (app *osApp) addSeparator() cocoa.NSMenuItem {
	sep := cocoa.NSMenuItem_Separator()
	app.menuApp.AddItem(sep)
	return sep
}

func (app *osApp) addMenuItemWithAction(title, sel string, action func(object objc.Object)) cocoa.NSMenuItem {
	item := cocoa.NSMenuItem_New()
	item.SetTitle(title)
	item.SetAction(objc.Sel(sel))
	cocoa.DefaultDelegateClass.AddMethod(sel, action)
	app.menuApp.AddItem(item)
	return item
}

func (app *osApp) addMenuItem(title string) cocoa.NSMenuItem {
	item := cocoa.NSMenuItem_New()
	item.SetTitle(title)
	app.menuApp.AddItem(item)
	return item
}

func (app *osApp) addMenu(parent cocoa.NSMenu, title string) cocoa.NSMenuItem {
	menu := cocoa.NSMenu_New()
	item := cocoa.NSMenuItem_New()
	item.SetTitle(title)
	item.SetSubmenu(menu)
	parent.AddItem(item)
	return item
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
