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

/* Portions of this file are on Go stdlib's encoding/json/encode.go */
// Copyright 2010 The Go Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package pills

import (
	"bytes"
	"unicode/utf8"
)

/**
 * Function ported from encoding/json: func (e *encodeState) string(s string) (int, error)
 */
func WriteJsonString(buf *bytes.Buffer, s string) error {
	const hex = "0123456789abcdef"

	err := buf.WriteByte('"')
  if err != nil {
		return err
	}
	start := 0
	for i := 0; i < len(s); {
		if b := s[i]; b < utf8.RuneSelf {
			if 0x20 <= b && b != '\\' && b != '"' && b != '<' && b != '>' && b != '&' {
				i++
				continue
			}
			if start < i {
				_, err = buf.WriteString(s[start:i])
				if err != nil {
					return err
				}
			}
			switch b {
			case '\\', '"':
				err = buf.WriteByte('\\')
				if err != nil {
					return err
				}

				err = buf.WriteByte(b)
				if err != nil {
					return err
				}
			case '\n':
				err = buf.WriteByte('\\')
				if err != nil {
					return err
				}
				err = buf.WriteByte('n')
				if err != nil {
					return err
				}
			case '\r':
				err = buf.WriteByte('\\')
				if err != nil {
					return err
				}
				err = buf.WriteByte('r')
				if err != nil {
					return err
				}
			default:
				// This encodes bytes < 0x20 except for \n and \r,
				// as well as < and >. The latter are escaped because they
				// can lead to security holes when user-controlled strings
				// are rendered into JSON and served to some browsers.
				_, err = buf.WriteString(`\u00`)
				if err != nil {
					return err
				}
				err = buf.WriteByte(hex[b>>4])
				if err != nil {
					return err
				}
				err = buf.WriteByte(hex[b&0xF])
				if err != nil {
					return err
				}
			}
			i++
			start = i
			continue
		}
		c, size := utf8.DecodeRuneInString(s[i:])
		if c == utf8.RuneError && size == 1 {
			if start < i {
				_, err = buf.WriteString(s[start:i])
				if err != nil {
					return err
				}
			}
			_, err = buf.WriteString(`\ufffd`)
			if err != nil {
				return err
			}
			i += size
			start = i
			continue
		}
		// U+2028 is LINE SEPARATOR.
		// U+2029 is PARAGRAPH SEPARATOR.
		// They are both technically valid characters in JSON strings,
		// but don't work in JSONP, which has to be evaluated as JavaScript,
		// and can lead to security holes there. It is valid JSON to
		// escape them, so we do so unconditionally.
		// See http://timelessrepo.com/json-isnt-a-javascript-subset for discussion.
		if c == '\u2028' || c == '\u2029' {
			if start < i {
				_, err = buf.WriteString(s[start:i])
				if err != nil {
					return err
				}
			}
			_, err = buf.WriteString(`\u202`)
			if err != nil {
				return err
			}
			err = buf.WriteByte(hex[c&0xF])
			if err != nil {
				return err
			}
			i += size
			start = i
			continue
		}
		i += size
	}
	if start < len(s) {
		_, err = buf.WriteString(s[start:])
		if err != nil {
			return err
		}
	}
	err = buf.WriteByte('"')
	if err != nil {
		return err
	}
	return nil
}
