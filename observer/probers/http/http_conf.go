package probers

import (
	"fmt"
	"net/url"

	"github.com/letsencrypt/boulder/observer/probers"
	"github.com/prometheus/client_golang/prometheus"
	"gopkg.in/yaml.v3"
)

// HTTPConf is exported to receive YAML configuration.
type HTTPConf struct {
	URL       string `yaml:"url"`
	RCodes    []int  `yaml:"rcodes"`
	UserAgent string `yaml:"useragent"`
}

// UnmarshalSettings takes YAML as bytes and unmarshals it to the to an
// HTTPConf object.
func (c HTTPConf) UnmarshalSettings(settings []byte) (probers.Configurer, error) {
	var conf HTTPConf
	err := yaml.Unmarshal(settings, &conf)
	if err != nil {
		return nil, err
	}
	return conf, nil
}

func (c HTTPConf) validateURL() error {
	url, err := url.Parse(c.URL)
	if err != nil {
		return fmt.Errorf(
			"invalid 'url', got: %q, expected a valid url", c.URL)
	}
	if url.Scheme == "" {
		return fmt.Errorf(
			"invalid 'url', got: %q, missing scheme", c.URL)
	}
	return nil
}

func (c HTTPConf) validateRCodes() error {
	if len(c.RCodes) == 0 {
		return fmt.Errorf(
			"invalid 'rcodes', got: %q, please specify at least one", c.RCodes)
	}
	for _, c := range c.RCodes {
		// ensure rcode entry is in range 100-599
		if c < 100 || c > 599 {
			return fmt.Errorf(
				"'rcodes' contains an invalid HTTP response code, '%d'", c)
		}
	}
	return nil
}

// MakeProber constructs a `HTTPProbe` object from the contents of the
// bound `HTTPConf` object. If the `HTTPConf` cannot be validated, an
// error appropriate for end-user consumption is returned instead.
func (c HTTPConf) MakeProber(_ map[string]*prometheus.Collector) (probers.Prober, error) {
	// validate `url`
	err := c.validateURL()
	if err != nil {
		return nil, err
	}

	// validate `rcodes`
	err = c.validateRCodes()
	if err != nil {
		return nil, err
	}

	// Set default User-Agent if none set.
	if c.UserAgent == "" {
		c.UserAgent = "letsencrypt/boulder-observer-http-client"
	}
	return HTTPProbe{c.URL, c.RCodes, c.UserAgent}, nil
}

// Instrument constructs any `prometheus.Collector` objects the `HTTPProbe` will
// need to report its own metrics. A map is returned containing the constructed
// objects, indexed by the name of the prometheus metric. If no objects were
// constructed, an empty map is returned.
func (c HTTPConf) Instrument() map[string]*prometheus.Collector {
	return map[string]*prometheus.Collector{}
}

// init is called at runtime and registers `HTTPConf`, a `Prober`
// `Configurer` type, as "HTTP".
func init() {
	probers.Register("HTTP", HTTPConf{})
}
