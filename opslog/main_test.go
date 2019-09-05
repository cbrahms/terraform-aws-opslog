package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHarvestTags(t *testing.T) {
	cases := []struct {
		in   string
		tags []string
	}{
		{in: "testing this thing out #i:foo", tags: []string{"i:foo"}},
		{in: "words # #: :#: #::: #i:foo", tags: []string{"i:foo"}},
		{in: "#dog:o #y:far #a:rig #e:bar #i:foo", tags: []string{"dog:o", "y:far", "a:rig", "e:bar", "i:foo"}},
	}
	for _, c := range cases {
		assert.Equal(t, c.tags, harvestTags(c.in))
	}
}

func TestDetagOrig(t *testing.T) {
	cases := []struct {
		in   string
		tags []string
		out  string
	}{
		{in: "red alert #server:1234", tags: []string{"server:1234"}, out: "red alert"},
	}
	for _, c := range cases {
		assert.Equal(t, c.out, detagOrig(c.in, c.tags))
	}
}
