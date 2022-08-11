package probers

import (
	"reflect"
	"testing"

	"github.com/letsencrypt/boulder/observer/probers"
	"github.com/letsencrypt/boulder/test"
	"github.com/prometheus/client_golang/prometheus"
	"gopkg.in/yaml.v3"
)

func TestHTTPConf_MakeProber(t *testing.T) {
	conf := HTTPConf{}
	colls := conf.Instrument()
	badColl := prometheus.Collector(prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "obs_http_foo",
			Help: "Hmmm, this shouldn't be here...",
		},
		[]string{},
	));
	type fields struct {
		URL    string
		RCodes []int
	}
	tests := []struct {
		name    string
		fields  fields
		colls   map[string]*prometheus.Collector
		wantErr bool
	}{
		// valid
		{"valid fqdn valid rcode", fields{"http://example.com", []int{200}}, colls, false},
		{"valid hostname valid rcode", fields{"example", []int{200}}, colls, true},
		// invalid
		{"valid fqdn no rcode", fields{"http://example.com", nil}, colls, true},
		{"valid fqdn invalid rcode", fields{"http://example.com", []int{1000}}, colls, true},
		{"valid fqdn 1 invalid rcode", fields{"http://example.com", []int{200, 1000}}, colls, true},
		{"bad fqdn good rcode", fields{":::::", []int{200}}, colls, true},
		{"missing scheme", fields{"example.com", []int{200}}, colls, true},
		{
			"unexpected collector",
			fields{"http://example.com", []int{200}},
			map[string]*prometheus.Collector{"obs_http_foo": &badColl},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := HTTPConf{
				URL:    tt.fields.URL,
				RCodes: tt.fields.RCodes,
			}
			if _, err := c.MakeProber(tt.colls); (err != nil) != tt.wantErr {
				t.Errorf("HTTPConf.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestHTTPConf_Instrument(t *testing.T) {
	t.Run("instrument", func(t *testing.T) {
		conf := HTTPConf{}
		colls := conf.Instrument()
		for name := range colls {
			switch name {
			default:
				t.Errorf("HTTPConf.Instrument() returned unexpected Collector '%s'", name)
			}
		}
	})
}

func TestHTTPConf_UnmarshalSettings(t *testing.T) {
	type fields struct {
		url       interface{}
		rcodes    interface{}
		useragent interface{}
	}
	tests := []struct {
		name    string
		fields  fields
		want    probers.Configurer
		wantErr bool
	}{
		{"valid", fields{"google.com", []int{200}, "boulder_observer"}, HTTPConf{"google.com", []int{200}, "boulder_observer"}, false},
		{"invalid", fields{42, 42, 42}, nil, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			settings := probers.Settings{
				"url":       tt.fields.url,
				"rcodes":    tt.fields.rcodes,
				"useragent": tt.fields.useragent,
			}
			settingsBytes, _ := yaml.Marshal(settings)
			c := HTTPConf{}
			got, err := c.UnmarshalSettings(settingsBytes)
			if (err != nil) != tt.wantErr {
				t.Errorf("DNSConf.UnmarshalSettings() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("DNSConf.UnmarshalSettings() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHTTPProberName(t *testing.T) {
	// Test with blank `useragent`
	proberYAML := `
url: https://www.google.com
rcodes: [ 200 ]
useragent: ""
`
	c := HTTPConf{}
	colls := c.Instrument()
	configurer, err := c.UnmarshalSettings([]byte(proberYAML))
	test.AssertNotError(t, err, "Got error for valid prober config")
	prober, err := configurer.MakeProber(colls)
	test.AssertNotError(t, err, "Got error for valid prober config")
	test.AssertEquals(t, prober.Name(), "https://www.google.com-[200]-letsencrypt/boulder-observer-http-client")

	// Test with custom `useragent`
	proberYAML = `
url: https://www.google.com
rcodes: [ 200 ]
useragent: fancy-custom-http-client
`
	c = HTTPConf{}
	colls = c.Instrument()
	configurer, err = c.UnmarshalSettings([]byte(proberYAML))
	test.AssertNotError(t, err, "Got error for valid prober config")
	prober, err = configurer.MakeProber(colls)
	test.AssertNotError(t, err, "Got error for valid prober config")
	test.AssertEquals(t, prober.Name(), "https://www.google.com-[200]-fancy-custom-http-client")

}
