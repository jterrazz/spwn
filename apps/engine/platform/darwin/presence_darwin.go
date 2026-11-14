//go:build darwin && cgo

package darwin

/*
#cgo CFLAGS: -x objective-c -fobjc-arc
#cgo LDFLAGS: -framework LocalAuthentication -framework Foundation
#include <stdlib.h>
#import <Foundation/Foundation.h>
#import <LocalAuthentication/LocalAuthentication.h>

static int station_presence(const char *reason, double within) {
	@autoreleasepool {
		LAContext *context = [[LAContext alloc] init];
		NSError *why = nil;
		if (![context canEvaluatePolicy:LAPolicyDeviceOwnerAuthentication error:&why]) return 2;
		dispatch_semaphore_t done = dispatch_semaphore_create(0);
		__block int answer = 1;
		[context evaluatePolicy:LAPolicyDeviceOwnerAuthentication localizedReason:[NSString stringWithUTF8String:reason] reply:^(BOOL ok, NSError *e) {
			if (ok) answer = 0;
			else if (e.code == LAErrorNotInteractive || e.code == LAErrorPasscodeNotSet) answer = 2;
			dispatch_semaphore_signal(done);
		}];
		if (dispatch_semaphore_wait(done, dispatch_time(DISPATCH_TIME_NOW, (int64_t)(within * NSEC_PER_SEC))) != 0) {
			[context invalidate];
			return 1;
		}
		return answer;
	}
}
*/
import "C"

import (
	"time"
	"unsafe"
)

// invariant: the owner's presence is asked of LocalAuthentication with
// station's own reason, Touch ID or else the login password; no answer
// within the time given is a refusal, and a Mac that cannot ask (no
// session, no password) is absent.
func AskPresence(reason string, within time.Duration) Presence {
	text := C.CString(reason)
	defer C.free(unsafe.Pointer(text))
	return Presence(C.station_presence(text, C.double(within.Seconds())))
}
