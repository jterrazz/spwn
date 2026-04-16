// Package mailbox provides agent-to-agent communication via a
// filesystem-based inbox system.
//
// Every recipient has a directory under an inbox root; each
// message is a JSON file named msg-<sender>-<timestamp>-<ms>.json
// with a "from", "to", "content", "type" (task, reply, question,
// announcement), and "status" (unread, read, delivered).
//
// Send, Check, CheckUnread, MarkRead, and ListAll are the public
// verbs — no daemon, no transport, no queue. The CLI commands
// `spwn agent send`, `spwn agent inbox`, and `spwn agent watch`
// feed off this package; the web UI uses ListAll for the
// cross-agent view.
package mailbox
