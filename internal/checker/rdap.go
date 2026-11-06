package checker

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/publicsuffix"
)

// Reasons a domain expiry is unknown.
const (
	ReasonLocal       = "local"       // no public registration: IP or home network name
	ReasonUnsupported = "unsupported" // the registry offers no RDAP service
	ReasonNotFound    = "not_found"
	ReasonNoExpiry    = "no_expiry" // the registry does not publish the date
)

const ianaBootstrap = "https://data.iana.org/rdap/dns.json"

// DomainInfo is a settled answer. Temporary failures are errors instead, so a
// registry hiccup does not erase a known expiry date.
type DomainInfo struct {
	Domain  string
	Expires time.Time
	Reason  string
}

// Domains looks up registration expiry over RDAP. Registry addresses come from
// IANA's bootstrap file, so domain names go only to their own registry.
type Domains struct {
	Bootstrap string // empty means IANA
	Client    *http.Client

	mu       sync.Mutex
	services map[string]string // TLD to RDAP base URL
	fetched  time.Time
}

func NewDomains() *Domains {
	return &Domains{Bootstrap: ianaBootstrap, Client: &http.Client{Timeout: 15 * time.Second}}
}

// Registrable returns the name a registry knows, such as example.co.uk for
// app.example.co.uk. It reports false for addresses without one.
func Registrable(rawURL string) (string, bool) {
	if IsLocal(rawURL) {
		return "", false
	}
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", false
	}
	host := strings.ToLower(u.Hostname())
	if net.ParseIP(host) != nil {
		return "", false
	}
	return registeredName(host)
}

// registeredName walks past private suffixes such as vercel.app and github.io:
// those are real domains to a registry, though browsers treat them as TLDs.
func registeredName(host string) (string, bool) {
	suffix, icann := publicsuffix.PublicSuffix(host)
	for !icann && strings.Contains(suffix, ".") {
		suffix = suffix[strings.IndexByte(suffix, '.')+1:]
		_, icann = publicsuffix.PublicSuffix(suffix)
	}
	rest, found := strings.CutSuffix(host, "."+suffix)
	if !found {
		return "", false
	}
	return rest[strings.LastIndexByte(rest, '.')+1:] + "." + suffix, true
}

func (d *Domains) Lookup(ctx context.Context, rawURL string) (DomainInfo, error) {
	domain, ok := Registrable(rawURL)
	if !ok {
		return DomainInfo{Reason: ReasonLocal}, nil
	}
	info := DomainInfo{Domain: domain}
	base, err := d.service(ctx, domain)
	if err != nil {
		return info, err
	}
	if base == "" {
		info.Reason = ReasonUnsupported
		return info, nil
	}

	var body struct {
		Events []struct {
			Action string `json:"eventAction"`
			Date   string `json:"eventDate"`
		} `json:"events"`
	}
	status, err := d.getJSON(ctx, strings.TrimRight(base, "/")+"/domain/"+domain, "application/rdap+json", &body)
	if err != nil {
		return info, err
	}
	if status == http.StatusNotFound {
		info.Reason = ReasonNotFound
		return info, nil
	}
	if status != http.StatusOK {
		return info, fmt.Errorf("rdap %s: status %d", domain, status)
	}
	for _, event := range body.Events {
		if event.Action != "expiration" {
			continue
		}
		expires, err := time.Parse(time.RFC3339, event.Date)
		if err != nil {
			return info, fmt.Errorf("rdap %s: %w", domain, err)
		}
		info.Expires = expires
		return info, nil
	}
	info.Reason = ReasonNoExpiry
	return info, nil
}

// service finds the RDAP base for a domain's TLD, refreshing the table daily.
func (d *Domains) service(ctx context.Context, domain string) (string, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.services == nil || time.Since(d.fetched) > 24*time.Hour {
		services, err := d.fetchServices(ctx)
		if err != nil {
			if d.services == nil {
				return "", err
			}
		} else {
			d.services, d.fetched = services, time.Now()
		}
	}
	return d.services[domain[strings.LastIndexByte(domain, '.')+1:]], nil
}

func (d *Domains) fetchServices(ctx context.Context) (map[string]string, error) {
	var body struct {
		Services [][][]string `json:"services"`
	}
	status, err := d.getJSON(ctx, d.Bootstrap, "application/json", &body)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("rdap bootstrap: status %d", status)
	}
	services := map[string]string{}
	for _, service := range body.Services {
		if len(service) != 2 || len(service[1]) == 0 {
			continue
		}
		base := service[1][0]
		for _, candidate := range service[1] {
			if strings.HasPrefix(candidate, "https://") {
				base = candidate
				break
			}
		}
		for _, tld := range service[0] {
			services[strings.ToLower(tld)] = base
		}
	}
	if len(services) == 0 {
		return nil, errors.New("rdap bootstrap: empty")
	}
	return services, nil
}

func (d *Domains) getJSON(ctx context.Context, target, accept string, value any) (int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return 0, err
	}
	req.Header.Set("Accept", accept)
	req.Header.Set("User-Agent", "Lucy/0.1")
	resp, err := d.Client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		io.Copy(io.Discard, io.LimitReader(resp.Body, maxBody))
		return resp.StatusCode, nil
	}
	return resp.StatusCode, json.NewDecoder(io.LimitReader(resp.Body, 4*maxBody)).Decode(value)
}
