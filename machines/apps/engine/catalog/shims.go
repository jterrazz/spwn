package catalog

type Shims struct {
	// value: the system folder macOS keeps them in.
	Dir string
	// value: the name of each command, as xcode-select(1)'s FILES section
	// lists it.
	Command []string
}

func XcodeShims() Shims {
	// value: the shims xcode-select manages: run with no developer directory
	// installed, each opens macOS's dialog offering to install the command
	// line developer package instead of running.
	return Shims{Dir: "/usr/bin", Command: []string{
		"xcrun", "actool", "agvtool", "desdp", "genstrings", "ibtool", "ictool", "opendiff", "pip3", "python3", "sdef",
		"sdp", "stapler", "xcodebuild", "xcscontrol", "xcsdiagnose", "xctrace", "xed",
		"DeRez", "GetFileInfo", "ResMerger", "Rez", "SetFile", "SplitForks", "ar", "as", "asa", "bm4", "bison", "c89",
		"c99", "clang++", "clang", "clangd", "cmpdylib", "codesign_allocate", "cpp", "ctags", "ctf_insert", "dsymutil",
		"dwarfdump", "dyld_info", "flex++", "flex", "g++", "gatherheaderdoc", "gcc", "gcov", "git-receive-pack",
		"git-shell", "git-upload-archive", "git-upload-pack", "git", "gm4", "gnumake", "gperf", "hdxml2manxml",
		"headerdoc2html", "indent", "install_name_tool", "ld", "lex", "libtool", "lipo", "lldb", "lorder", "m4", "make",
		"mig", "nm", "nmedit", "objdump", "otool", "pagestuff", "ranlib", "resolveLinks", "rpcgen", "segedit", "size",
		"strings", "strip", "swift", "swiftc", "unifdef", "unifdefall", "vtool", "xml2man", "yacc",
	}}
}
