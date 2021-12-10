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
	"github.com/pairmesh/pairmesh/node/resources"
	"go.uber.org/zap"
)

type osApp struct {
	baseApp

	mw   *walk.MainWindow
	tray *walk.NotifyIcon
}

func newOSApp() *osApp {
	return &osApp{}
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

	icon := resources.Logo
	if app.cfg.IsGuest() {
		icon = resources.LogoDisabled
	}
	err = tray.SetIcon(icon)
	if err != nil {
		return err
	}

	tray.MouseDown().Attach(app.renderTrayContextMenu)

	return tray.SetVisible(true)
}

func (app *osApp) run() {
	app.mw.Run()
}

func (app *osApp) dispose() {
	zap.L().Info("This node is shutdown, see you next time.")
	_ = app.tray.Dispose()
	app.mw.Dispose()
	walk.App().Exit(0)
}

// onTrayClicked handles the app event
func (app *osApp) renderTrayContextMenu(x, y int, button walk.MouseButton) {
	err := app.tray.ContextMenu().Actions().Clear()
	if err != nil {
		zap.L().Error("Clear actions failed", zap.Error(err))
		return
	}

	summary := app.driver.Summarize()

	if app.cfg.IsGuest() {
		// Login
		login := app.addAction(nil, i18n.L("tray.login"))
		login.Triggered().Attach(app.onOpenLoginWeb)
	} else {
		profile := summary.Profile
		// Current device information.
		app.addAction(nil, i18n.L("tray.device", fmt.Sprintf("%s (%s)", profile.Name, profile.IPv4))).
			Triggered().Attach(app.copyAddressToClipboard(profile.Name, profile.IPv4))

		// Display current device network status.
		app.addAction(nil, i18n.L("tray.status."+summary.Status)).SetEnabled(false)

		// Enabled devices
		enable := app.addAction(nil, i18n.L("tray.enable"))
		enable.Triggered().Attach(func() {
			if summary.Enabled {
				app.driver.Disable()
			} else {
				app.driver.Enable()
			}
		})
		_ = enable.SetChecked(summary.Enabled)

		app.addSeparator()

		// Profile
		app.addAction(nil, i18n.L("tray.profile.console")).
			Triggered().Attach(app.onOpenConsole)

		// My devices list
		myDevicesMenu, myDevice, _ := app.addMenu(i18n.L("tray.my_devices"))
		_ = myDevice.SetEnabled(len(summary.Mesh.MyDevices) != 0)
		for _, d := range summary.Mesh.MyDevices {
			app.addAction(myDevicesMenu, i18n.L("tray.device", fmt.Sprintf("%s (%s)", d.Name, d.IPv4))).
				Triggered().Attach(app.copyAddressToClipboard(d.Name, d.IPv4))
		}

		myNetworksMenu, myNetworks, _ := app.addMenu(i18n.L("tray.my_networks"))
		_ = myNetworks.SetEnabled(len(summary.Mesh.Networks) != 0)
		for _, n := range summary.Mesh.Networks {
			submenu, err := walk.NewMenu()
			if err != nil {
				continue
			}
			sub := walk.NewMenuAction(submenu)
			_ = sub.SetText(n.Name)
			_ = myNetworksMenu.Actions().Add(sub)
			_ = sub.SetEnabled(len(n.Devices) != 0)
			for _, d := range n.Devices {
				app.addAction(submenu, i18n.L("tray.device", fmt.Sprintf("%s (%s)", d.Name, d.IPv4))).
					Triggered().Attach(app.copyAddressToClipboard(d.Name, d.IPv4))
			}
		}
	}

	app.addSeparator()

	// Auto start
	start := app.addAction(nil, i18n.L("tray.autorun"))
	_ = start.SetChecked(app.auto.IsEnabled())
	start.Triggered().Attach(app.onAutoStart)

	if !app.cfg.IsGuest() {
		app.addAction(nil, i18n.L("tray.profile.logout")).
			Triggered().Attach(app.onLogout)
	}

	// About
	app.addAction(nil, i18n.L("tray.about")).
		Triggered().Attach(app.onOpenAbout)

	app.addSeparator()

	// Exit
	app.addAction(nil, i18n.L("tray.exit")).
		Triggered().Attach(app.onQuit)
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

func (app *osApp) addSeparator() {
	_ = app.tray.ContextMenu().Actions().Add(walk.NewSeparatorAction())
}

func (app *osApp) addMenu(name string) (*walk.Menu, *walk.Action, error) {
	menu, err := walk.NewMenu()
	if err != nil {
		return nil, nil, err
	}

	action := walk.NewMenuAction(menu)
	_ = action.SetText(name)
	err = app.tray.ContextMenu().Actions().Add(action)
	if err != nil {
		return nil, nil, err
	}
	return menu, action, nil
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

func (app *osApp) setTrayIcon(enable bool) {
	logo := resources.Logo
	if !enable {
		logo = resources.LogoDisabled
	}
	err := app.tray.SetIcon(logo)
	if err != nil {
		zap.L().Error("Set tray icon failed", zap.Error(err))
	}
}

func (app *osApp) showMessage(title, content string) {
	walk.MsgBox(app.mw, title, content, walk.MsgBoxApplModal)
}
