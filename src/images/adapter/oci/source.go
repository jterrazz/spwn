package oci

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/jterrazz/studio-station/src/images"
	"github.com/jterrazz/studio-station/src/images/contract"
)

type Getter interface {
	Get(ctx context.Context, url string, header map[string]string) (status int, body []byte, err error)
}

type Source struct {
	// invariant: the registry of station's published images, read anonymously
	// over the OCI distribution API: a token for pull, then the tags.
	net   Getter
	base  string
	owner string
	// value: the hypervisors that fetch an OCI reference themselves.
	fetchers []string
}

var _ contract.ImageSource = (*Source)(nil)

func New(net Getter, base, owner string, fetchers ...string) *Source {
	return &Source{net: net, base: strings.TrimSuffix(base, "/"), owner: owner, fetchers: fetchers}
}

func (s *Source) repository(name string) string { return s.owner + "/station-" + name }

func (s *Source) registry() string {
	_, registry, found := strings.Cut(s.base, "://")
	if !found {
		return s.base
	}
	return registry
}

func (s *Source) Latest(ctx context.Context, name string) (images.Ref, error) {
	repo := s.repository(name)
	token, err := s.token(ctx, repo)
	if err != nil {
		return images.Ref{}, err
	}
	status, body, err := s.net.Get(ctx, s.base+"/v2/"+repo+"/tags/list", map[string]string{"Authorization": "Bearer " + token})
	switch {
	case err != nil:
		return images.Ref{}, err
	case status == 404:
		return images.Ref{}, contract.ErrNotReleased
	case status != 200:
		return images.Ref{}, fmt.Errorf("the registry answered %d for %s", status, repo)
	}
	var list struct {
		Tags []string `json:"tags"`
	}
	if err := json.Unmarshal(body, &list); err != nil {
		return images.Ref{}, fmt.Errorf("the registry's tags of %s: %w", repo, err)
	}
	newest := ""
	for _, tag := range list.Tags {
		if version(tag) && (newest == "" || images.CompareVersions(tag, newest) > 0) {
			newest = tag
		}
	}
	if newest == "" {
		return images.Ref{}, contract.ErrNotReleased
	}
	return images.Ref{Name: name, Version: newest}, nil
}

func (s *Source) token(ctx context.Context, repo string) (string, error) {
	status, body, err := s.net.Get(ctx, s.base+"/token?scope=repository:"+repo+":pull", nil)
	switch {
	case err != nil:
		return "", err
	case status == 403 || status == 404:
		// an image nobody released answers a pull token with 403 (a private or
		// absent package), the same as a tags list with 404.
		return "", contract.ErrNotReleased
	case status != 200:
		return "", fmt.Errorf("the registry refused a pull token for %s: %d", repo, status)
	}
	var answer struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(body, &answer); err != nil || answer.Token == "" {
		return "", errors.Join(errors.New("the registry answered no pull token for "+repo), err)
	}
	return answer.Token, nil
}

// invariant: a published version is dotted digits; any other tag (latest, a
// digest's name) is not one.
func version(tag string) bool {
	if tag == "" {
		return false
	}
	for part := range strings.SplitSeq(tag, ".") {
		if part == "" || strings.Trim(part, "0123456789") != "" {
			return false
		}
	}
	return true
}

func (s *Source) Locate(_ context.Context, r images.Ref, hypervisor string) (contract.ImageLocation, error) {
	if !slices.Contains(s.fetchers, hypervisor) {
		return contract.ImageLocation{}, contract.ErrNotForHypervisor
	}
	return contract.ImageLocation{Hypervisor: hypervisor, Ref: s.registry() + "/" + s.repository(r.Name) + ":" + r.Version}, nil
}
