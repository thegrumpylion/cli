package cnc

import (
	"fmt"
	"sort"
	"testing"
)

func TestCompleteFiles(t *testing.T) {
	fmt.Println(filesCompleter("./a"))
	fmt.Println(filesCompleter("bash/t"))
}

func TestCompleteHosts(t *testing.T) {
	fmt.Println(hostsCompleter("ip6-"))
}

func TestCompleteFlags(t *testing.T) {
	s := []string{"--base", "-c", "--able", "-b", "-a", "--dive", "--card"}
	sort.Strings(s)
	fmt.Println(s)
}
