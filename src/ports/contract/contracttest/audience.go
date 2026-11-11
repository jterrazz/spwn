package contracttest

import (
	"context"
	"strconv"
	"strings"
	"testing"

	"github.com/jterrazz/studio-station/src/catalog"
	"github.com/jterrazz/studio-station/src/ports"
	"github.com/jterrazz/studio-station/src/ports/contract"
)

func Audience(t *testing.T, open func(t *testing.T) (contract.Audience, contract.Tailnet)) {
	// value: open answers an audience over a node signed in to its tailnet,
	// sharing on, sharing nothing yet, and that node, which reads its shares
	// back.
	ctx := context.Background()
	web := ports.Share{Port: 40001, Reserved: 40001, Scheme: ports.SchemeHTTPS, To: catalog.AudiencePrivate}
	raw := ports.Share{Port: 40002, Reserved: 40002, Scheme: ports.SchemeTCP, To: catalog.AudiencePrivate}
	mounted := ports.Share{Port: 443, Path: "/linux-1/5173", Reserved: 40003, Scheme: ports.SchemeHTTPS, To: catalog.AudiencePrivate}
	public := ports.Share{Port: 8443, Reserved: 40004, Scheme: ports.SchemeHTTPS, To: catalog.AudiencePublic}
	local := ports.Share{Port: 40005, Reserved: 40005, Scheme: ports.SchemeHTTPS, To: catalog.AudienceLocal}

	t.Run("a private share is served where it is listed, carried from this Mac's loopback only", func(t *testing.T) {
		a, tn := open(t)
		ready, _ := tn.Ready(ctx)
		want := map[ports.Share]string{
			web:     "https://" + ready.DNSName + ":40001",
			raw:     ready.DNSName + ":40002",
			mounted: "https://" + ready.DNSName + "/linux-1/5173/",
		}
		for _, s := range []ports.Share{web, raw, mounted} {
			url, err := a.Share(ctx, s)
			if err != nil || url != want[s] {
				t.Fatalf("Share(%+v) = %q, %v; want %q", s, url, err, want[s])
			}
		}
		shared, err := tn.Shared(ctx)
		if err != nil || len(shared) != 3 {
			t.Fatalf("Shared = %+v, %v; want the three shares", shared, err)
		}
		for _, s := range []ports.Share{web, raw, mounted} {
			got, ok := find(shared, s)
			if !ok {
				t.Fatalf("Shared = %+v; want %+v listed", shared, s)
			}
			if got.Public() || !strings.Contains(got.Target, "127.0.0.1:"+strconv.Itoa(s.Reserved)) || strings.Contains(got.Target, "0.0.0.0") {
				t.Fatalf("share %+v is %+v; want private, carried from 127.0.0.1:%d", s, got, s.Reserved)
			}
		}
	})

	t.Run("a public share is opened on its Funnel port, to the internet", func(t *testing.T) {
		a, tn := open(t)
		ready, _ := tn.Ready(ctx)
		url, err := a.Share(ctx, public)
		if want := "https://" + ready.DNSName + ":8443"; err != nil || url != want {
			t.Fatalf("Share(%+v) = %q, %v; want %q", public, url, err, want)
		}
		shared, err := tn.Shared(ctx)
		got, ok := find(shared, public)
		if err != nil || !ok || !got.Public() || !strings.Contains(got.Target, "127.0.0.1:40004") {
			t.Fatalf("Shared = %+v, %v; want %+v public, carried from 127.0.0.1:40004", shared, err, public)
		}
		if err := a.Unshare(ctx, public); err != nil {
			t.Fatalf("Unshare: %v", err)
		}
		if shared, _ := tn.Shared(ctx); len(shared) != 0 {
			t.Fatalf("Shared after unsharing = %+v; want none", shared)
		}
	})

	t.Run("a local share opens nothing: the loopback listener is all of it", func(t *testing.T) {
		a, tn := open(t)
		url, err := a.Share(ctx, local)
		if err != nil || url != "" {
			t.Fatalf("Share(%+v) = %q, %v; want no URL and no error", local, url, err)
		}
		if shared, err := tn.Shared(ctx); err != nil || len(shared) != 0 {
			t.Fatalf("Shared = %+v, %v; want nothing served", shared, err)
		}
		if err := a.Unshare(ctx, local); err != nil {
			t.Fatalf("Unshare(%+v): %v", local, err)
		}
	})

	t.Run("an unshared port leaves the others served", func(t *testing.T) {
		a, tn := open(t)
		for _, s := range []ports.Share{web, raw, mounted} {
			if _, err := a.Share(ctx, s); err != nil {
				t.Fatalf("Share(%+v): %v", s, err)
			}
		}
		if err := a.Unshare(ctx, web); err != nil {
			t.Fatalf("Unshare: %v", err)
		}
		shared, err := tn.Shared(ctx)
		if _, still := find(shared, web); err != nil || still || len(shared) != 2 {
			t.Fatalf("Shared after unsharing %+v = %+v, %v; want the two others", web, shared, err)
		}
	})
}

func find(shared []ports.Share, s ports.Share) (ports.Share, bool) {
	for _, got := range shared {
		if got.Port == s.Port && got.Scheme == s.Scheme && got.Mounted() == s.Mounted() && (!s.Mounted() || got.Path == s.Path) {
			return got, true
		}
	}
	return ports.Share{}, false
}
