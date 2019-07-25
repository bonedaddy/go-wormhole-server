package log

import (
	"encoding/json"
	"testing"
)

func testOptions(opts Options, t *testing.T) {
	//Check its even valid
	err := opts.Verify()
	if err != nil {
		t.Error(err)
	}

	//Check marshaling
	jstr, err := json.Marshal(opts)
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

	if !jobj.Equals(opts) {
		t.Error("unmarshalled version did not equate to original")
	}
}

func TestDefaultOptions(t *testing.T) {
	opts := DefaultOptions

	testOptions(opts, t)
}

func TestLevel(t *testing.T) {
	opts := DefaultOptions

	opts.Level = "DUMMY"

	err := opts.Verify()
	if err == nil {
		t.Error("failed to catch bad  level")
	}
}

func TestMerge(t *testing.T) {
	tgt := DefaultOptions

	err := tgt.MergeFrom(Options{
		Level: "DEBUG",
	})
	if err != nil {
		t.Error(err)
	} else if tgt.Level != "DEBUG" {
		t.Error("expected a different  level")
	}

	err = tgt.MergeFrom(Options{
		Path:      "some-path",
		BlurTimes: true,
	})
	if err != nil {
		t.Error(err)
	} else if tgt.Path != "some-path" {
		t.Error("expected a different path")
	} else if tgt.BlurTimes == false {
		t.Error("expected a different blur")
	}

	err = tgt.MergeFrom(Options{
		BlurTimes: false,
	})
	if err != nil {
		t.Error(err)
	} else if tgt.BlurTimes == false {
		t.Error("blur should have stuck")
	}
}

func TestCombine(t *testing.T) {
	opts, err := CombineOptions(Options{
		Level: "BAD_LEVEL",
	})
	if err == nil {
		t.Error("expected the level to trip an error")
	}

	tgt := Options{
		Level:     "DEBUG",
		Path:      "some-path",
		BlurTimes: true,
	}

	opts, err = CombineOptions(tgt)
	if err != nil {
		t.Error(err)
	}

	testOptions(opts, t)
}
