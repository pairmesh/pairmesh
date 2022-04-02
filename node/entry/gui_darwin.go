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

	"github.com/pairmesh/pairmesh/node/resources"

	"github.com/atotto/clipboard"
	"github.com/pairmesh/pairmesh/i18n"
	"github.com/pairmesh/pairmesh/node/driver"
	"github.com/pairmesh/pairmesh/node/entry/systray"
	"go.uber.org/atomic"
	"go.uber.org/zap"
)

type osApp struct {
	baseApp

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
	language   *systray.MenuItem
	quit       *systray.MenuItem

	// Separators which are showed only when login status
	seps []*systray.MenuItem

	myDevicesList  []*systray.MenuItem
	myNetworksList []*systray.MenuItem
	myDevicesSep   *systray.MenuItem
	myNetworksSep  *systray.MenuItem

	actionLocaleNameMap map[*walk.Action]string
}

func newOSApp() *osApp {
	return &osApp{
		baseApp:             baseApp{events: make(chan struct{}, 4)},
		actionLocaleNameMap: make(map[*walk.Action]string),
	}
}

func (app *osApp) createTray() {
	systray.SetTemplateIcon(resources.Logo)

	// Set locale name
	if app.cfg.LocaleName != "" {
		i18n.SetLocale(app.cfg.LocaleName)
	}

	// Guest status
	app.login = app.addMenuItemWithActionWithTK("tray.login"), pp.onOpenLoginWeb)

	// Login status
	app.device = app.addMenuItemWithTK("tray.unknown")
	app.status = app.addMenuItemWithTK("tray.unknown")
	app.status.SetDisabled(true)
	app.enable = app.addMenuItemWithTK("tray.enable")
	app.console = app.addMenuItemWithActionWithTK("tray.profile.console", app.onOpenConsole)

	app.seps = append(app.seps, app.addSeparator())

	app.myDevices = app.addMenuItemWithTK("tray.my_devices")
	app.myDevices.SetDisabled(true)
	app.myDevicesSep = app.addSeparator()
	app.myNetworks = app.addMenuItemWithTK("tray.my_networks")
	app.myNetworks.SetDisabled(true)
	app.myNetworksSep = app.addSeparator()

	app.seps = append(app.seps, app.myDevicesSep, app.myNetworksSep)

	// General menu item
	app.start = app.addMenuItemWithActionWithTK("tray.autorun"), pp.onAutoStart)
	app.start.SetChecked(app.auto.IsEnabled())

	app.logout = app.addMenuItemWithActionWithTK("tray.profile.logout", app.onLogout)
	app.about = app.addMenuItemWithActionWithTK("tray.about"), pp.onOpenAbout)
	app.addSeparator()

	app.language = app.addMenuItem(i18n.L("tray.language", i18n.GetCurrentLocaleName()))
	app.addSeparator()

	app.quit = app.addMenuItemWithActionWithTK("tray.exit"), pp.onQuit)

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

	// Change some separator visibility.
	for _, sep := range app.seps {
		sep.SetHidden(isGuest)
	}

	if isGuest {
		for _, item := range app.myDevicesList {
			item.SetHidden(true)
		}
		for _, item := range app.myNetworksList {
			item.SetHidden(true)
		}
	}
}

func (app *osApp) render(summary *driver.Summary) {
	isGuest := app.cfg.IsGuest()

	app.setMenuVisibility(isGuest)

	if !isGuest {
		profile := summary.Profile

		// Current device information.
		app.device.SetTitle(i18n.L("tray.device", profile.Name))
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

	app.displayLanguageList()
	app.start.SetDisabled(!app.auto.IsEnabled())
}

func (app *osApp) addSeparator() *systray.MenuItem {
	sep := systray.NewSeparator()
	systray.AddMenuItem(nil, sep)
	return sep
}

func (app *osApp) addMenuItemWithActionWithTK(titleKey string, action func()) *systray.MenuItem {
	act := app.addMenuItemWithAction(i18n.L(titleKey), action)
	app.actionLocaleNameMap[action] = titleKey
	return action
}

func (app *osApp) addMenuItemWithAction(title string, action func()) *systray.MenuItem {
	item := systray.NewMenuItem(title)
	item.SetAction(action)
	systray.AddMenuItem(nil, item)
	return item
}

func (app *osApp) addMenuItemWithTK(titleKey string) *systray.MenuItem {
	action := app.addMenuItem(i18n.L(titleKey))
	app.actionLocaleNameMap[action] = titleKey
	return action
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

func (app *osApp) displayLanguageList() {
	locales := i18n.Locales()
	children := app.language.Children()
	lShowCount := len(children)
	curName := i18n.GetCurrentLocaleName()
	for i, name := range locales {
		desc := i18n.GetLocaleDesc(name)
		checked := (curName == name)
		var language *systray.MenuItem
		if i < lShowCount {
			language = children[i]
			language.SetTitle(desc)
			language.SetHidden(false)
		} else {
			language = systray.NewMenuItem(desc)
			systray.AddMenuItem(app.language, language)
		}
		language.SetDisabled(checked)
		language.SetChecked(checked)
		language.SetAction(app.setLocale(name))
	}
}

func (app *osApp) setLocale(name string) func() {
	return func() {
		err := i18n.SetLocale(name)
		if err != nil {
			zap.L().Error("SetLocale failed", zap.Error(err))
		} else {
			for k, v := range app.actionLocaleNameMap {
				k.SetText(i18n.L(v))
			}
			app.language.SetTitle(i18n.L("tray.language", i18n.GetCurrentLocaleName()))

			summary := app.driver.Summarize()
			app.render(summary)

			app.cfg.SetLocaleName(name)
		}
	}
}
