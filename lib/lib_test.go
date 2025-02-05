package lib

import "bytes"
import "testing"

type testCase struct {
	tmdData string

	fullHtml            bool
	supportCustomBlocks bool

	expectedMinOutputLength int
	expectedMaxOutputLength int // inclusive if not negative
}

func TestTmdLib(t *testing.T) {
	tmdLib, err := NewTmdLib()
	if err != nil {
		t.Fatal(err)
	}
	defer tmdLib.Destroy()

	testCases := []testCase{
		{
			tmdData:                 "",
			fullHtml:                false,
			supportCustomBlocks:     false,
			expectedMinOutputLength: 0,
			expectedMaxOutputLength: 0,
		},
		{
			tmdData:                 "",
			fullHtml:                false,
			supportCustomBlocks:     true,
			expectedMinOutputLength: 0,
			expectedMaxOutputLength: 0,
		},
		{
			tmdData:                 "",
			fullHtml:                true,
			supportCustomBlocks:     true,
			expectedMinOutputLength: 10,
			expectedMaxOutputLength: -1,
		},
		{
			tmdData: `"""html
			<textarea>foo bar</textarea>
			"""`,
			fullHtml:                false,
			supportCustomBlocks:     false,
			expectedMinOutputLength: 0,
			expectedMaxOutputLength: 0,
		},
		{
			tmdData: `"""html
			<textarea>foo bar</textarea>
			"""`,
			fullHtml:                false,
			supportCustomBlocks:     true,
			expectedMinOutputLength: 1,
			expectedMaxOutputLength: 1000,
		},
	}

	for i, testCase := range testCases {
		htmlData, err := tmdLib.GenerateHTML([]byte(testCase.tmdData), testCase.fullHtml, testCase.supportCustomBlocks)
		if err != nil {
			t.Fatalf("[test case %d]: generate HTML error: %s", i, err)
			return
		}
		if len(htmlData) < testCase.expectedMinOutputLength {
			t.Fatalf("[test case %d]: output is too short (%d < %d)", i, len(htmlData), testCase.expectedMinOutputLength)
			return
		}
		if testCase.expectedMaxOutputLength >= 0 && len(htmlData) > testCase.expectedMaxOutputLength {
			t.Fatalf("[test case %d]: output is too long (%d > %d)", i, len(htmlData), testCase.expectedMaxOutputLength)
			return
		}

		formatted, err := tmdLib.FormatTMD([]byte(testCase.tmdData))
		if err != nil {
			t.Fatalf("[test case %d]: format TMD error: %s", i, err)
			return
		}
		formatted = bytes.Clone(formatted) // ! old formatted will be overwritten
		if formatted != nil {
			htmlData2, err := tmdLib.GenerateHTML(formatted, testCase.fullHtml, testCase.supportCustomBlocks)
			if err != nil {
				t.Fatalf("[test case %d]: generate HTML error 2: %s", i, err)
				return
			}
			if !bytes.Equal(htmlData2, htmlData) {
				t.Fatalf("[test case %d]: different generated HTML", i)
				return
			}

			formatted2, err := tmdLib.FormatTMD(formatted)
			if err != nil {
				t.Fatalf("[test case %d]: format TMD error 2: %s", i, err)
				return
			}
			if formatted2 != nil {
				t.Fatalf("[test case %d]: format(formatted) != formatted\n%s\n%s\n", i, formatted, formatted2)
				return
			}
		}

	}

	var tmdData = bytes.Repeat([]byte("foo "), 1<<18)
	var lastLength = -1
	for i := range 100 {
		output, err := tmdLib.GenerateHTML(tmdData, true, true)
		if err != nil {
			t.Fatalf("[stress testing at step %d] error: %s", i, err)
			return
		}
		if lastLength < 0 {
			lastLength = len(output)
		} else if lastLength != len(output) {
			t.Fatalf("[stress testing at step %d] lastLength != len(output (%d != %d)", i, lastLength, len(output))
			return
		}

		if lastLength < 1<<20 {
			t.Fatalf("[stress testing at step %d] output is too short (%d < %d)", i, lastLength, 1<<20)
			return
		}
	}
}
