package local

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
	"strconv"
	"time"
)

type Check struct {
	// value: the host:port it connects to, the URL it asks (empty: a
	// connection is the answer), the name a certificate must hold, and
	// whether that certificate is verified.
	Dial, URL, Name string
	Verify          bool
	Budget          time.Duration
}

type StatusError struct{ Status int }

func (e *StatusError) Error() string { return "answered " + strconv.Itoa(e.Status) }

func Answers(ctx context.Context, c Check) error {
	_, err := Answer(ctx, c)
	return err
}

func Timed(ctx context.Context, c Check) (int, time.Duration, error) {
	// invariant: Answer, with how long it waited for the answer.
	start := time.Now()
	status, err := Answer(ctx, c)
	return status, time.Since(start), err
}

func Answer(ctx context.Context, c Check) (int, error) {
	// invariant: a URL answers with 2xx or 3xx, its status said, a host:port
	// by accepting a connection (status 0), each within its budget;
	// redirects are not followed, a 3xx being an answer.
	ctx, cancel := context.WithTimeout(ctx, c.Budget)
	defer cancel()
	dialer := net.Dialer{}
	if c.URL == "" {
		conn, err := dialer.DialContext(ctx, "tcp", c.Dial)
		if err != nil {
			return 0, err
		}
		return 0, conn.Close()
	}
	transport := &http.Transport{
		DialContext: func(ctx context.Context, network, _ string) (net.Conn, error) {
			return dialer.DialContext(ctx, network, c.Dial)
		},
		TLSClientConfig: &tls.Config{ServerName: c.Name, InsecureSkipVerify: !c.Verify}, //nolint:gosec // a VM's own certificate is not verified for its own name (SPEC §7.2)
	}
	defer transport.CloseIdleConnections()
	client := http.Client{Transport: transport, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.URL, http.NoBody)
	if err != nil {
		return 0, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	_ = resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 400 {
		return resp.StatusCode, &StatusError{Status: resp.StatusCode}
	}
	return resp.StatusCode, nil
}
