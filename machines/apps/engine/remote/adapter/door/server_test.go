package door_test

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"errors"
	"io"
	"slices"
	"strings"
	"sync"
	"testing"

	"golang.org/x/crypto/ssh"

	"github.com/jterrazz/studio-station/apps/engine/platform/local"
	"github.com/jterrazz/studio-station/apps/engine/platform/local/localtest"
	"github.com/jterrazz/studio-station/apps/engine/remote/adapter/door"
)

type p256 struct{ k *ecdsa.PrivateKey }

func newKey(t *testing.T) p256 {
	t.Helper()
	k, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	return p256{k: k}
}

func (p p256) Public(context.Context) (*ecdsa.PublicKey, error) { return &p.k.PublicKey, nil }

func (p p256) Sign(_ context.Context, message []byte) ([]byte, error) {
	digest := sha256.Sum256(message)
	return ecdsa.SignASN1(rand.Reader, p.k, digest[:])
}

func (p p256) line(t *testing.T) string {
	t.Helper()
	pub, err := ssh.NewPublicKey(&p.k.PublicKey)
	if err != nil {
		t.Fatal(err)
	}
	return strings.TrimSpace(string(ssh.MarshalAuthorizedKey(pub)))
}

func (p p256) signer(t *testing.T) ssh.Signer {
	t.Helper()
	s, err := ssh.NewSignerFromKey(p.k)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

var errNotInBook = errors.New("not in the device book")

type book struct {
	mu    sync.Mutex
	held  map[string]string
	asked int
	keys  []string
	// value: the asks the book answers before its keys are forgotten; 0
	// keeps them.
	forgetAfter int
}

func (b *book) admit(_ context.Context, key string) (string, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.asked++
	b.keys = append(b.keys, key)
	if b.forgetAfter > 0 && b.asked > b.forgetAfter {
		return "", errNotInBook
	}
	if name, ok := b.held[key]; ok {
		return name, nil
	}
	return "", errNotInBook
}

type served struct {
	mu       sync.Mutex
	verdicts []door.Verdict
	done     chan struct{}
}

func (s *served) said(v door.Verdict) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.verdicts = append(s.verdicts, v)
}

func (s *served) only(t *testing.T) door.Verdict {
	t.Helper()
	<-s.done
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.verdicts) != 1 {
		t.Fatalf("the door said %d verdicts, want one: %+v", len(s.verdicts), s.verdicts)
	}
	return s.verdicts[0]
}

func serve(t *testing.T, host p256, b *book) (local.NetConn, *served) {
	t.Helper()
	s := &served{done: make(chan struct{})}
	d, err := door.New(context.Background(), host, b.admit, s.said)
	if err != nil {
		t.Fatal(err)
	}
	server, client := localtest.Pipe(t)
	go func() {
		defer close(s.done)
		_ = d.Serve(context.Background(), server)
	}()
	return client, s
}

func clientConfig(t *testing.T, host p256, auth ...ssh.AuthMethod) *ssh.ClientConfig {
	t.Helper()
	pub, err := ssh.NewPublicKey(&host.k.PublicKey)
	if err != nil {
		t.Fatal(err)
	}
	return &ssh.ClientConfig{User: "spwn", Auth: auth, HostKeyCallback: ssh.FixedHostKey(pub)}
}

func TestADeviceOfTheHeldBookIsLetIn(t *testing.T) {
	host, laptop := newKey(t), newKey(t)
	c, s := serve(t, host, &book{held: map[string]string{laptop.line(t): "laptop"}})
	if err := door.Knock(context.Background(), c, laptop, &host.k.PublicKey); err != nil {
		t.Fatalf("a device of the book was refused: %v", err)
	}
	v := s.only(t)
	if v.Device != "laptop" || v.Refused != "" || v.Line() != "door: laptop let in, key "+ssh.FingerprintSHA256(laptop.signer(t).PublicKey()) {
		t.Fatalf("verdict %+v, line %q", v, v.Line())
	}
}

func TestAKeyTheBookDoesNotHoldIsRefusedAndTheClientIsToldNoReason(t *testing.T) {
	host, stranger := newKey(t), newKey(t)
	c, s := serve(t, host, &book{held: map[string]string{}})
	err := door.Knock(context.Background(), c, stranger, &host.k.PublicKey)
	if err == nil {
		t.Fatal("a key outside the book was let in")
	}
	if strings.Contains(err.Error(), errNotInBook.Error()) {
		t.Fatalf("the client was told why: %v", err)
	}
	v := s.only(t)
	if v.Device != "" || v.Refused != errNotInBook.Error() || v.Fingerprint != ssh.FingerprintSHA256(stranger.signer(t).PublicKey()) {
		t.Fatalf("verdict %+v", v)
	}
}

func TestAHostKeyThatIsNotThePinnedOneIsNeverAuthenticatedTo(t *testing.T) {
	host, other, laptop := newKey(t), newKey(t), newKey(t)
	b := &book{held: map[string]string{laptop.line(t): "laptop"}}
	c, s := serve(t, host, b)
	if err := door.Knock(context.Background(), c, laptop, &other.k.PublicKey); err == nil {
		t.Fatal("a knock authenticated to a door whose host key is not the pinned one")
	}
	<-s.done
	if b.asked != 0 {
		t.Fatalf("the knock offered its key to a door it did not pin: asked %d", b.asked)
	}
}

type borrowed struct {
	pub  ssh.PublicKey
	sign ssh.Signer
}

func (b borrowed) PublicKey() ssh.PublicKey { return b.pub }

func (b borrowed) Sign(r io.Reader, data []byte) (*ssh.Signature, error) {
	sig, err := b.sign.Sign(r, data)
	if err != nil {
		return nil, err
	}
	sig.Format = b.pub.Type()
	return sig, nil
}

func TestAConnectionIsTheDeviceWhoseKeySignedAfterAKeyRefusedBeforeIt(t *testing.T) {
	host, laptop, phone, stranger := newKey(t), newKey(t), newKey(t), newKey(t)
	c, s := serve(t, host, &book{held: map[string]string{laptop.line(t): "laptop", phone.line(t): "phone"}})
	cfg := clientConfig(t, host, ssh.PublicKeys(stranger.signer(t), phone.signer(t)))
	conn, chans, reqs, err := ssh.NewClientConn(c, "door", cfg)
	if err != nil {
		t.Fatalf("the phone was refused: %v", err)
	}
	go ssh.DiscardRequests(reqs)
	go func() {
		for ch := range chans {
			_ = ch.Reject(ssh.Prohibited, "")
		}
	}()
	_ = conn.Close()
	if v := s.only(t); v.Device != "phone" || v.Refused != "" || v.Fingerprint != ssh.FingerprintSHA256(phone.signer(t).PublicKey()) {
		t.Fatalf("a connection refused a stranger's key, then signed with phone's: %+v", v)
	}
}

func TestASignatureByAnotherKeyLetsNobodyIn(t *testing.T) {
	host, laptop, stranger := newKey(t), newKey(t), newKey(t)
	c, s := serve(t, host, &book{held: map[string]string{laptop.line(t): "laptop"}})
	asksLaptop := borrowed{pub: laptop.signer(t).PublicKey(), sign: stranger.signer(t)}
	if _, _, _, err := ssh.NewClientConn(c, "door", clientConfig(t, host, ssh.PublicKeys(asksLaptop))); err == nil {
		t.Fatal("a signature by a key outside the book let its client in as laptop")
	}
	if v := s.only(t); v.Device != "" {
		t.Fatalf("verdict %+v", v)
	}
}

var errWithheld = errors.New("this key is only asked about")

type askedOnly struct{ pub ssh.PublicKey }

func (a askedOnly) PublicKey() ssh.PublicKey { return a.pub }

func (askedOnly) Sign(io.Reader, []byte) (*ssh.Signature, error) { return nil, errWithheld }

func scripted(methods ...ssh.AuthMethod) ssh.ClientAuthCallback {
	return func(*ssh.ClientAuthContext) (ssh.AuthMethod, error) {
		if len(methods) == 0 {
			return nil, errWithheld
		}
		next := methods[0]
		methods = methods[1:]
		return next, nil
	}
}

func onlyAsks(key p256, t *testing.T) ssh.AuthMethod {
	return ssh.PublicKeysCallback(func() ([]ssh.Signer, error) { return []ssh.Signer{askedOnly{pub: key.signer(t).PublicKey()}}, nil })
}

func TestAKeyAskedAboutLastIsNotTheDeviceThatSigned(t *testing.T) {
	host, laptop, phone := newKey(t), newKey(t), newKey(t)
	b := &book{held: map[string]string{laptop.line(t): "laptop", phone.line(t): "phone"}}
	c, s := serve(t, host, b)
	cfg := clientConfig(t, host)
	cfg.AuthCallback = scripted(onlyAsks(phone, t), onlyAsks(laptop, t), ssh.PublicKeys(phone.signer(t)))
	conn, chans, reqs, err := ssh.NewClientConn(c, "door", cfg)
	if err != nil {
		t.Fatalf("the phone was refused: %v", err)
	}
	go ssh.DiscardRequests(reqs)
	go func() {
		for ch := range chans {
			_ = ch.Reject(ssh.Prohibited, "")
		}
	}()
	_ = conn.Close()
	if v := s.only(t); v.Device != "phone" || v.Fingerprint != ssh.FingerprintSHA256(phone.signer(t).PublicKey()) {
		t.Fatalf("asked about phone, then laptop, signed as phone: the door said %+v", v)
	}
	if !slices.Contains(b.keys, laptop.line(t)) {
		t.Fatalf("the door was never asked about laptop's key: %v", b.keys)
	}
}

func TestABookKeyOnlyAskedAboutLetsNobodyIn(t *testing.T) {
	host, laptop := newKey(t), newKey(t)
	b := &book{held: map[string]string{laptop.line(t): "laptop"}}
	c, s := serve(t, host, b)
	cfg := clientConfig(t, host)
	cfg.AuthCallback = scripted(onlyAsks(laptop, t))
	if _, _, _, err := ssh.NewClientConn(c, "door", cfg); err == nil {
		t.Fatal("a client that only asked about laptop's key was let in")
	}
	if v := s.only(t); v.Device != "" || v.Refused == "" {
		t.Fatalf("a key asked about and never signed for: the door said %+v", v)
	}
	if !slices.Equal(b.keys, []string{laptop.line(t)}) {
		t.Fatalf("the door was asked about %v, want laptop's key alone", b.keys)
	}
}

func TestTheDoorTakesNoRSAKeyAndOffersNoOtherHostKey(t *testing.T) {
	host := newKey(t)
	rsaKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	rsaSigner, err := ssh.NewSignerFromKey(rsaKey)
	if err != nil {
		t.Fatal(err)
	}
	b := &book{held: map[string]string{strings.TrimSpace(string(ssh.MarshalAuthorizedKey(rsaSigner.PublicKey()))): "old"}}
	c, s := serve(t, host, b)
	if _, _, _, err := ssh.NewClientConn(c, "door", clientConfig(t, host, ssh.PublicKeys(rsaSigner))); err == nil {
		t.Fatal("an RSA key was let in")
	}
	<-s.done
	if b.asked != 0 {
		t.Fatalf("the door asked the book about an RSA key %d times", b.asked)
	}
	for _, offer := range []func(*ssh.ClientConfig){
		func(cfg *ssh.ClientConfig) { cfg.HostKeyAlgorithms = []string{ssh.KeyAlgoED25519, ssh.KeyAlgoRSA} },
		func(cfg *ssh.ClientConfig) { cfg.KeyExchanges = []string{ssh.InsecureKeyExchangeDH1SHA1} },
		func(cfg *ssh.ClientConfig) {
			cfg.KeyExchanges = []string{ssh.KeyExchangeDH14SHA256, ssh.KeyExchangeECDHP256}
		},
		func(cfg *ssh.ClientConfig) { cfg.Ciphers = []string{ssh.InsecureCipherTripleDESCBC} },
		func(cfg *ssh.ClientConfig) { cfg.Ciphers = []string{ssh.InsecureCipherRC4256} },
		func(cfg *ssh.ClientConfig) { cfg.Ciphers = []string{ssh.CipherAES128CTR, ssh.CipherAES256CTR} },
	} {
		laptop := newKey(t)
		b := &book{held: map[string]string{laptop.line(t): "laptop"}}
		c, s := serve(t, host, b)
		cfg := clientConfig(t, host, ssh.PublicKeys(laptop.signer(t)))
		offer(cfg)
		if _, _, _, err := ssh.NewClientConn(c, "door", cfg); err == nil {
			t.Fatalf("a client offering only %v %v %v %v was let in", cfg.HostKeyAlgorithms, cfg.KeyExchanges, cfg.Ciphers, cfg.MACs)
		}
		<-s.done
		if b.asked != 0 {
			t.Fatal("a downgraded client reached the book")
		}
	}
}

func TestADoorKeyThatIsNotP256OpensNoDoor(t *testing.T) {
	k, err := ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := door.New(context.Background(), p256{k: k}, (&book{}).admit, func(door.Verdict) {}); err == nil {
		t.Fatal("a P-384 key opened a door that signs as ecdsa-sha2-nistp256")
	}
}

func TestALetInDeviceOpensNoChannelAndNoRequestAndReadsNoVersion(t *testing.T) {
	host, laptop := newKey(t), newKey(t)
	c, s := serve(t, host, &book{held: map[string]string{laptop.line(t): "laptop"}})
	conn, chans, reqs, err := ssh.NewClientConn(c, "door", clientConfig(t, host, ssh.PublicKeys(laptop.signer(t))))
	if err != nil {
		t.Fatal(err)
	}
	go ssh.DiscardRequests(reqs)
	go func() {
		for ch := range chans {
			_ = ch.Reject(ssh.Prohibited, "")
		}
	}()
	if got := string(conn.ServerVersion()); got != "SSH-2.0-spwn" {
		t.Fatalf("the door answers %q", got)
	}
	for kind, want := range map[string]ssh.RejectionReason{
		"session": ssh.Prohibited, "direct-tcpip": ssh.UnknownChannelType, "direct-streamlocal@openssh.com": ssh.UnknownChannelType,
		"x11": ssh.UnknownChannelType, "forwarded-tcpip": ssh.UnknownChannelType, "auth-agent@openssh.com": ssh.UnknownChannelType,
		"spwn-api@spwn": ssh.UnknownChannelType, "pipe@spwn": ssh.UnknownChannelType, "terminal@spwn": ssh.UnknownChannelType,
	} {
		_, _, err := conn.OpenChannel(kind, nil)
		var open *ssh.OpenChannelError
		if !errors.As(err, &open) || open.Reason != want {
			t.Fatalf("channel %s: %v, want %v", kind, err, want)
		}
	}
	for _, kind := range []string{"tcpip-forward", "cancel-tcpip-forward", "streamlocal-forward@openssh.com", "keepalive@openssh.com", "no-more-sessions@openssh.com"} {
		ok, _, err := conn.SendRequest(kind, true, nil)
		if err != nil || ok {
			t.Fatalf("request %s: ok %v, %v", kind, ok, err)
		}
	}
	_ = conn.Close()
	if v := s.only(t); v.Device != "laptop" {
		t.Fatalf("verdict %+v", v)
	}
}

func drain(conn ssh.Conn, chans <-chan ssh.NewChannel, reqs <-chan *ssh.Request) {
	go ssh.DiscardRequests(reqs)
	go func() {
		for ch := range chans {
			_ = ch.Reject(ssh.Prohibited, "")
		}
	}()
	_ = conn.Close()
}

type forgetOnSign struct {
	inner  ssh.Signer
	before func()
}

func (f forgetOnSign) PublicKey() ssh.PublicKey { return f.inner.PublicKey() }

func (f forgetOnSign) Sign(r io.Reader, data []byte) (*ssh.Signature, error) {
	f.before()
	return f.inner.Sign(r, data)
}

func TestADeviceForgottenBetweenItsQueryAndItsSignatureIsRefused(t *testing.T) {
	host, laptop := newKey(t), newKey(t)
	b := &book{held: map[string]string{laptop.line(t): "laptop"}}
	c, s := serve(t, host, b)
	forget := func() {
		b.mu.Lock()
		defer b.mu.Unlock()
		delete(b.held, laptop.line(t))
	}
	conn, chans, reqs, err := ssh.NewClientConn(c, "door", clientConfig(t, host, ssh.PublicKeys(forgetOnSign{inner: laptop.signer(t), before: forget})))
	if err == nil {
		drain(conn, chans, reqs)
		t.Fatal("a device forgotten after its key was asked about, before it signed, finished its handshake")
	}
	if v := s.only(t); v.Device != "" || v.Refused != errNotInBook.Error() || v.Fingerprint != ssh.FingerprintSHA256(laptop.signer(t).PublicKey()) {
		t.Fatalf("verdict %+v", v)
	}
}

func TestAJudgeThatPanicsEndsThatConnectionAloneWithItsLine(t *testing.T) {
	host, laptop, broken := newKey(t), newKey(t), newKey(t)
	s := &served{done: make(chan struct{})}
	admit := func(_ context.Context, key string) (string, error) {
		if key == broken.line(t) {
			panic("the judge broke")
		}
		return "laptop", nil
	}
	d, err := door.New(context.Background(), host, admit, s.said)
	if err != nil {
		t.Fatal(err)
	}
	server, client := localtest.Pipe(t)
	knocked := make(chan error, 1)
	go func() { knocked <- door.Knock(context.Background(), client, broken, &host.k.PublicKey) }()
	if err := d.Serve(context.Background(), server); err == nil {
		t.Fatal("a connection whose judge panicked was served")
	}
	if err := <-knocked; err == nil {
		t.Fatal("a connection whose judge panicked was let in")
	}
	close(s.done)
	if v := s.only(t); v.Device != "" || v.Refused != "the door broke on this connection: the judge broke" {
		t.Fatalf("verdict %+v", v)
	}
	server, client = localtest.Pipe(t)
	go func() { knocked <- door.Knock(context.Background(), client, laptop, &host.k.PublicKey) }()
	_ = d.Serve(context.Background(), server)
	if err := <-knocked; err != nil {
		t.Fatalf("the door let nobody in after a connection broke: %v", err)
	}
}

type contextBound struct{ p256 }

func (k contextBound) Sign(ctx context.Context, message []byte) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return k.p256.Sign(ctx, message)
}

func TestTheDoorStillSignsOnceTheContextItWasMadeInEnds(t *testing.T) {
	host, laptop := newKey(t), newKey(t)
	made, end := context.WithCancel(context.Background())
	s := &served{done: make(chan struct{})}
	d, err := door.New(made, contextBound{host}, (&book{held: map[string]string{laptop.line(t): "laptop"}}).admit, s.said)
	if err != nil {
		t.Fatal(err)
	}
	end()
	server, client := localtest.Pipe(t)
	go func() {
		defer close(s.done)
		_ = d.Serve(context.Background(), server)
	}()
	if err := door.Knock(context.Background(), client, laptop, &host.k.PublicKey); err != nil {
		t.Fatalf("a door whose constructor's context ended no longer signs: %v", err)
	}
	if v := s.only(t); v.Device != "laptop" {
		t.Fatalf("verdict %+v", v)
	}
}

func TestAKeyAdmittedUnderNoNameIsRefused(t *testing.T) {
	host, laptop := newKey(t), newKey(t)
	c, s := serve(t, host, &book{held: map[string]string{laptop.line(t): ""}})
	if err := door.Knock(context.Background(), c, laptop, &host.k.PublicKey); err == nil {
		t.Fatal("a key the book admitted under no name was let in")
	}
	if v := s.only(t); v.Device != "" || v.Refused != "the book names no device for this key" {
		t.Fatalf("verdict %+v", v)
	}
}

func TestADeviceForgottenAsItsHandshakeEndsIsRefused(t *testing.T) {
	host, laptop := newKey(t), newKey(t)
	counted := &book{held: map[string]string{laptop.line(t): "laptop"}}
	c, s := serve(t, host, counted)
	if err := door.Knock(context.Background(), c, laptop, &host.k.PublicKey); err != nil {
		t.Fatal(err)
	}
	if v := s.only(t); v.Device != "laptop" {
		t.Fatalf("verdict %+v", v)
	}
	b := &book{held: map[string]string{laptop.line(t): "laptop"}, forgetAfter: counted.asked - 1}
	c, s = serve(t, host, b)
	conn, chans, reqs, err := ssh.NewClientConn(c, "door", clientConfig(t, host, ssh.PublicKeys(laptop.signer(t))))
	if err != nil {
		t.Fatalf("the handshake itself refused the device, the door's last judgement was never reached: %v", err)
	}
	drain(conn, chans, reqs)
	if v := s.only(t); v.Device != "" || v.Refused != errNotInBook.Error() {
		t.Fatalf("a device forgotten once its handshake was over was let in: %+v", v)
	}
}
