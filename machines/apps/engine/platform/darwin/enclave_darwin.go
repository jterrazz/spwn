//go:build darwin && cgo

package darwin

/*
#cgo CFLAGS: -x objective-c -fobjc-arc
#cgo LDFLAGS: -framework Security -framework LocalAuthentication -framework Foundation
#include <stdlib.h>
#import <Foundation/Foundation.h>
#import <LocalAuthentication/LocalAuthentication.h>
#import <Security/Security.h>

static LAContext *station_session = nil;

static char *station_why(CFErrorRef e, const char *fallback) {
	if (e == NULL) return strdup(fallback);
	NSString *text = [(__bridge NSError *)e localizedDescription];
	CFRelease(e);
	return strdup(text.UTF8String ? text.UTF8String : fallback);
}

static SecKeyRef station_find(NSData *tag, LAContext *context) {
	NSMutableDictionary *q = [@{
		(id)kSecClass: (id)kSecClassKey,
		(id)kSecAttrApplicationTag: tag,
		(id)kSecAttrTokenID: (id)kSecAttrTokenIDSecureEnclave,
		(id)kSecAttrKeyType: (id)kSecAttrKeyTypeECSECPrimeRandom,
		(id)kSecUseDataProtectionKeychain: @YES,
		(id)kSecReturnRef: @YES,
	} mutableCopy];
	if (context != nil) q[(id)kSecUseAuthenticationContext] = context;
	CFTypeRef found = NULL;
	if (SecItemCopyMatching((__bridge CFDictionaryRef)q, &found) != errSecSuccess) return NULL;
	return (SecKeyRef)found;
}

static SecKeyRef station_make(NSData *tag, char **why) {
	CFErrorRef e = NULL;
	SecAccessControlRef acl = SecAccessControlCreateWithFlags(kCFAllocatorDefault, kSecAttrAccessibleWhenUnlockedThisDeviceOnly, kSecAccessControlPrivateKeyUsage | kSecAccessControlUserPresence, &e);
	if (acl == NULL) { *why = station_why(e, "no access control"); return NULL; }
	NSDictionary *attrs = @{
		(id)kSecAttrKeyType: (id)kSecAttrKeyTypeECSECPrimeRandom,
		(id)kSecAttrKeySizeInBits: @256,
		(id)kSecAttrTokenID: (id)kSecAttrTokenIDSecureEnclave,
		(id)kSecUseDataProtectionKeychain: @YES,
		(id)kSecPrivateKeyAttrs: @{
			(id)kSecAttrIsPermanent: @YES,
			(id)kSecAttrApplicationTag: tag,
			(id)kSecAttrAccessControl: (__bridge id)acl,
		},
	};
	SecKeyRef key = SecKeyCreateRandomKey((__bridge CFDictionaryRef)attrs, &e);
	CFRelease(acl);
	if (key == NULL) *why = station_why(e, "the Secure Enclave made no key");
	return key;
}

// invariant: 0 the public point (65 bytes, uncompressed) is in out; 1 a
// failure, worded in why; 2 this Mac has no Secure Enclave.
static int station_enclave_public(const char *label, unsigned char *out, char **why) {
	@autoreleasepool {
		NSData *tag = [NSData dataWithBytes:label length:strlen(label)];
		SecKeyRef key = station_find(tag, nil);
		if (key == NULL) key = station_make(tag, why);
		if (key == NULL) return 2;
		SecKeyRef pub = SecKeyCopyPublicKey(key);
		CFRelease(key);
		if (pub == NULL) { *why = strdup("the key has no public half"); return 1; }
		CFErrorRef e = NULL;
		CFDataRef point = SecKeyCopyExternalRepresentation(pub, &e);
		CFRelease(pub);
		if (point == NULL) { *why = station_why(e, "the public half cannot be read"); return 1; }
		if (CFDataGetLength(point) != 65) { CFRelease(point); *why = strdup("the public half is not a P-256 point"); return 1; }
		memcpy(out, CFDataGetBytePtr(point), 65);
		CFRelease(point);
		return 0;
	}
}

// invariant: the owner's presence is asked once, on the first signature, and
// no answer within the time given is a refusal; the evaluated context then
// opens every later use of the key for the life of this process, so a
// session asks once.
static int station_enclave_sign(const char *label, const char *reason, double within, const unsigned char *digest, int length, unsigned char **sig, int *sigLength, char **why) {
	@autoreleasepool {
		if (station_session == nil) {
			LAContext *context = [[LAContext alloc] init];
			dispatch_semaphore_t done = dispatch_semaphore_create(0);
			__block BOOL ok = NO;
			[context evaluatePolicy:LAPolicyDeviceOwnerAuthentication localizedReason:[NSString stringWithUTF8String:reason] reply:^(BOOL yes, NSError *e) {
				ok = yes;
				dispatch_semaphore_signal(done);
			}];
			if (dispatch_semaphore_wait(done, dispatch_time(DISPATCH_TIME_NOW, (int64_t)(within * NSEC_PER_SEC))) != 0) {
				[context invalidate];
				*why = strdup("the owner's presence was not given in time");
				return 1;
			}
			if (!ok) { *why = strdup("the owner's presence was not given"); return 1; }
			station_session = context;
		}
		NSData *tag = [NSData dataWithBytes:label length:strlen(label)];
		SecKeyRef key = station_find(tag, station_session);
		if (key == NULL) { *why = strdup("this Mac's device key is gone"); return 1; }
		CFErrorRef e = NULL;
		CFDataRef data = CFDataCreate(kCFAllocatorDefault, digest, length);
		CFDataRef out = SecKeyCreateSignature(key, kSecKeyAlgorithmECDSASignatureDigestX962SHA256, data, &e);
		CFRelease(data);
		CFRelease(key);
		if (out == NULL) { *why = station_why(e, "the Secure Enclave did not sign"); return 1; }
		*sigLength = (int)CFDataGetLength(out);
		*sig = malloc(*sigLength);
		memcpy(*sig, CFDataGetBytePtr(out), *sigLength);
		CFRelease(out);
		return 0;
	}
}
*/
import "C"

import (
	"errors"
	"time"
	"unsafe"
)

func enclaveError(why *C.char) error {
	defer C.free(unsafe.Pointer(why))
	return errors.New(C.GoString(why))
}

func EnclavePublic(label string) ([]byte, error) {
	tag := C.CString(label)
	defer C.free(unsafe.Pointer(tag))
	out := make([]byte, 65)
	var why *C.char
	switch C.station_enclave_public(tag, (*C.uchar)(unsafe.Pointer(&out[0])), &why) {
	case 0:
		return out, nil
	case 2:
		return nil, &NoEnclaveError{Why: C.GoString(why)}
	default:
		return nil, enclaveError(why)
	}
}

func EnclaveSign(label, reason string, within time.Duration, digest []byte) ([]byte, error) {
	if len(digest) == 0 {
		return nil, errors.New("nothing to sign")
	}
	tag, text := C.CString(label), C.CString(reason)
	defer C.free(unsafe.Pointer(tag))
	defer C.free(unsafe.Pointer(text))
	var sig *C.uchar
	var n C.int
	var why *C.char
	if C.station_enclave_sign(tag, text, C.double(within.Seconds()), (*C.uchar)(unsafe.Pointer(&digest[0])), C.int(len(digest)), &sig, &n, &why) != 0 {
		return nil, enclaveError(why)
	}
	defer C.free(unsafe.Pointer(sig))
	return C.GoBytes(unsafe.Pointer(sig), n), nil
}
