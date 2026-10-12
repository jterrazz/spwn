package automation

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"spwn.sh/apps/cli/cliproject"
)

var (
	logsFollow bool
	logsLimit  int
)

var logsCmd = &cobra.Command{
	Use:   "logs",
	Short: "Tail .spwn/runs.jsonl receipts",
	RunE: func(cmd *cobra.Command, args []string) error {
		p, err := cliproject.Require()
		if err != nil {
			return err
		}

		rows, err := readReceipts(p.Root)
		if err != nil {
			return fmt.Errorf("read receipts: %w", err)
		}

		if logsLimit > 0 && len(rows) > logsLimit {
			rows = rows[len(rows)-logsLimit:]
		}

		out := cmd.OutOrStdout()
		for _, r := range rows {
			fmt.Fprintln(out, formatReceiptLine(r))
		}

		if !logsFollow {
			return nil
		}
		path := receiptsPath(p.Root)
		f, err := os.Open(path)
		if err != nil {
			if !os.IsNotExist(err) {
				return err
			}
			f, err = waitForFile(path, cmd.Context())
			if err != nil {
				return err
			}
		}
		defer f.Close()

		if _, err := f.Seek(0, io.SeekEnd); err != nil {
			return err
		}
		return tailReceipts(f, out, cmd.Context())
	},
}

func init() {
	logsCmd.Flags().BoolVarP(&logsFollow, "follow", "f", false, "stream new receipts as they arrive")
	logsCmd.Flags().IntVarP(&logsLimit, "limit", "n", 50, "show at most N most-recent receipts before --follow takes over")
}

func formatReceiptLine(r receiptRow) string {
	status := "ok"
	if !r.OK {
		status = "FAIL"
	}
	dur := time.Duration(r.DurationMS) * time.Millisecond
	durStr := dur.Round(time.Millisecond).String()
	line := fmt.Sprintf("%s  %s/%s  %s  %s  %s  %s",
		r.Fired.UTC().Format(time.RFC3339),
		r.World, r.Automation,
		r.Trigger,
		r.Reason,
		status,
		durStr,
	)
	if r.Missed > 0 {
		line += fmt.Sprintf("  missed=%d", r.Missed)
	}
	if len(r.EventPaths) > 1 {
		line += fmt.Sprintf("  paths=%d", len(r.EventPaths))
	}
	if r.Error != "" {
		msg := strings.ReplaceAll(r.Error, "\n", " ")
		if len(msg) > 120 {
			msg = msg[:120] + "…"
		}
		line += "  err=" + msg
	}
	return line
}

func waitForFile(path string, ctx context.Context) (*os.File, error) {
	t := time.NewTicker(250 * time.Millisecond)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-t.C:
			f, err := os.Open(path)
			if err == nil {
				return f, nil
			}
			if !os.IsNotExist(err) {
				return nil, err
			}
		}
	}
}

func tailReceipts(f *os.File, out io.Writer, ctx context.Context) error {
	buf := make([]byte, 4096)
	var pending []byte
	t := time.NewTicker(250 * time.Millisecond)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-t.C:
			n, err := f.Read(buf)
			if err != nil && err != io.EOF {
				return err
			}
			if n == 0 {
				continue
			}
			pending = append(pending, buf[:n]...)
			for {
				idx := indexByte(pending, '\n')
				if idx < 0 {
					break
				}
				line := pending[:idx]
				pending = pending[idx+1:]
				row, ok := parseReceiptLine(line)
				if ok {
					fmt.Fprintln(out, formatReceiptLine(row))
				}
			}
		}
	}
}

func indexByte(b []byte, c byte) int {
	for i, x := range b {
		if x == c {
			return i
		}
	}
	return -1
}

func parseReceiptLine(b []byte) (receiptRow, bool) {
	rows, err := decodeReceipts(bytesReader(b))
	if err != nil || len(rows) == 0 {
		return receiptRow{}, false
	}
	return rows[0], true
}

func bytesReader(b []byte) io.Reader {
	return &sliceReader{b: b}
}

type sliceReader struct {
	b []byte
	i int
}

func (r *sliceReader) Read(p []byte) (int, error) {
	if r.i >= len(r.b) {
		return 0, io.EOF
	}
	n := copy(p, r.b[r.i:])
	r.i += n
	return n, nil
}
