package main

/*
   #cgo darwin CFLAGS: -DDARWIN -x objective-c -fobjc-arc
   #cgo darwin LDFLAGS: -framework Cocoa

   #include "stdbool.h"
   void runNativeApp(char* path);
*/
import "C"

func runNativeApp(path string) {
	C.runNativeApp(C.CString(path))
}
