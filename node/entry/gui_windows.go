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
	"github.com/lxn/walk"
	"github.com/pairmesh/pairmesh/i18n"
	"github.com/pairmesh/pairmesh/node/driver"
	"github.com/pairmesh/pairmesh/node/resources"
	"go.uber.org/zap"
)

type osApp struct {
	baseApp

	mw   *walk.MainWindow
	tray *walk.NotifyIcon

	login      *walk.Action
	device     *walk.Action
	status     *walk.Action
	enable     *walk.Action
	console    *walk.Action
	myDevices  *walk.Action
	myNetworks *walk.Action
	start      *walk.Action
	logout     *walk.Action
	about      *walk.Action
	language   *walk.Action
	exit       *walk.Action

	// Separators which are showed only when login status
	seps []*walk.Action

	myDevicesList  []*walk.Action
	myNetworksList []*walk.Action
	myDevicesSep   *walk.Action
	myNetworksSep  *walk.Action

	actionLocaleNameMap map[*walk.Action]string
}

func newOSApp() *osApp {
	return &osApp{
		baseApp:             baseApp{events: make(chan struct{}, 4)},
		actionLocaleNameMap: make(map[*walk.Action]string),
	}
}

func (app *osApp) run() {
	err := app.createTray()
	if err != nil {
		zap.L().Fatal("Create tray failed", zap.Error(err))
	}

	app.mw.Run()
}

func (app *osApp) dispose() {
	zap.L().Info("This node is shutdown, see you next time.")
	_ = app.tray.Dispose()
	app.mw.Dispose()
	walk.App().Exit(0)
}

func (app *osApp) createTray() error {
	wnd, err := walk.NewMainWindow()
	if err != nil {
		return fmt.Errorf("create main window failed: %w", err)
	}
	app.mw = wnd

	tray, err := walk.NewNotifyIcon(app.mw)
	if err != nil {
		return fmt.Errorf("create notify icon failed: %w", err)
	}
	app.tray = tray
	app.tray.MouseDown().Attach(app.onShowTray)

	err = tray.SetIcon(resources.Logo)
	if err != nil {
		return err
	}

	// Guest status
	app.login = app.addActionWithActionWithTK(nil, "tray.login", app.onOpenLoginWeb)

	// Login status
	app.device = app.addActionWithTK(nil, "tray.unknown")
	app.status = app.addActionWithTK(nil, "tray.unknown")
	app.status.SetEnabled(false)
	app.enable = app.addActionWithTK(nil, "tray.enable")
	app.console = app.addActionWithActionWithTK(nil, "tray.profile.console", app.onOpenConsole)

	app.seps = append(app.seps, app.addSeparator())

	app.myDevices = app.addActionWithTK(nil, "tray.my_devices")
	app.myDevices.SetEnabled(false)
	app.myDevicesSep = app.addSeparator()
	app.myNetworks = app.addActionWithTK(nil, "tray.my_networks")
	app.myNetworks.SetEnabled(false)
	app.myNetworksSep = app.addSeparator()

	app.seps = append(app.seps, app.myDevicesSep, app.myNetworksSep)

	// General menu item
	app.start = app.addActionWithActionWithTK(nil, "tray.autorun", app.onAutoStart)
	app.start.SetChecked(app.auto.IsEnabled())
	app.logout = app.addActionWithActionWithTK(nil, "tray.profile.logout", app.onLogout)
	app.about = app.addActionWithActionWithTK(nil, "tray.about", app.onOpenAbout)
	app.addSeparator()

	app.language = app.addMenuAction(nil, i18n.L("tray.language", i18n.GetCurrentLocaleName()))
	app.addSeparator()

	app.exit = app.addActionWithActionWithTK(nil, "tray.exit", app.onQuit)

	app.initialized.Store(true)
	app.setMenuVisibility(app.cfg.IsGuest())
	app.refreshEvent()

	return tray.SetVisible(true)
}

// onShowTray refreshes the tray menu items when the tray button clicked.
func (app *osApp) onShowTray(_, _ int, _ walk.MouseButton) {
	app.refreshEvent()
}

func (app *osApp) setMenuVisibility(isGuest bool) {
	// Guest status menu items
	app.login.SetVisible(isGuest)

	// Login status menu items
	app.device.SetVisible(!isGuest)
	app.status.SetVisible(!isGuest)
	app.console.SetVisible(!isGuest)
	app.enable.SetVisible(!isGuest)
	app.myDevices.SetVisible(!isGuest)
	app.myNetworks.SetVisible(!isGuest)
	app.logout.SetVisible(!isGuest)

	// Change some separator visibility.
	for _, sep := range app.seps {
		sep.SetVisible(!isGuest)
	}

	if isGuest {
		for _, action := range app.myDevicesList {
			action.SetVisible(false)
		}
		for _, action := range app.myNetworksList {
			action.SetVisible(false)
		}
	}
}

// Only called by UI thread.
func (app *osApp) render(summary *driver.Summary) {
	isGuest := app.cfg.IsGuest()

	app.setMenuVisibility(isGuest)

	if !isGuest {
		profile := summary.Profile

		// Only keep the first handler.
		app.device.SetText(i18n.L("tray.device", profile.Name))
		app.replaceHandler(app.device, app.copyAddressToClipboard(profile.Name, profile.IPv4))

		// Display current device network status.
		app.status.SetText(i18n.L("tray.status." + summary.Status))

		// Enabled devices
		app.enable.SetChecked(summary.Enabled)
		app.replaceHandler(app.enable, func() { app.switchDriverEnable(summary.Enabled) })

		contextMenu := app.tray.ContextMenu()

		// My devices list
		deviceShowCount := len(app.myDevicesList)
		for i, d := range summary.Mesh.MyDevices {
			deviceName := fmt.Sprintf("%s\t%s", d.Name, d.IPv4)
			var device *walk.Action
			if i < deviceShowCount {
				device = app.myDevicesList[i]
				device.SetVisible(true)
				app.replaceHandler(device, app.copyAddressToClipboard(d.Name, d.IPv4))
			} else {
				device = walk.NewAction()
				device.Triggered().Attach(app.copyAddressToClipboard(d.Name, d.IPv4))
				contextMenu.Actions().Insert(contextMenu.Actions().Index(app.myDevicesSep), device)
				app.myDevicesList = append(app.myDevicesList, device)
			}
			device.SetText(deviceName)
			device.SetVisible(true)
		}
		// remove extra actions
		if actualDeviceCount := len(summary.Mesh.MyDevices); actualDeviceCount < len(app.myDevicesList) {
			for _, action := range app.myDevicesList[actualDeviceCount:] {
				contextMenu.Actions().Remove(action)
			}
			app.myDevicesList = app.myDevicesList[:actualDeviceCount]
		}

		// My networks list
		networksShowCount := len(app.myNetworksList)
		for i, n := range summary.Mesh.Networks {
			var network *walk.Action
			if i < networksShowCount {
				network = app.myNetworksList[i]
				submenu := network.Menu()
				deviceShowCount := submenu.Actions().Len()
				for _, d := range n.Devices {
					deviceName := fmt.Sprintf("%s\t%s", d.Name, d.IPv4)
					if i < deviceShowCount {
						device := submenu.Actions().At(i)
						device.SetText(deviceName)
						device.SetVisible(true)
						app.replaceHandler(device, app.copyAddressToClipboard(d.Name, d.IPv4))
					} else {
						app.addAction(submenu, deviceName).Triggered().Attach(app.copyAddressToClipboard(d.Name, d.IPv4))
					}
				}
				app.removeExtraItem(submenu, len(n.Devices))
			} else {
				submenu, err := walk.NewMenu()
				if err != nil {
					continue
				}
				network = walk.NewMenuAction(submenu)
				contextMenu.Actions().Insert(contextMenu.Actions().Index(app.myNetworksSep), network)
				app.myNetworksList = append(app.myNetworksList, network)
				for _, d := range n.Devices {
					deviceName := fmt.Sprintf("%s\t%s", d.Name, d.IPv4)
					app.addAction(submenu, deviceName).Triggered().Attach(app.copyAddressToClipboard(d.Name, d.IPv4))
				}
			}
			network.SetText(n.Name)
			network.SetVisible(true)
			network.SetEnabled(len(n.Devices) != 0)
		}
		// remove extra actions
		if actualDeviceCount := len(summary.Mesh.Networks); actualDeviceCount < len(app.myNetworksList) {
			for _, action := range app.myNetworksList[actualDeviceCount:] {
				contextMenu.Actions().Remove(action)
			}
			app.myNetworksList = app.myNetworksList[:actualDeviceCount]
		}
	}

	app.displayLanguageList()
	app.start.SetChecked(app.auto.IsEnabled())
}

func (app *osApp) removeExtraItem(menu *walk.Menu, expectLength int) {
	menuItemCount := menu.Actions().Len()
	if expectLength >= menuItemCount {
		return
	}
	for i := menuItemCount - 1; i >= expectLength; i-- {
		menu.Actions().RemoveAt(i)
	}
}

// replaceHandler attach the handler to the action trigger list and remove all the previous.
func (app *osApp) replaceHandler(action *walk.Action, handler walk.EventHandler) {
	index := action.Triggered().Attach(handler)
	if index == 0 {
		return
	}
	// Remove existing handlers
	for i := 0; i < index; i++ {
		action.Triggered().Detach(i)
	}
}

func (app *osApp) addMenuAction(parent *walk.Menu, title string) *walk.Action {
	submenu, _ := walk.NewMenu()
	action := walk.NewMenuAction(submenu)
	_ = action.SetVisible(true)
	_ = action.SetText(title)
	if parent != nil {
		_ = parent.Actions().Add(action)
		_ = action.SetVisible(true)
	} else {
		_ = app.tray.ContextMenu().Actions().Add(action)
	}
	return action
}

func (app *osApp) addActionWithTK(parent *walk.Menu, titleKey string) *walk.Action {
	action := app.addAction(parent, i18n.L(titleKey))
	app.actionLocaleNameMap[action] = titleKey
	return action
}

func (app *osApp) addAction(parent *walk.Menu, title string) *walk.Action {
	action := walk.NewAction()
	_ = action.SetVisible(true)
	_ = action.SetText(title)
	if parent != nil {
		_ = parent.Actions().Add(action)
		_ = action.SetVisible(true)
	} else {
		_ = app.tray.ContextMenu().Actions().Add(action)
	}
	return action
}

func (app *osApp) addActionWithActionWithTK(parent *walk.Menu, titleKey string, handler walk.EventHandler) *walk.Action {
	action := app.addActionWithAction(parent, i18n.L(titleKey), handler)
	app.actionLocaleNameMap[action] = titleKey
	return action
}

func (app *osApp) addActionWithAction(parent *walk.Menu, title string, handler walk.EventHandler) *walk.Action {
	action := app.addAction(parent, title)
	action.Triggered().Attach(handler)
	return action
}

func (app *osApp) addSeparator() *walk.Action {
	action := walk.NewSeparatorAction()
	_ = app.tray.ContextMenu().Actions().Add(action)
	return action
}

func (app *osApp) addMenu(name string) *walk.Action {
	menu, err := walk.NewMenu()
	if err != nil {
		return nil
	}

	action := walk.NewMenuAction(menu)
	_ = action.SetText(name)
	err = app.tray.ContextMenu().Actions().Add(action)
	if err != nil {
		return nil
	}
	return action
}

func (app *osApp) openLogWindow() {
	zap.L().Warn("Open log windows not implemented")
}

func (app *osApp) copyAddressToClipboard(name, address string) func() {
	return func() {
		err := clipboard.WriteAll(address)
		if err != nil {
			zap.L().Error("Write to clipboard failed", zap.Error(err))
			return
		}
		_ = app.tray.ShowCustom(name, i18n.L("tray.toast.peer_address.tips_format", address), resources.Logo)
	}
}

func (app *osApp) showMessage(title, content string) {
	walk.MsgBox(app.mw, title, content, walk.MsgBoxApplModal)
}

func (app *osApp) displayLanguageList() {
	locales := i18n.Locales()
	lSubmenu := app.language.Menu()
	lShowCount := lSubmenu.Actions().Len()
	curName := i18n.GetCurrentLocaleName()
	for i, name := range locales {
		desc := i18n.GetLocaleDesc(name)
		checked := (curName == name)
		var language *walk.Action
		if i < lShowCount {
			language = lSubmenu.Actions().At(i)
			language.SetText(desc)
			app.replaceHandler(language, app.setLocale(name))
		} else {
			language = app.addAction(lSubmenu, desc)
			language.Triggered().Attach(app.setLocale(name))
		}
		language.SetEnabled(!checked)
		language.SetChecked(checked)
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
			app.language.SetText(i18n.L("tray.language", i18n.GetCurrentLocaleName()))

			summary := app.driver.Summarize()
			app.render(summary)

			app.cfg.SetLocaleName(name)
		}
	}
}
