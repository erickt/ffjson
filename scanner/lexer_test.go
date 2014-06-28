/**
 *  Copyright 2014 Paul Querna
 *
 *  Licensed under the Apache License, Version 2.0 (the "License");
 *  you may not use this file except in compliance with the License.
 *  You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 *  Unless required by applicable law or agreed to in writing, software
 *  distributed under the License is distributed on an "AS IS" BASIS,
 *  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *  See the License for the specific language governing permissions and
 *  limitations under the License.
 *
 */

package scanner

import (
	"bytes"
	"fmt"
	"testing"
)

func scanAll(ffl *FFLexer) []FFTok {
	rv := make([]FFTok, 0, 0)
	for {
		tok := ffl.Scan()
		rv = append(rv, tok)
		if tok == FFTok_eof || tok == FFTok_error {
			break
		}
	}

	return rv
}

func assertTokensEqual(t *testing.T, a []FFTok, b []FFTok) {

	if len(a) != len(b) {
		t.Error(fmt.Sprintf("Token lists of mixed length: expected=%v found=%v", a, b))
		return
	}

	for i, v := range a {
		if b[i] != v {
			t.Error(
				fmt.Sprintf("Invalid Token:  Expected %d Found %d at token %d",
					v, b, i))
			return
		}
	}
}

func TestBasicLexing(t *testing.T) {
	ffl := NewFFLexer(bytes.NewBufferString(`{}`))
	toks := scanAll(ffl)
	assertTokensEqual(t, []FFTok{
		FFTok_left_bracket,
		FFTok_right_bracket,
		FFTok_eof,
	}, toks)
}

func TestHelloWorld(t *testing.T) {
	ffl := NewFFLexer(bytes.NewBufferString(`{"hello":"world"}`))
	toks := scanAll(ffl)
	assertTokensEqual(t, []FFTok{
		FFTok_left_bracket,
		FFTok_string,
		FFTok_colon,
		FFTok_string,
		FFTok_right_bracket,
		FFTok_eof,
	}, toks)

	ffl = NewFFLexer(bytes.NewBufferString(`{"hello": 1}`))
	toks = scanAll(ffl)
	assertTokensEqual(t, []FFTok{
		FFTok_left_bracket,
		FFTok_string,
		FFTok_colon,
		FFTok_integer,
		FFTok_right_bracket,
		FFTok_eof,
	}, toks)
}
