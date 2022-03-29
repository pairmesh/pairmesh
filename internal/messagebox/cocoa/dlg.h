#include <objc/NSObjCRuntime.h>

typedef enum {
	MSG_YESNO,
	MSG_ERROR,
	MSG_INFO,
} AlertStyle;

typedef struct {
	char* msg;
	char* title;
	AlertStyle style;
} AlertDlgParams;

typedef enum {
	DLG_OK,
	DLG_CANCEL,
	DLG_URLFAIL,
} DlgResult;

DlgResult alertDlg(AlertDlgParams*);

void* NSStr(void* buf, int len);
void NSRelease(void* obj);
