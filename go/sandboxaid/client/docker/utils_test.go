package docker

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_parseEnvKeyVal(t *testing.T) {
	cases := []struct {
		input  string
		expKey string
		expVal string
	}{
		{
			input:  "",
			expKey: "",
			expVal: "",
		},
		{
			input:  "FOO=bar",
			expKey: "FOO",
			expVal: "bar",
		},
		{
			input:  "A=B=c",
			expKey: "A",
			expVal: "B=c",
		},
	}

	for _, c := range cases {
		t.Run(c.input, func(t *testing.T) {
			key, val := parseEnvKeyVal(c.input)
			require.Equal(t, c.expKey, key, "key")
			require.Equal(t, c.expVal, val, "val")
		})
	}
}
