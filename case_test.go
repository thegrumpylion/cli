package cli

import "testing"

func TestCase(t *testing.T) {
	type caseTest struct {
		in, out string
	}
	cases := map[Case][]caseTest{
		CaseNone: {
			caseTest{in: "What_Ever", out: "What_Ever"},
		},
		CaseLower: {
			caseTest{in: "wHaTeVeR", out: "whatever"},
		},
		CaseUpper: {
			caseTest{in: "wHaTeVeR", out: "WHATEVER"},
			caseTest{in: "wHaT_eVeR", out: "WHAT_EVER"},
		},
		CaseCamel: {
			caseTest{in: "WhatEver", out: "WhatEver"},
			caseTest{in: "whatEver", out: "WhatEver"},
			caseTest{in: "whatEverDB", out: "WhatEverDB"},
			caseTest{in: "whatEverDBConn", out: "WhatEverDBConn"},
		},
		CaseCamelLower: {
			caseTest{in: "WhatEver", out: "whatEver"},
			caseTest{in: "whatEver", out: "whatEver"},
			caseTest{in: "whatEverDB", out: "whatEverDB"},
		},
		CaseSnake: {
			caseTest{in: "WhatEver", out: "what_ever"},
			caseTest{in: "whatEverDB", out: "what_ever_db"},
			caseTest{in: "whatEverDBConn", out: "what_ever_db_conn"},
		},
		CaseSnakeUpper: {
			caseTest{in: "WhatEver", out: "WHAT_EVER"},
		},
		CaseKebab: {
			caseTest{in: "WhatEver", out: "what-ever"},
		},
		CaseKebabUpper: {
			caseTest{in: "WhatEver", out: "WHAT-EVER"},
		},
	}

	for c, tests := range cases {
		for _, tst := range tests {
			r := c.Parse(tst.in)
			if r != tst.out {
				t.Fatal(r, "not", tst.out, tst.in)
			}
		}
	}
}
