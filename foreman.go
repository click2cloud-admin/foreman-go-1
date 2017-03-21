package foreman

import (
	"net/http"
	"strings"
)

var (
	defaultHTTPClient = &http.Client{}
	defaultModifier   = mdfyFunc(func(req *http.Request) *http.Request { return req })
	defaultAddress    = "http://localhost:3000"
	defaultAPIVersion = "v2"
)

// Client represents the Foreman client.
type Client struct {
	httpClient *http.Client
	mods       modifier
}

// Options represents options to configure a Foreman client.
type Options struct {
	// Foreman address and API version
	Address    string
	APIVersion string

	// Foreman basic auth credentials
	Username string
	Password string

	// Use a specific HTTP Client
	HTTPClient *http.Client
}

// New setups a new Foreman client from the given options `opts` and returns it.
func New(opts Options) Client {
	client := Client{httpClient: defaultHTTPClient}
	if opts.HTTPClient != nil {
		client.httpClient = opts.HTTPClient
	}
	if opts.Address == "" {
		opts.Address = defaultAddress
	}
	if opts.APIVersion == "" {
		opts.APIVersion = defaultAPIVersion
	}
	mod := modifier(defaultModifier)
	mods := []modifierDecorator{
		setURLHostModifier(opts.Address, opts.APIVersion),
		addHeaderModifier("Content-Type", "application/json"),
		addHeaderModifier("Agent", "GoForemanAPIClient"),
	}
	if opts.Username != "" {
		mods = append(mods, setBasicAuthModifier(opts.Username, opts.Password))
	}
	for _, m := range mods {
		mod = m(mod)
	}
	client.mods = mod
	return client
}

// Head sends an Head HTTP request to Foreman and returns the response.
func (c Client) Head(resource string) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodHead, resource, nil)
	if err != nil {
		return nil, err
	}
	return c.Do(req)
}

// Do sends an HTTP request and returns its HTTP response.
// If there was a problem during that process, an error is returned.
func (c Client) Do(req *http.Request) (*http.Response, error) {
	return c.httpClient.Do(c.mods.Modify(req))
}

type modifier interface {
	Modify(*http.Request) *http.Request
}

type mdfyFunc func(*http.Request) *http.Request

func (m mdfyFunc) Modify(req *http.Request) *http.Request {
	return m(req)
}

type modifierDecorator func(modifier) modifier

func newModifierDecorator(ops func(*http.Request) *http.Request) modifierDecorator {
	return func(m modifier) modifier {
		return mdfyFunc(func(req *http.Request) *http.Request {
			newReq := ops(req)
			return m.Modify(newReq)
		})
	}
}

func addHeaderModifier(key, value string) modifierDecorator {
	return newModifierDecorator(func(req *http.Request) *http.Request {
		req.Header.Add(key, value)
		return req
	})
}

func setURLHostModifier(address, apiVersion string) modifierDecorator {
	return newModifierDecorator(func(req *http.Request) *http.Request {
		var scheme string
		switch {
		case strings.HasPrefix(address, "http://"):
			scheme = "http"
		case strings.HasPrefix(address, "https://"):
			scheme = "https"
		default:
			scheme = ""
		}
		req.URL.Host = strings.TrimPrefix(address, scheme+"://")
		req.URL.Path = apiVersion + "/" + strings.TrimPrefix(req.URL.Path, "/")
		req.URL.Scheme = scheme
		return req
	})
}

func setBasicAuthModifier(username, password string) modifierDecorator {
	return newModifierDecorator(func(req *http.Request) *http.Request {
		req.SetBasicAuth(username, password)
		return req
	})
}
