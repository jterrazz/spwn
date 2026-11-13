package catalog

// value: what the procedures write or point at outside this machine: the
// profile of the dns setting, and the pages of GitHub where a new key is
// added.
const (
	DNSProfileID  = "com.jterrazz.dns.quad9"
	GitHubDomain  = "github.com"
	GitHubSSHKeys = "https://github.com/settings/ssh/new"
	GitHubGPGKeys = "https://github.com/settings/gpg/new"
	// value: the openclaw script the lock-after-login agent runs.
	LockAfterLoginScript = "/Users/jterrazz.agent/.openclaw/scripts/lock-after-login.sh"
)

const DNSProfile = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>PayloadContent</key>
	<array>
		<dict>
			<key>DNSSettings</key>
			<dict>
				<key>DNSProtocol</key>
				<string>HTTPS</string>
				<key>ServerAddresses</key>
				<array>
					<string>2620:fe::fe</string>
					<string>2620:fe::9</string>
					<string>9.9.9.9</string>
					<string>149.112.112.112</string>
				</array>
				<key>ServerURL</key>
				<string>https://dns.quad9.net/dns-query</string>
			</dict>
			<key>PayloadDescription</key>
			<string>Configures device to use Quad9 Encrypted DNS over HTTPS</string>
			<key>PayloadDisplayName</key>
			<string>Quad9 DNS over HTTPS</string>
			<key>PayloadIdentifier</key>
			<string>com.jterrazz.dns.quad9.doh</string>
			<key>PayloadType</key>
			<string>com.apple.dnsSettings.managed</string>
			<key>PayloadUUID</key>
			<string>B9C4F3A2-5E6D-7F8A-9B0C-1D2E3F4A5B6C</string>
			<key>PayloadVersion</key>
			<integer>1</integer>
			<key>ProhibitDisablement</key>
			<false/>
		</dict>
	</array>
	<key>PayloadDescription</key>
	<string>Configures encrypted DNS over HTTPS using Quad9 (9.9.9.9)</string>
	<key>PayloadDisplayName</key>
	<string>Quad9 Encrypted DNS</string>
	<key>PayloadIdentifier</key>
	<string>com.jterrazz.dns.quad9</string>
	<key>PayloadRemovalDisallowed</key>
	<false/>
	<key>PayloadScope</key>
	<string>System</string>
	<key>PayloadType</key>
	<string>Configuration</string>
	<key>PayloadUUID</key>
	<string>A8B3E2F1-4D5C-6E7F-8A9B-0C1D2E3F4A5B</string>
	<key>PayloadVersion</key>
	<integer>1</integer>
</dict>
</plist>`
