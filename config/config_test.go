package config

import (
	"encoding/json"
	"testing"
)

func testOptions(opt Options, t *testing.T) {
	err := opt.Verify()
	if err != nil {
		t.Error(err)
	}

	//Check json marshaling
	jstr, err := json.Marshal(opt)
	if err != nil {
		t.Error(err)
	}

	var jobj Options
	err = json.Unmarshal(jstr, &jobj)
	if err != nil {
		t.Error(err)
	}

	err = jobj.Verify()
	if err != nil {
		t.Error(err)
	}

	if !jobj.Equals(opt) {
		t.Error("unmarshalled version did not equate to original")
	}
}

func TestDefaultOptions(t *testing.T) {
	opts := DefaultOptions

	testOptions(opts, t)
}

func TestOptionsMode(t *testing.T) {
	opts := DefaultOptions
	opts.Mode = "DUMMY"

	err := opts.Verify()
	if err == nil {
		t.Error("failed to catch bad server mode")
	}
}

func TestOptionsMerge(t *testing.T) {
	tgt := DefaultOptions

	opts := Options{}
	opts.Mode = "RELAY"
	opts.Relay.CleaningInterval = 2
	opts.Relay.ChannelExpiration = 5

	if err := tgt.MergeFrom(opts); err != nil {
		t.Error(err)
	}

	opts.Relay.CleaningInterval = 10
	if err := tgt.MergeFrom(opts); err == nil {
		t.Error("failed to find bad time intervals")
	}
}
