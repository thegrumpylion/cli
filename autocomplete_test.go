package cli

import (
	"fmt"
	"testing"
)

func TestAutocompleteFiles(t *testing.T) {
	fmt.Println(filesAutocompleter("./a"))
	fmt.Println(filesAutocompleter("bash/t"))
}

func TestAutocompleteHosts(t *testing.T) {
	fmt.Println(hostsAutocompleter("ip6-"))
}
