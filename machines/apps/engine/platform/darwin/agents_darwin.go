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

static char *station_login_item(const char *identifier, int on) {
	@autoreleasepool {
		SMAppService *service = [SMAppService loginItemServiceWithIdentifier:[NSString stringWithUTF8String:identifier]];
		NSError *err = nil;
		BOOL ok = on ? [service registerAndReturnError:&err] : [service unregisterAndReturnError:&err];
		if (ok) {
			return NULL;
		}
		NSString *why = [NSString stringWithFormat:@"%@ %ld: %@", err.domain, (long)err.code, err.localizedDescription];
		return strdup(why.UTF8String);
	}
}

static int station_agent_enabled(const char *plist) {
	@autoreleasepool {
		SMAppService *service = [SMAppService agentServiceWithPlistName:[NSString stringWithUTF8String:plist]];
		return service.status == SMAppServiceStatusEnabled;
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

func (Agents) Registered(plist string) bool {
	name := C.CString(plist)
	defer C.free(unsafe.Pointer(name))
	return C.station_agent_enabled(name) != 0
}

// value: registers the login item of the app whose bundle holds this
// binary, named by its bundle identifier in Contents/Library/LoginItems,
// which macOS starts now and at every login.
func (Agents) ShowLoginItem(identifier string) error { return loginItem(identifier, 1) }

// value: unregisters that login item, which macOS stops now and starts at no
// later login.
func (Agents) HideLoginItem(identifier string) error { return loginItem(identifier, 0) }

func loginItem(identifier string, on C.int) error {
	name := C.CString(identifier)
	defer C.free(unsafe.Pointer(name))
	why := C.station_login_item(name, on)
	if why == nil {
		return nil
	}
	defer C.free(unsafe.Pointer(why))
	return errors.New(C.GoString(why))
}

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
