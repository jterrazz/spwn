import ServiceManagement

enum LoginItem {
    static var status: String {
        switch SMAppService.mainApp.status {
        case .enabled: "enabled"
        case .notRegistered: "notRegistered"
        case .requiresApproval: "requiresApproval"
        case .notFound: "notFound"
        @unknown default: "unknown"
        }
    }

    static var on: Bool { SMAppService.mainApp.status == .enabled }

    static func set(_ on: Bool) throws {
        if on {
            try SMAppService.mainApp.register()
        } else {
            try SMAppService.mainApp.unregister()
        }
    }

    static func unregister() throws { try SMAppService.mainApp.unregister() }
}
