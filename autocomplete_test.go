package cnc

import (
	"fmt"
	"sort"
	"testing"
)

func TestAutocompleteFiles(t *testing.T) {
	fmt.Println(filesAutocompleter("./a"))
	fmt.Println(filesAutocompleter("bash/t"))
}

func TestAutocompleteHosts(t *testing.T) {
	fmt.Println(hostsAutocompleter("ip6-"))
}

func TestAutocompleteFlags(t *testing.T) {
	s := []string{"--base", "-c", "--able", "-b", "-a", "--dive", "--card"}
	sort.Strings(s)
	fmt.Println(s)
}
