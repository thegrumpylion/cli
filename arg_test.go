package cnc

import (
	"fmt"
	"reflect"
	"testing"
)

func TestArgUsage(t *testing.T) {
	bl := &argument{
		long:  "--help",
		short: "-h",
		typ:   reflect.TypeOf(true),
	}

	str := &argument{
		long:        "--namespace",
		short:       "-n",
		typ:         reflect.TypeOf(""),
		placeholder: "NS",
		separator:   ' ',
	}

	fmt.Println(bl.Usage())
	fmt.Println(str.Usage())
}
