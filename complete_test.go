package cli

import (
	"bytes"
	"os"
	"testing"
)

func TestCompletion(t *testing.T) {

	cmd := &struct {
		SubcmdA *struct {
			Val string
		}
		SubcmdB *struct {
			Num int
		}
		Host string
		Port int
	}{}

	NewCommand("testcmd", cmd)

	cases := []struct {
		Name    string
		Cmdline string
		Point   string
		Expect  string
	}{
		{
			"subcommands",
			"testcmd ",
			"8",
			"subcmda \nsubcmdb \n",
		},
		{
			"flags",
			"testcmd --",
			"10",
			"--host \n--port \n",
		},
		{
			"suba",
			"testcmd subcmda ",
			"16",
			"--val \n",
		},
		{
			"subb",
			"testcmd subcmdb ",
			"16",
			"--num \n",
		},
	}

	for _, c := range cases {
		os.Setenv("COMP_LINE", c.Cmdline)
		os.Setenv("COMP_POINT", c.Point)
		t.Run(c.Name, func(t *testing.T) {
			buf := &bytes.Buffer{}
			defaultCLI.completeOut = buf
			defaultCLI.osExit = func(i int) {
				if buf.String() != c.Expect {
					t.Fatal("wrong autocompletion", buf.String())
				}
				t.Skip("correct completion", buf.String())
			}
			Parse([]string{"testcmd"})
			t.Fatal("should not be reached in completion")
		})
	}

}
