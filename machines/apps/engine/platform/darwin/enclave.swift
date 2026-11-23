import CryptoKit
import Foundation
import LocalAuthentication
import Security

// invariant: every call returns 0 with its bytes in *out, 1 with a worded
// failure in *out, or 2 when this Mac has no Secure Enclave; *out is malloc'd
// and the caller frees it.

private var session: LAContext?

private func give(_ bytes: Data, _ out: UnsafeMutablePointer<UnsafeMutablePointer<UInt8>?>, _ count: UnsafeMutablePointer<Int>) {
    let buffer = malloc(max(bytes.count, 1))!.assumingMemoryBound(to: UInt8.self)
    bytes.copyBytes(to: buffer, count: bytes.count)
    out.pointee = buffer
    count.pointee = bytes.count
}

private func refuse(_ why: String, _ code: Int32, _ out: UnsafeMutablePointer<UnsafeMutablePointer<UInt8>?>, _ count: UnsafeMutablePointer<Int>) -> Int32 {
    give(Data(why.utf8), out, count)
    return code
}

// invariant: a key made unattended signs with nobody present, from the Mac's
// first unlock after its start on; any other key signs only for an owner
// present at an unlocked Mac. The Enclave fixes this when it makes the key.
private func protection(_ unattended: Int32) -> (CFString, SecAccessControlCreateFlags) {
    unattended != 0
        ? (kSecAttrAccessibleAfterFirstUnlockThisDeviceOnly, [.privateKeyUsage])
        : (kSecAttrAccessibleWhenUnlockedThisDeviceOnly, [.privateKeyUsage, .userPresence])
}

@_cdecl("station_enclave_access")
public func stationEnclaveAccess(_ unattended: Int32, _ flags: UnsafeMutablePointer<UInt>, _ out: UnsafeMutablePointer<UnsafeMutablePointer<UInt8>?>, _ count: UnsafeMutablePointer<Int>) -> Int32 {
    let (accessible, options) = protection(unattended)
    flags.pointee = options.rawValue
    give(Data((accessible as String).utf8), out, count)
    return 0
}

@_cdecl("station_enclave_create")
public func stationEnclaveCreate(_ unattended: Int32, _ out: UnsafeMutablePointer<UnsafeMutablePointer<UInt8>?>, _ count: UnsafeMutablePointer<Int>) -> Int32 {
    guard SecureEnclave.isAvailable else { return refuse("this Mac has no Secure Enclave", 2, out, count) }
    let (accessible, options) = protection(unattended)
    var failure: Unmanaged<CFError>?
    guard let access = SecAccessControlCreateWithFlags(nil, accessible, options, &failure) else {
        return refuse(failure.map { "\($0.takeRetainedValue())" } ?? "no access control", 1, out, count)
    }
    do {
        give(try SecureEnclave.P256.Signing.PrivateKey(accessControl: access).dataRepresentation, out, count)
        return 0
    } catch {
        return refuse("the Secure Enclave made no key: \(error.localizedDescription)", 1, out, count)
    }
}

@_cdecl("station_enclave_public")
public func stationEnclavePublic(_ blob: UnsafePointer<UInt8>, _ blobCount: Int, _ out: UnsafeMutablePointer<UnsafeMutablePointer<UInt8>?>, _ count: UnsafeMutablePointer<Int>) -> Int32 {
    guard SecureEnclave.isAvailable else { return refuse("this Mac has no Secure Enclave", 2, out, count) }
    do {
        give(try SecureEnclave.P256.Signing.PrivateKey(dataRepresentation: Data(bytes: blob, count: blobCount)).publicKey.x963Representation, out, count)
        return 0
    } catch {
        return refuse("this Mac's Secure Enclave refuses the device key: \(error.localizedDescription)", 1, out, count)
    }
}

// invariant: a blob this Mac's Enclave did not wrap is refused before any
// presence is asked. An unattended key signs with no presence asked. For any
// other, the owner's presence is asked once, on the first signature, and
// no answer within the time given is a refusal; the evaluated context then
// opens every later use of the key for the life of this process, so a
// session asks once.
@_cdecl("station_enclave_sign")
public func stationEnclaveSign(_ blob: UnsafePointer<UInt8>, _ blobCount: Int, _ unattended: Int32, _ reason: UnsafePointer<CChar>, _ within: Double, _ message: UnsafePointer<UInt8>, _ messageCount: Int, _ out: UnsafeMutablePointer<UnsafeMutablePointer<UInt8>?>, _ count: UnsafeMutablePointer<Int>) -> Int32 {
    guard SecureEnclave.isAvailable else { return refuse("this Mac has no Secure Enclave", 2, out, count) }
    let wrapped = Data(bytes: blob, count: blobCount)
    let opened: SecureEnclave.P256.Signing.PrivateKey
    do {
        opened = try SecureEnclave.P256.Signing.PrivateKey(dataRepresentation: wrapped)
    } catch {
        return refuse("this Mac's Secure Enclave refuses the device key: \(error.localizedDescription)", 1, out, count)
    }
    if unattended != 0 {
        do {
            give(try opened.signature(for: Data(bytes: message, count: messageCount)).derRepresentation, out, count)
            return 0
        } catch {
            return refuse("the Secure Enclave did not sign: \(error.localizedDescription)", 1, out, count)
        }
    }
    if session == nil {
        let context = LAContext()
        let done = DispatchSemaphore(value: 0)
        var given = false
        context.evaluatePolicy(.deviceOwnerAuthentication, localizedReason: String(cString: reason)) { yes, _ in
            given = yes
            done.signal()
        }
        if done.wait(timeout: .now() + within) == .timedOut {
            context.invalidate()
            return refuse("the owner's presence was not given in time", 1, out, count)
        }
        guard given else { return refuse("the owner's presence was not given", 1, out, count) }
        session = context
    }
    do {
        let key = try SecureEnclave.P256.Signing.PrivateKey(dataRepresentation: wrapped, authenticationContext: session)
        give(try key.signature(for: Data(bytes: message, count: messageCount)).derRepresentation, out, count)
        return 0
    } catch {
        return refuse("the Secure Enclave did not sign: \(error.localizedDescription)", 1, out, count)
    }
}
