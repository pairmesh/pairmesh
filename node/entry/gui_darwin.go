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
	"github.com/pairmesh/pairmesh/node/entry/systray"
	"github.com/pairmesh/pairmesh/node/resources"
	"go.uber.org/atomic"
	"go.uber.org/zap"
)

type osApp struct {
	baseApp

	initialized atomic.Bool
	monitorInit atomic.Bool

	login      *systray.MenuItem
	device     *systray.MenuItem
	status     *systray.MenuItem
	enable     *systray.MenuItem
	console    *systray.MenuItem
	myDevices  *systray.MenuItem
	myNetworks *systray.MenuItem
	start      *systray.MenuItem
	logout     *systray.MenuItem
	about      *systray.MenuItem
	quit       *systray.MenuItem

	// Separators which are showed only when login status
	seps []*systray.MenuItem

	myDevicesList  []*systray.MenuItem
	myNetworksList []*systray.MenuItem
	myDevicesSep   *systray.MenuItem
	myNetworksSep  *systray.MenuItem
}

func newOSApp() *osApp {
	return &osApp{
		baseApp: baseApp{events: make(chan struct{}, 4)},
	}
}

func (app *osApp) createTray() {
	app.setTrayIcon(app.cfg.IsGuest())

	// Guest status
	app.login = app.addMenuItemWithAction(i18n.L("tray.login"), app.onOpenLoginWeb)

	// Login status
	app.device = app.addMenuItem(i18n.L("tray.unknown"))
	app.status = app.addMenuItem(i18n.L("tray.unknown"))
	app.status.SetDisabled(true)
	app.enable = app.addMenuItem(i18n.L("tray.enable"))
	app.console = app.addMenuItemWithAction(i18n.L("tray.profile.console"), app.onOpenConsole)

	app.seps = append(app.seps, app.addSeparator())

	app.myDevices = app.addMenuItem(i18n.L("tray.my_devices"))
	app.myDevices.SetDisabled(true)
	app.myDevicesSep = app.addSeparator()
	app.myNetworks = app.addMenuItem(i18n.L("tray.my_networks"))
	app.myNetworks.SetDisabled(true)
	app.myNetworksSep = app.addSeparator()

	app.seps = append(app.seps, app.myDevicesSep, app.myNetworksSep)

	// General menu item
	app.start = app.addMenuItemWithAction(i18n.L("tray.autorun"), app.onAutoStart)
	app.start.SetChecked(app.auto.IsEnabled())

	app.logout = app.addMenuItemWithAction(i18n.L("tray.profile.logout"), app.onLogout)
	app.about = app.addMenuItemWithAction(i18n.L("tray.about"), app.onOpenAbout)

	app.addSeparator()

	app.quit = app.addMenuItemWithAction(i18n.L("tray.exit"), app.onQuit)

	app.initialized.Store(true)
	app.setMenuVisibility(app.cfg.IsGuest())
	app.refreshEvent()
}

func (app *osApp) run() {
	if !app.monitorInit.Swap(true) {
		go func() {
			for {
				m := <-systray.Events()
				a := m.Action()
				if a != nil {
					a()
				}
			}
		}()
	}
	systray.Run(app.createTray, nil)
}

func (app *osApp) setMenuVisibility(isGuest bool) {
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

	if isGuest {
		// Hidden some separators
		for _, sep := range app.seps {
			sep.SetHidden(true)
		}
		for _, item := range app.myDevicesList {
			item.SetHidden(true)
		}
		for _, item := range app.myNetworksList {
			item.SetHidden(true)
		}
	}
}

func (app *osApp) render(summary *driver.Summary) {
	if !app.initialized.Load() {
		return
	}

	isGuest := app.cfg.IsGuest() && false
	summary.Profile.Name = "xxx"
	summary.Profile.IPv4 = "10.0.2.3"

	app.setMenuVisibility(isGuest)

	if !isGuest {
		profile := summary.Profile

		// Current device information.
		app.device.SetTitle(i18n.L("tray.device", fmt.Sprintf("%s (%s)", profile.Name, profile.IPv4)))
		app.device.SetAction(app.copyAddressToClipboard(profile.Name, profile.IPv4))

		// Display current network status.
		app.status.SetTitle(i18n.L("tray.status." + summary.Status))

		// Enabled devices
		app.enable.SetChecked(summary.Enabled)
		app.enable.SetAction(func() { app.switchDriverEnable(summary.Enabled) })

		// My devices list
		devicesShowCount := len(app.myDevicesList)
		for i, d := range summary.Mesh.MyDevices {
			deviceName := fmt.Sprintf("%s\t%s", d.Name, d.IPv4)
			var device *systray.MenuItem
			if i < devicesShowCount {
				device = app.myDevicesList[i]
				device.SetTitle(deviceName)
				device.SetHidden(false)
			} else {
				device = systray.NewMenuItem(deviceName)
				systray.AddMenuItemBefore(nil, device, app.myDevicesSep)
				app.myDevicesList = append(app.myDevicesList, device)
			}
			device.SetAction(app.copyAddressToClipboard(d.Name, d.IPv4))
		}
		// remove extra menu items
		if actualDeviceCount := len(summary.Mesh.MyDevices); actualDeviceCount < len(app.myDevicesList) {
			for _, item := range app.myDevicesList[actualDeviceCount:] {
				item.SetHidden(true)
			}
		}

		// My networks list
		networksShowCount := len(app.myNetworksList)
		for j, n := range summary.Mesh.Networks {
			var network *systray.MenuItem
			if j < networksShowCount {
				network = app.myNetworksList[j]
				children := network.Children()
				deviceShowCount := len(children)
				for i, d := range n.Devices {
					deviceName := fmt.Sprintf("%s\t%s", d.Name, d.IPv4)
					var device *systray.MenuItem
					if i < deviceShowCount {
						device = children[i]
						device.SetTitle(deviceName)
						device.SetHidden(false)
					} else {
						device = systray.NewMenuItem(deviceName)
						systray.AddMenuItem(network, device)
					}
					device.SetAction(app.copyAddressToClipboard(d.Name, d.IPv4))
				}
				// remove extra menu items
				if actualDeviceCount := len(n.Devices); actualDeviceCount < deviceShowCount {
					for _, item := range children[actualDeviceCount:] {
						item.SetHidden(true)
					}
				}
			} else {
				network = systray.NewMenuItem(n.Name)
				systray.AddMenuItemBefore(nil, network, app.myNetworksSep)
				app.myNetworksList = append(app.myNetworksList, network)
				for _, d := range n.Devices {
					deviceName := fmt.Sprintf("%s\t%s", d.Name, d.IPv4)
					device := systray.NewMenuItem(deviceName)
					device.SetAction(app.copyAddressToClipboard(d.Name, d.IPv4))
					systray.AddMenuItem(network, device)
				}
			}
			network.SetTitle(n.Name)
			network.SetHidden(false)
			network.SetDisabled(len(n.Devices) == 0)
		}
		// remove extra menu items
		if actualDeviceCount := len(summary.Mesh.Networks); actualDeviceCount < len(app.myNetworksList) {
			for _, item := range app.myNetworksList[actualDeviceCount:] {
				item.SetHidden(true)
			}
		}
	}

	app.start.SetDisabled(!app.auto.IsEnabled())
}

func (app *osApp) addSeparator() *systray.MenuItem {
	sep := systray.NewSeparator()
	systray.AddMenuItem(nil, sep)
	return sep
}

func (app *osApp) addMenuItemWithAction(title string, action func()) *systray.MenuItem {
	item := systray.NewMenuItem(title)
	item.SetAction(action)
	systray.AddMenuItem(nil, item)
	return item
}

func (app *osApp) addMenuItem(title string) *systray.MenuItem {
	item := systray.NewMenuItem(title)
	systray.AddMenuItem(nil, item)
	return item
}

func (app *osApp) addMenu(parent *systray.MenuItem, title string) *systray.MenuItem {
	menu := systray.NewMenuItem(title)
	systray.AddMenuItem(parent, menu)
	return menu
}

func (app *osApp) dispose() {
	systray.Quit()
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

func (app *osApp) copyAddressToClipboard(name, address string) func() {
	return func() {
		err := clipboard.WriteAll(address)
		if err != nil {
			zap.L().Error("Write to clipboard failed", zap.Error(err))
			return
		}

		if app.cfg.OnceAlert {
			return
		}
		//core.Dispatch(func() {
		//	msg := i18n.L("tray.toast.my_devices.tips_format", address)
		//	suppressed := app.showAlbert(name, msg)
		//	if suppressed {
		//		app.cfg.OnceAlert = true
		//		if err := app.cfg.Save(); err != nil {
		//			zap.L().Error("Set once alert failed", zap.Error(err))
		//		}
		//	}
		//})
	}
}

func (app *osApp) setTrayIcon(enabled bool) {
	if enabled {
		systray.SetIcon(resources.DisabledLogo)
	} else {
		systray.SetIcon(resources.Logo)
	}
}
