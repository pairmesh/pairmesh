package systray

/*
   #cgo linux pkg-config: gtk+-3.0 appindicator3-0.1
   #cgo darwin CFLAGS: -DDARWIN -x objective-c -fobjc-arc
   #cgo darwin LDFLAGS: -framework Cocoa
   #include "systray.h"
*/
import "C"

import (
	"unsafe"
)

func register_systray() {
	C.register_systray()
}

func native_loop() {
	C.native_loop()
}

func quit() {
	C.quit()
}

// SetIcon sets the systray icon.
// iconBytes should be the content of .ico for windows and .ico/.jpg/.png
// for other platforms.
func SetIcon(iconBytes []byte) {
	cstr := (*C.char)(unsafe.Pointer(&iconBytes[0]))
	C.set_icon(cstr, (C.int)(len(iconBytes)), false)
}

// SetTitle sets the systray title, only available on Mac and Linux.
func SetTitle(title string) {
	C.set_title(C.CString(title))
}

func addSeparator(id uint32) {
	C.add_separator(C.int(id))
}

func hideMenuItem(item *MenuItem) {
	C.hide_menu_item(
		C.int(item.id),
	)
}

func showMenuItem(item *MenuItem) {
	C.show_menu_item(
		C.int(item.id),
	)
}

func removeAllItems(item *MenuItem) {
	C.remove_all_items(
		C.int(item.id),
	)
}

//export systray_ready
func systray_ready() {
	systrayReady()
}

//export systray_on_exit
func systray_on_exit() {
	systrayExit()
}

//export systray_menu_item_selected
func systray_menu_item_selected(cID C.int) {
	systrayMenuItemSelected(uint32(cID))
}

// update propagates changes on a menu item to systray
func (item *MenuItem) update() {
	var disabled C.short
	if item.disabled {
		disabled = 1
	}
	var checked C.short
	if item.checked {
		checked = 1
	}
	C.upsert_menu_item(
		C.int(item.id),
		C.int(0),
		C.int(0),
		C.CString(item.title),
		disabled,
		checked,
	)
}

// insert propagates changes on a menu item to systray
func (item *MenuItem) insert(parentID, siblingID uint32) {
	var disabled C.short
	if item.disabled {
		disabled = 1
	}
	var checked C.short
	if item.checked {
		checked = 1
	}
	C.upsert_menu_item(
		C.int(item.id),
		C.int(parentID),
		C.int(siblingID),
		C.CString(item.title),
		disabled,
		checked,
	)
}
