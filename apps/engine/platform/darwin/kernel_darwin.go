//go:build darwin && cgo

package darwin

/*
#cgo LDFLAGS: -framework IOKit -framework CoreFoundation
#include <stdlib.h>
#include <string.h>
#include <time.h>
#include <mach/mach.h>
#include <mach/mach_time.h>
#include <sys/sysctl.h>
#include <sys/socket.h>
#include <net/if.h>
#include <net/if_mib.h>
#include <libproc.h>
#include <CoreFoundation/CoreFoundation.h>
#include <IOKit/IOKitLib.h>
#include <IOKit/storage/IOBlockStorageDriver.h>
#include <IOKit/storage/IOStorageProtocolCharacteristics.h>

typedef struct { uint64_t user, system, idle, nice; } station_ticks_t;

static int station_cpu_ticks(station_ticks_t *out, int max) {
	natural_t count = 0;
	processor_info_array_t info;
	mach_msg_type_number_t size = 0;
	if (host_processor_info(mach_host_self(), PROCESSOR_CPU_LOAD_INFO, &count, &info, &size) != KERN_SUCCESS) return -1;
	int n = (int)count < max ? (int)count : max;
	for (int i = 0; i < n; i++) {
		unsigned int *t = (unsigned int *)info + CPU_STATE_MAX * i;
		out[i].user = t[CPU_STATE_USER];
		out[i].system = t[CPU_STATE_SYSTEM];
		out[i].idle = t[CPU_STATE_IDLE];
		out[i].nice = t[CPU_STATE_NICE];
	}
	vm_deallocate(mach_task_self(), (vm_address_t)info, size * sizeof(integer_t));
	return n;
}

typedef struct { uint64_t size, internal, purgeable, wired, compressor; } station_pages_t;

static int station_pages(station_pages_t *out) {
	vm_statistics64_data_t vm;
	mach_msg_type_number_t count = HOST_VM_INFO64_COUNT;
	if (host_statistics64(mach_host_self(), HOST_VM_INFO64, (host_info64_t)&vm, &count) != KERN_SUCCESS) return -1;
	vm_size_t page = 0;
	if (host_page_size(mach_host_self(), &page) != KERN_SUCCESS) return -1;
	out->size = page;
	out->internal = vm.internal_page_count;
	out->purgeable = vm.purgeable_count;
	out->wired = vm.wire_count;
	out->compressor = vm.compressor_page_count;
	return 0;
}

static int station_load(double *out) { return getloadavg(out, 3); }

typedef struct {
	uint64_t swap_used, memsize;
	int pressure;
	int64_t boot_sec, boot_usec;
} station_memory_t;

static void station_memory(station_memory_t *out) {
	memset(out, 0, sizeof *out);
	struct xsw_usage swap;
	size_t n = sizeof swap;
	if (sysctlbyname("vm.swapusage", &swap, &n, NULL, 0) == 0) out->swap_used = swap.xsu_used;
	uint32_t level = 0;
	n = sizeof level;
	if (sysctlbyname("kern.memorystatus_vm_pressure_level", &level, &n, NULL, 0) == 0) out->pressure = (int)level;
	uint64_t mem = 0;
	n = sizeof mem;
	if (sysctlbyname("hw.memsize", &mem, &n, NULL, 0) == 0) out->memsize = mem;
	struct timeval boot;
	n = sizeof boot;
	if (sysctlbyname("kern.boottime", &boot, &n, NULL, 0) == 0) {
		out->boot_sec = boot.tv_sec;
		out->boot_usec = boot.tv_usec;
	}
}

typedef struct {
	char media[64];
	char interconnect[64];
	char location[64];
	uint64_t read, written;
} station_driver_t;

static void station_copy(CFDictionaryRef d, CFStringRef key, char *out, size_t n) {
	out[0] = 0;
	if (!d) return;
	CFTypeRef v = CFDictionaryGetValue(d, key);
	if (v && CFGetTypeID(v) == CFStringGetTypeID()) CFStringGetCString((CFStringRef)v, out, n, kCFStringEncodingUTF8);
}

static uint64_t station_number(CFDictionaryRef d, CFStringRef key) {
	int64_t n = 0;
	if (!d) return 0;
	CFTypeRef v = CFDictionaryGetValue(d, key);
	if (v && CFGetTypeID(v) == CFNumberGetTypeID()) CFNumberGetValue((CFNumberRef)v, kCFNumberSInt64Type, &n);
	return (uint64_t)n;
}

static int station_drivers(station_driver_t *out, int max) {
	io_iterator_t it;
	if (IOServiceGetMatchingServices(kIOMainPortDefault, IOServiceMatching(kIOBlockStorageDriverClass), &it) != KERN_SUCCESS) return -1;
	int n = 0;
	io_registry_entry_t driver;
	while ((driver = IOIteratorNext(it)) && n < max) {
		station_driver_t *d = &out[n];
		memset(d, 0, sizeof *d);
		CFDictionaryRef stats = IORegistryEntryCreateCFProperty(driver, CFSTR(kIOBlockStorageDriverStatisticsKey), kCFAllocatorDefault, 0);
		if (stats) {
			d->read = station_number(stats, CFSTR(kIOBlockStorageDriverStatisticsBytesReadKey));
			d->written = station_number(stats, CFSTR(kIOBlockStorageDriverStatisticsBytesWrittenKey));
			CFRelease(stats);
		}
		io_registry_entry_t device;
		if (IORegistryEntryGetParentEntry(driver, kIOServicePlane, &device) == KERN_SUCCESS) {
			CFDictionaryRef proto = IORegistryEntryCreateCFProperty(device, CFSTR(kIOPropertyProtocolCharacteristicsKey), kCFAllocatorDefault, 0);
			if (proto) {
				station_copy(proto, CFSTR(kIOPropertyPhysicalInterconnectTypeKey), d->interconnect, sizeof d->interconnect);
				station_copy(proto, CFSTR(kIOPropertyPhysicalInterconnectLocationKey), d->location, sizeof d->location);
				CFRelease(proto);
			}
			IOObjectRelease(device);
		}
		io_registry_entry_t media;
		if (IORegistryEntryGetChildEntry(driver, kIOServicePlane, &media) == KERN_SUCCESS) {
			CFTypeRef bsd = IORegistryEntryCreateCFProperty(media, CFSTR("BSD Name"), kCFAllocatorDefault, 0);
			if (bsd) {
				if (CFGetTypeID(bsd) == CFStringGetTypeID()) CFStringGetCString((CFStringRef)bsd, d->media, sizeof d->media, kCFStringEncodingUTF8);
				CFRelease(bsd);
			}
			IOObjectRelease(media);
		}
		IOObjectRelease(driver);
		n++;
	}
	IOObjectRelease(it);
	return n;
}

typedef struct {
	char name[IF_NAMESIZE];
	uint64_t in, out;
} station_link_t;

// outside: the routing table's RTM_IFINFO2 counters reach a process without the
// network entitlement cut to 32 bits and rounded to the KiB, so a link past
// 4 GiB wraps; the interface MIB answers every byte.
static int station_links(station_link_t *out, int max) {
	struct if_nameindex *names = if_nameindex();
	if (!names) return -1;
	int n = 0;
	for (struct if_nameindex *i = names; i->if_index != 0 && n < max; i++) {
		int mib[6] = {CTL_NET, PF_LINK, NETLINK_GENERIC, IFMIB_IFDATA, (int)i->if_index, IFDATA_GENERAL};
		struct ifmibdata d;
		size_t len = sizeof d;
		if (sysctl(mib, 6, &d, &len, NULL, 0) != 0) continue;
		strlcpy(out[n].name, i->if_name, sizeof out[n].name);
		out[n].in = d.ifmd_data.ifi_ibytes;
		out[n].out = d.ifmd_data.ifi_obytes;
		n++;
	}
	if_freenameindex(names);
	return n;
}

typedef struct {
	int pid;
	uint64_t cpu, resident;
	char name[2 * MAXCOMLEN + 1];
} station_proc_t;

static uint64_t station_ns(uint64_t ticks) {
	static mach_timebase_info_data_t base;
	if (base.denom == 0) mach_timebase_info(&base);
	return ticks * base.numer / base.denom;
}

static uint64_t station_now(void) { return clock_gettime_nsec_np(CLOCK_UPTIME_RAW); }

static int station_proc(int pid, station_proc_t *out) {
	struct proc_taskinfo info;
	if (proc_pidinfo(pid, PROC_PIDTASKINFO, 0, &info, sizeof info) != (int)sizeof info) return -1;
	out->pid = pid;
	out->cpu = station_ns(info.pti_total_user + info.pti_total_system);
	out->resident = info.pti_resident_size;
	out->name[0] = 0;
	proc_name(pid, out->name, sizeof out->name);
	return 0;
}

static int station_started(int pid, uint64_t *sec, uint64_t *usec) {
	struct proc_bsdinfo info;
	if (proc_pidinfo(pid, PROC_PIDTBSDINFO, 0, &info, sizeof info) != (int)sizeof info) return -1;
	*sec = info.pbi_start_tvsec;
	*usec = info.pbi_start_tvusec;
	return 0;
}

static station_proc_t *station_procs(int *count) {
	*count = 0;
	int want = proc_listallpids(NULL, 0);
	if (want <= 0) return NULL;
	want += 64;
	pid_t *pids = calloc(want, sizeof(pid_t));
	if (!pids) return NULL;
	int got = proc_listallpids(pids, want * sizeof(pid_t));
	if (got <= 0) { free(pids); return NULL; }
	station_proc_t *out = calloc(got, sizeof(station_proc_t));
	if (!out) { free(pids); return NULL; }
	int n = 0;
	for (int i = 0; i < got; i++) {
		if (pids[i] > 0 && station_proc(pids[i], &out[n]) == 0) n++;
	}
	free(pids);
	*count = n;
	return out;
}

static station_proc_t station_proc_at(station_proc_t *list, int i) { return list[i]; }

static char *station_files(int pid, int *used) {
	*used = -1;
	int want = proc_pidinfo(pid, PROC_PIDLISTFDS, 0, NULL, 0);
	if (want <= 0) return NULL;
	struct proc_fdinfo *fds = malloc(want);
	if (!fds) return NULL;
	int got = proc_pidinfo(pid, PROC_PIDLISTFDS, 0, fds, want);
	if (got <= 0) { free(fds); return NULL; }
	int count = got / (int)sizeof *fds;
	char *out = malloc((size_t)count * (MAXPATHLEN + 1) + 1);
	if (!out) { free(fds); return NULL; }
	*used = 0;
	for (int i = 0; i < count; i++) {
		if (fds[i].proc_fdtype != PROX_FDTYPE_VNODE) continue;
		struct vnode_fdinfowithpath v;
		if (proc_pidfdinfo(pid, fds[i].proc_fd, PROC_PIDFDVNODEPATHINFO, &v, sizeof v) != (int)sizeof v) continue;
		int n = (int)strnlen(v.pvip.vip_path, MAXPATHLEN);
		if (n == 0) continue;
		memcpy(out + *used, v.pvip.vip_path, n);
		out[*used + n] = '\n';
		*used += n + 1;
	}
	free(fds);
	return out;
}

static void station_free_files(char *files) { free(files); }

static void station_free(station_proc_t *list) { free(list); }
*/
import "C"

import (
	"errors"
	"strings"
	"time"
)

const (
	maxCPUs    = 256
	maxDrivers = 64
	maxLinks   = 128
)

var errUnread = errors.New("the kernel did not answer")

type Kernel struct{}

func (Kernel) Counters() (Counters, error) {
	var c Counters
	ticks := make([]C.station_ticks_t, maxCPUs)
	n := int(C.station_cpu_ticks(&ticks[0], maxCPUs))
	if n <= 0 {
		return Counters{}, errUnread
	}
	for _, t := range ticks[:n] {
		c.CPUs = append(c.CPUs, Ticks{User: uint64(t.user), System: uint64(t.system), Idle: uint64(t.idle), Nice: uint64(t.nice)})
	}
	var load [3]C.double
	if C.station_load(&load[0]) == 3 {
		c.Load = [3]float64{float64(load[0]), float64(load[1]), float64(load[2])}
	}
	var pages C.station_pages_t
	if C.station_pages(&pages) == 0 {
		c.Pages = Pages{Size: uint64(pages.size), Internal: uint64(pages.internal), Purgeable: uint64(pages.purgeable), Wired: uint64(pages.wired), Compressor: uint64(pages.compressor)}
	}
	var mem C.station_memory_t
	C.station_memory(&mem)
	c.SwapUsed, c.Memsize, c.Pressure = uint64(mem.swap_used), uint64(mem.memsize), int(mem.pressure)
	if mem.boot_sec > 0 {
		c.Up = time.Since(time.Unix(int64(mem.boot_sec), int64(mem.boot_usec)*1000))
	}
	drivers := make([]C.station_driver_t, maxDrivers)
	for _, d := range drivers[:max(int(C.station_drivers(&drivers[0], maxDrivers)), 0)] {
		c.Disks = append(c.Disks, BlockDriver{
			Media: C.GoString(&d.media[0]), Interconnect: C.GoString(&d.interconnect[0]), Location: C.GoString(&d.location[0]),
			Read: uint64(d.read), Written: uint64(d.written),
		})
	}
	links := make([]C.station_link_t, maxLinks)
	for _, l := range links[:max(int(C.station_links(&links[0], maxLinks)), 0)] {
		c.Links = append(c.Links, Link{Name: C.GoString(&l.name[0]), In: uint64(l.in), Out: uint64(l.out)})
	}
	return c, nil
}

func (Kernel) Processes() (Procs, error) {
	var n C.int
	list := C.station_procs(&n)
	if list == nil {
		return Procs{}, errUnread
	}
	defer C.station_free(list)
	out := Procs{At: time.Duration(C.station_now())}
	for i := range int(n) {
		out.List = append(out.List, proc(C.station_proc_at(list, C.int(i))))
	}
	return out, nil
}

func (Kernel) Process(pid int) (Procs, error) {
	var p C.station_proc_t
	if C.station_proc(C.int(pid), &p) != 0 {
		return Procs{}, errUnread
	}
	one := proc(p)
	var sec, usec C.uint64_t
	if C.station_started(C.int(pid), &sec, &usec) == 0 && sec > 0 {
		one.Started = time.Unix(int64(sec), int64(usec)*1000).UTC()
	}
	return Procs{At: time.Duration(C.station_now()), List: []Proc{one}}, nil
}

func (Kernel) Files(pid int) ([]string, error) {
	var n C.int
	files := C.station_files(C.int(pid), &n)
	if n < 0 {
		return nil, errUnread
	}
	defer C.station_free_files(files)
	if n == 0 {
		return []string{}, nil
	}
	return strings.Split(strings.TrimSuffix(C.GoStringN(files, n), "\n"), "\n"), nil
}

func proc(p C.station_proc_t) Proc {
	return Proc{PID: int(p.pid), Name: C.GoString(&p.name[0]), CPU: time.Duration(p.cpu), Resident: uint64(p.resident)}
}
