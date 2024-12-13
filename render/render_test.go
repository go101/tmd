package render

import "bytes"
import "testing"

type testCase struct {
	tmdData string

	fullHtml            bool
	supportCustomBlocks bool

	expectedMinOutputLength int
	expectedMaxOutputLength int // inclusive if not negative
}

func TestRenderer(t *testing.T) {
	renderer, err := NewRenderer()
	if err != nil {
		t.Fatal(err)
	}
	defer renderer.Destroy()

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
		htmlData, err := renderer.Render([]byte(testCase.tmdData), testCase.fullHtml, testCase.supportCustomBlocks)
		if err != nil {
			t.Fatalf("[test case %d]: render error: %s", i, err)
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
	}

	var tmdData = bytes.Repeat([]byte("foo "), 1<<18)
	var lastLength = -1
	for i := range 100 {
		output, err := renderer.Render(tmdData, true, true)
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
