//go:build darwin && cgo

package darwin

/*
#cgo CFLAGS: -x objective-c -fobjc-arc
#cgo LDFLAGS: -framework ServiceManagement -framework Foundation
#include <stdlib.h>
#import <Foundation/Foundation.h>
#import <ServiceManagement/ServiceManagement.h>

static char *station_agent(const char *plist, int on) {
	@autoreleasepool {
		SMAppService *service = [SMAppService agentServiceWithPlistName:[NSString stringWithUTF8String:plist]];
		NSError *err = nil;
		BOOL ok = on ? [service registerAndReturnError:&err] : [service unregisterAndReturnError:&err];
		if (ok) {
			return NULL;
		}
		NSString *why = [NSString stringWithFormat:@"%@ %ld: %@", err.domain, (long)err.code, err.localizedDescription];
		return strdup(why.UTF8String);
	}
}
*/
import "C"

import (
	"errors"
	"unsafe"
)

type Agents struct {
	// value: SMAppService's agents of the app whose bundle holds this binary
	// (ADR-073, mode A): the calling code is resolved by macOS from the
	// running executable, so a run through a link into the bundle registers
	// the bundle's.
}

func (Agents) Register(plist string) error { return agent(plist, 1) }

func (Agents) Unregister(plist string) error { return agent(plist, 0) }

func agent(plist string, on C.int) error {
	name := C.CString(plist)
	defer C.free(unsafe.Pointer(name))
	why := C.station_agent(name, on)
	if why == nil {
		return nil
	}
	defer C.free(unsafe.Pointer(why))
	return errors.New(C.GoString(why))
}
