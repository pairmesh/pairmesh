package systray

import "C" //nolint
import (
	"fmt"
	"runtime"
	"sync"

	"go.uber.org/atomic"
	"go.uber.org/zap"
)

var (
	systrayReady func()
	systrayExit  func()

	currentID = &atomic.Uint32{}
	quitOnce  sync.Once
)

func init() {
	runtime.LockOSThread()
}

type (
	Action       func()
	menuItemType byte
)

const (
	menuItemTypeNormal menuItemType = 0
	menuItemTypeSep    menuItemType = 1
)

// MenuItem is used to keep track each menu item of systray.
// Don't create it directly, use the one systray.AddMenuItem() returned
type MenuItem struct {
	typ      menuItemType
	id       uint32 // id uniquely identify a menu item, not supposed to be modified
	title    string // title is the text shown on menu item
	disabled bool   // disabled menu item is grayed out and has no effect when clicked
	checked  bool   // checked menu item has a tick before the title
	hidden   bool   // hidden indicates whether is this menu item displayed
	action   Action
	children []*MenuItem
}

// NewMenuItem returns a MenuItem object with the specified title.
func NewMenuItem(title string) *MenuItem {
	return &MenuItem{
		typ:      menuItemTypeNormal,
		id:       currentID.Inc(),
		title:    title,
		disabled: false,
		checked:  false,
	}
}

// NewSeparator returns a separator without title.
func NewSeparator() *MenuItem {
	return &MenuItem{
		typ: menuItemTypeSep,
		id:  currentID.Inc(),
	}
}

// SetTitle set the text to display on a menu item
func (item *MenuItem) SetTitle(title string) {
	item.title = title
	item.update()
}

// SetDisabled sets enable status of the current menu item.
func (item *MenuItem) SetDisabled(disabled bool) {
	if item.disabled == disabled {
		return
	}
	item.disabled = disabled
	item.update()
}

// SetHidden sets the hidden status of the current menu item.
func (item *MenuItem) SetHidden(hidden bool) {
	if item.hidden == hidden {
		return
	}
	item.hidden = hidden
	if item.hidden {
		hideMenuItem(item)
	} else {
		showMenuItem(item)
	}
}

// SetChecked sets the checked status of the current menu item.
func (item *MenuItem) SetChecked(checked bool) {
	if item.checked == checked {
		return
	}
	item.checked = checked
	item.update()
}

// Action returns the event handler action.
func (item *MenuItem) Action() Action {
	return item.action
}

// SetAction sets the event handler action.
func (item *MenuItem) SetAction(action Action) {
	item.action = action
}

func (item *MenuItem) RemoveAllItems() {
	removeAllItems(item)
}

func (item *MenuItem) Children() []*MenuItem {
	return item.children
}

// String implements the fmt.Stringer interface.
func (item *MenuItem) String() string {
	return fmt.Sprintf("MenuItem[%d, %q]", item.id, item.title)
}

// Run initializes GUI and starts the event loop, then invokes the onReady
// callback. It blocks until systray.Quit() is called.
func Run(onReady func(), onExit func()) {
	Register(onReady, onExit)
	native_loop()
}

// Register initializes GUI and registers the callbacks but relies on the
// caller to run the event loop somewhere else. It's useful if the program
// needs to show other UI elements, for example, webview.
// To overcome some OS weirdness, On macOS versions before Catalina, calling
// this does exactly the same as Run().
func Register(onReady func(), onExit func()) {
	if onReady == nil {
		systrayReady = func() {}
	} else {
		// Run onReady on separate goroutine to avoid blocking event loop
		readyCh := make(chan interface{})
		go func() {
			<-readyCh
			onReady()
		}()
		systrayReady = func() {
			close(readyCh)
		}
	}
	// unlike onReady, onExit runs in the event loop to make sure it has time to
	// finish before the process terminates
	if onExit == nil {
		onExit = func() {}
	}
	systrayExit = onExit
	register_systray()
}

// Quit the systray
func Quit() {
	quitOnce.Do(quit)
}

// AddMenuItem appends the item to the parent menu list.
func AddMenuItem(parent, item *MenuItem) {
	globalRegistry.Store(item)
	if item.typ == menuItemTypeSep {
		addSeparator(item.id)
		return
	}
	if parent == nil {
		item.insert(0, 0)
	} else {
		item.insert(parent.id, 0)
		parent.children = append(parent.children, item)
	}
}

// AddMenuItemBefore adds the item to the parent menu list before the specified next item.
func AddMenuItemBefore(parent, item *MenuItem, next *MenuItem) {
	globalRegistry.Store(item)
	if parent == nil {
		item.insert(0, next.id)
	} else {
		item.insert(parent.id, next.id)
		parent.children = append(parent.children, item)
	}
}

func systrayMenuItemSelected(id uint32) {
	item := globalRegistry.MenuItem(id)
	if item == nil {
		zap.L().Error("No menu item with ID %v", zap.Uint32("id", id))
		return
	}
	select {
	case globalRegistry.events <- item:
	// in case no one waiting for the channel
	default:
		zap.L().Error("Event dropped", zap.Stringer("item", item))
	}
}
