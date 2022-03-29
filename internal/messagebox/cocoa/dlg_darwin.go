package cocoa

// #cgo darwin LDFLAGS: -framework Cocoa
// #include <stdlib.h>
// #include <sys/syslimits.h>
// #include "dlg.h"
import "C"

import (
	"unsafe"
)

type AlertParams struct {
	p C.AlertDlgParams
}

func mkAlertParams(title, content string, style C.AlertStyle) *AlertParams {
	a := AlertParams{C.AlertDlgParams{title: C.CString(title), msg: C.CString(content), style: style}}
	return &a
}

func (a *AlertParams) run() C.DlgResult {
	return C.alertDlg(&a.p)
}

func (a *AlertParams) free() {
	C.free(unsafe.Pointer(a.p.msg))
	C.free(unsafe.Pointer(a.p.title))
}

func Confirm(title, content string) bool {
	a := mkAlertParams(title, content, C.MSG_YESNO)
	defer a.free()
	return a.run() == C.DLG_OK
}

func Info(title, content string) {
	a := mkAlertParams(title, content, C.MSG_INFO)
	defer a.free()
	a.run()
}

func Error(title, content string) {
	a := mkAlertParams(title, content, C.MSG_ERROR)
	defer a.free()
	a.run()
}
