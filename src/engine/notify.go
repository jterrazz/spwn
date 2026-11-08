package engine

const NoticeTakeBackFailed = "take_back_failed"

// value: the events of SPEC §B3 the engine adds to the closed list of
// §10.4, each a Notification's Kind and the HUD's.
const (
	NoticeAgentStarted  = "agent_started"
	NoticeAgentWaiting  = "agent_waiting"
	NoticeAgentFinished = "agent_finished"
	NoticeVMReady       = "vm_ready"
	NoticeMacBack       = "mac_back"
	NoticeShareEnded    = "share_ended"
)

type Notification struct {
	Kind, Subject, Text string // value: what the engine tells the owner beyond the open surfaces, in the engine log's words.
}

func NoticeIcon(kind string) string {
	// value: the SF Symbol the HUD draws beside a notice's words, by its kind,
	// as STORIES.md's HUD words give it; empty for a kind the HUD never shows.
	return map[string]string{
		EventRestarting:     "arrow.clockwise",
		EventLoop:           "arrow.clockwise",
		EventPoweredOff:     "xmark.octagon.fill",
		EventBack:           "checkmark.circle.fill",
		NoticeAwakeEnded:    "moon",
		EventExposed:        "exclamationmark.shield",
		EventDiskLow:        "exclamationmark.circle.fill",
		NoticeAgentStarted:  "play.fill",
		NoticeAgentWaiting:  "bubble.left",
		NoticeAgentFinished: "checkmark.circle.fill",
		NoticeVMReady:       "checkmark.circle.fill",
		NoticeMacBack:       "checkmark.circle.fill",
		NoticeShareEnded:    "stop.fill",
	}[kind]
}
