package objc

/*
   #cgo darwin CFLAGS: -DDARWIN -x objective-c -fobjc-arc
   #cgo darwin LDFLAGS: -framework Cocoa
   #include "app.h"
*/
import "C"

func RunNativeApp(path string) {
	C.RunNativeApp(C.CString(path))
}
