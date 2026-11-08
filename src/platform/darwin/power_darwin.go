//go:build darwin && cgo

package darwin

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework IOKit -framework CoreFoundation -framework Foundation
#include <mach/mach.h>
#include <notify.h>
#include <string.h>
#include <IOKit/IOKitLib.h>
#include <IOKit/pwr_mgt/IOPMLib.h>
#include <IOKit/ps/IOPowerSources.h>
#include <IOKit/ps/IOPSKeys.h>
#include <IOKit/IOMessage.h>
#include <pthread.h>
#import <Foundation/Foundation.h>

static int station_hold_idle(_GoString_ name, uint32_t *id) {
	CFStringRef reason = CFStringCreateWithBytes(kCFAllocatorDefault, (const UInt8 *)_GoStringPtr(name), (CFIndex)_GoStringLen(name), kCFStringEncodingUTF8, false);
	IOPMAssertionID assertion = 0;
	IOReturn r = IOPMAssertionCreateWithName(kIOPMAssertPreventUserIdleSystemSleep, kIOPMAssertionLevelOn, reason, &assertion);
	CFRelease(reason);
	*id = assertion;
	return r;
}

static int station_release_idle(uint32_t id) { return IOPMAssertionRelease(id); }

static int station_clamshell(int disabled) {
	io_service_t root = IOServiceGetMatchingService(kIOMainPortDefault, IOServiceMatching("IOPMrootDomain"));
	if (!root) return -1;
	io_connect_t conn = 0;
	kern_return_t kr = IOServiceOpen(root, mach_task_self(), 0, &conn);
	IOObjectRelease(root);
	if (kr != KERN_SUCCESS) return kr;
	uint64_t in = disabled ? 1 : 0;
	kr = IOConnectCallScalarMethod(conn, 12, &in, 1, NULL, NULL);
	IOServiceClose(conn);
	return kr;
}

static int station_lid(void) {
	io_service_t root = IOServiceGetMatchingService(kIOMainPortDefault, IOServiceMatching("IOPMrootDomain"));
	if (!root) return -1;
	CFTypeRef v = IORegistryEntryCreateCFProperty(root, CFSTR("AppleClamshellState"), kCFAllocatorDefault, 0);
	IOObjectRelease(root);
	if (!v) return -1;
	int out = -1;
	if (CFGetTypeID(v) == CFBooleanGetTypeID()) out = CFBooleanGetValue((CFBooleanRef)v) ? 1 : 0;
	CFRelease(v);
	return out;
}

typedef struct {
	int on_battery;
	int has_battery;
	int percent;
	int charging;
} station_sources_t;

static station_sources_t station_sources(void) {
	station_sources_t s = {0, 0, 0, 0};
	CFTypeRef info = IOPSCopyPowerSourcesInfo();
	if (!info) return s;
	CFStringRef type = IOPSGetProvidingPowerSourceType(info);
	if (type && CFStringCompare(type, CFSTR(kIOPMBatteryPowerKey), 0) == kCFCompareEqualTo) s.on_battery = 1;
	CFArrayRef list = IOPSCopyPowerSourcesList(info);
	if (list) {
		for (CFIndex i = 0; i < CFArrayGetCount(list); i++) {
			CFDictionaryRef d = IOPSGetPowerSourceDescription(info, CFArrayGetValueAtIndex(list, i));
			if (!d) continue;
			CFStringRef t = CFDictionaryGetValue(d, CFSTR(kIOPSTypeKey));
			if (!t || CFStringCompare(t, CFSTR(kIOPSInternalBatteryType), 0) != kCFCompareEqualTo) continue;
			int cur = 0, max = 0;
			CFNumberRef n = CFDictionaryGetValue(d, CFSTR(kIOPSCurrentCapacityKey));
			if (n) CFNumberGetValue(n, kCFNumberIntType, &cur);
			n = CFDictionaryGetValue(d, CFSTR(kIOPSMaxCapacityKey));
			if (n) CFNumberGetValue(n, kCFNumberIntType, &max);
			CFBooleanRef c = CFDictionaryGetValue(d, CFSTR(kIOPSIsChargingKey));
			s.has_battery = 1;
			s.percent = max > 0 ? (cur * 100 + max / 2) / max : cur;
			s.charging = c && CFBooleanGetValue(c);
			break;
		}
		CFRelease(list);
	}
	CFRelease(info);
	return s;
}

typedef struct {
	int design;
	int full;
	int cycles;
} station_battery_t;

static int station_number(CFTypeRef v) {
	int out = 0;
	if (v && CFGetTypeID(v) == CFNumberGetTypeID()) CFNumberGetValue((CFNumberRef)v, kCFNumberIntType, &out);
	return out;
}

static int station_int(io_service_t s, CFStringRef key) {
	CFTypeRef v = IORegistryEntryCreateCFProperty(s, key, kCFAllocatorDefault, 0);
	if (!v) return 0;
	int out = station_number(v);
	CFRelease(v);
	return out;
}

static int station_in(CFTypeRef dict, CFStringRef key) {
	if (!dict || CFGetTypeID(dict) != CFDictionaryGetTypeID()) return 0;
	return station_number(CFDictionaryGetValue((CFDictionaryRef)dict, key));
}

static station_battery_t station_battery(void) {
	station_battery_t b = {0, 0, 0};
	io_service_t s = IOServiceGetMatchingService(kIOMainPortDefault, IOServiceMatching("AppleSmartBattery"));
	if (!s) return b;
	CFTypeRef data = IORegistryEntryCreateCFProperty(s, CFSTR("BatteryData"), kCFAllocatorDefault, 0);
	b.design = station_in(data, CFSTR("DesignCapacity"));
	b.full = station_in(data, CFSTR("NominalChargeCapacity"));
	if (data) CFRelease(data);
	if (b.design <= 0 || b.full <= 0) {
		b.design = station_int(s, CFSTR("DesignCapacity"));
		b.full = station_int(s, CFSTR("AppleRawMaxCapacity"));
	}
	b.cycles = station_int(s, CFSTR("CycleCount"));
	IOObjectRelease(s);
	return b;
}

static int station_low_power(void) {
	@autoreleasepool {
		return [[NSProcessInfo processInfo] isLowPowerModeEnabled] ? 1 : 0;
	}
}

static int station_thermal_token(int *token) {
	return notify_register_check("com.apple.system.thermalpressurelevel", token);
}

static int station_notify_fd(_GoString_ name, int *token) {
	char key[256];
	size_t n = _GoStringLen(name);
	if (n >= sizeof key) return -1;
	memcpy(key, _GoStringPtr(name), n);
	key[n] = 0;
	int fd = -1;
	if (notify_register_file_descriptor(key, &fd, 0, token) != NOTIFY_STATUS_OK) return -1;
	return fd;
}

static void station_notify_cancel(int token) { notify_cancel(token); }

static io_connect_t station_root_port = 0;
static int station_sleeping = 0;
static pthread_once_t station_sleep_once = PTHREAD_ONCE_INIT;

static void station_power_message(void *ref, io_service_t service, natural_t type, void *arg) {
	switch (type) {
	case kIOMessageCanSystemSleep:
		IOAllowPowerChange(station_root_port, (long)arg);
		break;
	case kIOMessageSystemWillSleep:
		__atomic_store_n(&station_sleeping, 1, __ATOMIC_SEQ_CST);
		IOAllowPowerChange(station_root_port, (long)arg);
		break;
	case kIOMessageSystemWillPowerOn:
	case kIOMessageSystemHasPoweredOn:
		__atomic_store_n(&station_sleeping, 0, __ATOMIC_SEQ_CST);
		break;
	}
}

static void *station_sleep_loop(void *unused) {
	IONotificationPortRef port = NULL;
	io_object_t notifier = 0;
	station_root_port = IORegisterForSystemPower(NULL, &port, station_power_message, &notifier);
	if (!station_root_port) return NULL;
	CFRunLoopAddSource(CFRunLoopGetCurrent(), IONotificationPortGetRunLoopSource(port), kCFRunLoopDefaultMode);
	CFRunLoopRun();
	return NULL;
}

static void station_sleep_start(void) {
	pthread_t t;
	if (pthread_create(&t, NULL, station_sleep_loop, NULL) == 0) pthread_detach(t);
}

static int station_sleep_announced(void) {
	pthread_once(&station_sleep_once, station_sleep_start);
	return __atomic_load_n(&station_sleeping, __ATOMIC_SEQ_CST);
}

static int station_thermal(int token) {
	uint64_t state = 0;
	if (notify_get_state(token, &state) != NOTIFY_STATUS_OK) return 0;
	return (int)state;
}
*/
import "C"

import (
	"context"
	"errors"
	"os"
	"strconv"
	"sync"
	"syscall"
)

const Supported = true

type Reading struct {
	Lid        string
	Source     string
	HasBattery bool
	Percent    int
	Charging   bool
	LowPower   bool
	Thermal    int
	Health     int
	Cycles     int
}

type Power struct {
	mu      sync.Mutex
	token   C.int
	tokened bool
}

func HoldIdle(name string) (func(), error) {
	var id C.uint32_t
	if r := C.station_hold_idle(name, &id); r != 0 {
		return nil, errors.New("IOPMAssertionCreateWithName returned " + strconv.Itoa(int(r)))
	}
	var once sync.Once
	return func() { once.Do(func() { C.station_release_idle(id) }) }, nil
}

func SetClamshellSleep(disabled bool) error {
	flag := C.int(0)
	if disabled {
		flag = 1
	}
	if r := C.station_clamshell(flag); r != 0 {
		return errors.New("kPMSetClamshellSleepState returned " + strconv.Itoa(int(r)))
	}
	return nil
}

func (p *Power) Read() Reading {
	p.mu.Lock()
	defer p.mu.Unlock()
	var r Reading
	switch C.station_lid() {
	case 0:
		r.Lid = "open"
	case 1:
		r.Lid = "closed"
	}
	s := C.station_sources()
	r.Source = "ac"
	if s.on_battery != 0 {
		r.Source = "battery"
	}
	r.HasBattery, r.Percent, r.Charging = s.has_battery != 0, int(s.percent), s.charging != 0
	if r.HasBattery {
		b := C.station_battery()
		r.Health, r.Cycles = BatteryHealth(int(b.full), int(b.design)), int(b.cycles)
	}
	r.LowPower = C.station_low_power() != 0
	if !p.tokened && C.station_thermal_token(&p.token) == 0 {
		p.tokened = true
	}
	if p.tokened {
		r.Thermal = int(C.station_thermal(p.token))
	}
	return r
}

func SleepAnnounced() bool {
	// invariant: kIOMessageSystemWillSleep is heard on a thread of C's own
	// with its own run loop, which only sets a flag this side reads: no
	// callback crosses into Go; the first call starts listening, and a wake
	// clears it.
	return C.station_sleep_announced() != 0
}

func PowerSourcesChanged(ctx context.Context) <-chan struct{} {
	return notified(ctx, C.kIOPSNotifyAnyPowerSource)
}

// invariant: notify(3) writes each post of name on a descriptor this side
// reads, so no callback ever crosses from C into Go; the channel never fires
// when the name cannot be registered, and the poll alone remains.
func notified(ctx context.Context, name string) <-chan struct{} {
	wake := make(chan struct{}, 1)
	var token C.int
	fd := C.station_notify_fd(name, &token)
	if fd < 0 {
		return wake
	}
	// value: a duplicate this side owns and closes, so notify_cancel closing
	// its own descriptor never races the read.
	own, err := syscall.Dup(int(fd))
	if err != nil {
		C.station_notify_cancel(token)
		return wake
	}
	if err := syscall.SetNonblock(own, true); err != nil {
		_ = syscall.Close(own)
		C.station_notify_cancel(token)
		return wake
	}
	f := os.NewFile(uintptr(own), name)
	go func() {
		<-ctx.Done()
		C.station_notify_cancel(token)
		_ = f.Close()
	}()
	go func() {
		buf := make([]byte, 4)
		for {
			if _, err := f.Read(buf); err != nil {
				return
			}
			select {
			case wake <- struct{}{}:
			default:
			}
		}
	}()
	return wake
}
