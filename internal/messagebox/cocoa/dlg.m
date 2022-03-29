#import <Cocoa/Cocoa.h>
#include "dlg.h"

void* NSStr(void* buf, int len) {
	return (void*)[[NSString alloc] initWithBytes:buf length:len encoding:NSUTF8StringEncoding];
}

void NSRelease(void* obj) {
	[(NSObject*)obj release];
}

@interface AlertDlg : NSObject {
	AlertDlgParams* params;
	DlgResult result;
}
+ (AlertDlg*)init:(AlertDlgParams*)params;
- (DlgResult)run;
@end

DlgResult alertDlg(AlertDlgParams* params) {
	return [[AlertDlg init:params] run];
}

@implementation AlertDlg
+ (AlertDlg*)init:(AlertDlgParams*)params {
	AlertDlg* d = [AlertDlg alloc];
	d->params = params;
	return d;
}

- (DlgResult)run {
	if(![NSThread isMainThread]) {
		[self performSelectorOnMainThread:@selector(run) withObject:nil waitUntilDone:YES];
		return self->result;
	}
	NSAlert* alert = [[NSAlert alloc] init];
    [[alert window] setTitle:[[NSString alloc] initWithUTF8String:self->params->title]];
	[alert setMessageText:[[NSString alloc] initWithUTF8String:self->params->msg]];
	switch (self->params->style) {
	case MSG_YESNO:
		[alert addButtonWithTitle:@"Yes"];
		[alert addButtonWithTitle:@"No"];
		break;
	case MSG_ERROR:
		[alert setIcon:[NSImage imageNamed:NSImageNameCaution]];
		[alert addButtonWithTitle:@"OK"];
		break;
	case MSG_INFO:
		[alert setIcon:[NSImage imageNamed:NSImageNameInfo]];
		[alert addButtonWithTitle:@"OK"];
		break;
	}
	self->result = [alert runModal] == NSAlertFirstButtonReturn ? DLG_OK : DLG_CANCEL;
	return self->result;
}
@end
