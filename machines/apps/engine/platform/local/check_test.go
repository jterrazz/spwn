package local

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"math/big"
	"net"
	"net/http"
	"testing"
	"time"
)

// value: a server on this Mac's loopback answering status, over TLS with a
// self-signed certificate for infra alone when tlsOn, as a cluster's own.
func serve(t *testing.T, status *int, tlsOn bool) string {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	s := &http.Server{ReadHeaderTimeout: time.Second, Handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Location", "/elsewhere")
		w.WriteHeader(*status)
	})}
	if tlsOn {
		key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		if err != nil {
			t.Fatal(err)
		}
		tmpl := x509.Certificate{SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "infra"}, DNSNames: []string{"infra"}, NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().Add(time.Hour)}
		der, err := x509.CreateCertificate(rand.Reader, &tmpl, &tmpl, &key.PublicKey, key)
		if err != nil {
			t.Fatal(err)
		}
		l = tls.NewListener(l, &tls.Config{MinVersion: tls.VersionTLS12, Certificates: []tls.Certificate{{Certificate: [][]byte{der}, PrivateKey: key}}})
	}
	go func() { _ = s.Serve(l) }()
	t.Cleanup(func() { _ = s.Close() })
	return l.Addr().String()
}

func TestAServersOwnSelfSignedCertificatePassesAndAnotherNamesFails(t *testing.T) {
	status := http.StatusOK
	at := serve(t, &status, true)
	_, port, _ := net.SplitHostPort(at)
	if err := Answers(t.Context(), Check{Dial: at, URL: "https://infra:" + port + "/readyz", Name: "infra", Budget: 5 * time.Second}); err != nil {
		t.Errorf("the VM's own name with its self-signed certificate failed: %v", err)
	}
	if err := Answers(t.Context(), Check{Dial: at, URL: "https://api.example.com:" + port + "/health", Name: "api.example.com", Verify: true, Budget: 5 * time.Second}); err == nil {
		t.Error("another name with a certificate it does not hold passed")
	}
}

func TestAURLAnswersWith2xxOr3xxAndFailsOtherwise(t *testing.T) {
	status := http.StatusFound
	at := serve(t, &status, false)
	c := Check{Dial: at, URL: "http://infra/readyz", Name: "infra", Budget: 5 * time.Second}
	if err := Answers(t.Context(), c); err != nil {
		t.Errorf("a 302 failed: %v", err)
	}
	status = http.StatusServiceUnavailable
	if err := Answers(t.Context(), c); err == nil {
		t.Error("a 503 passed")
	}
}

func TestAMacPortPassesWhileItAcceptsAndFailsOnceItRefuses(t *testing.T) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	c := Check{Dial: l.Addr().String(), Budget: 5 * time.Second}
	if err := Answers(t.Context(), c); err != nil {
		t.Errorf("a listening port failed: %v", err)
	}
	_ = l.Close()
	if err := Answers(t.Context(), c); err == nil {
		t.Error("a closed port passed")
	}
}
