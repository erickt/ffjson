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

/* Portions of this file are on derived from yajl: <https://github.com/lloyd/yajl> */
/*
 * Copyright (c) 2007-2014, Lloyd Hilaiel <me@lloyd.io>
 *
 * Permission to use, copy, modify, and/or distribute this software for any
 * purpose with or without fee is hereby granted, provided that the above
 * copyright notice and this permission notice appear in all copies.
 *
 * THE SOFTWARE IS PROVIDED "AS IS" AND THE AUTHOR DISCLAIMS ALL WARRANTIES
 * WITH REGARD TO THIS SOFTWARE INCLUDING ALL IMPLIED WARRANTIES OF
 * MERCHANTABILITY AND FITNESS. IN NO EVENT SHALL THE AUTHOR BE LIABLE FOR
 * ANY SPECIAL, DIRECT, INDIRECT, OR CONSEQUENTIAL DAMAGES OR ANY DAMAGES
 * WHATSOEVER RESULTING FROM LOSS OF USE, DATA OR PROFITS, WHETHER IN AN
 * ACTION OF CONTRACT, NEGLIGENCE OR OTHER TORTIOUS ACTION, ARISING OUT OF
 * OR IN CONNECTION WITH THE USE OR PERFORMANCE OF THIS SOFTWARE.
 */

package scanner

import (
	"bufio"
	"fmt"
	"io"
)

type FFTok int

const (
	FFTok_init          FFTok = iota
	FFTok_bool          FFTok = iota
	FFTok_colon         FFTok = iota
	FFTok_comma         FFTok = iota
	FFTok_eof           FFTok = iota
	FFTok_error         FFTok = iota
	FFTok_left_brace    FFTok = iota
	FFTok_left_bracket  FFTok = iota
	FFTok_null          FFTok = iota
	FFTok_right_brace   FFTok = iota
	FFTok_right_bracket FFTok = iota

	/* we differentiate between integers and doubles to allow the
	 * parser to interpret the number without re-scanning */
	FFTok_integer FFTok = iota
	FFTok_double  FFTok = iota

	/* we differentiate between strings which require further processing,
	 * and strings that do not */
	FFTok_string              FFTok = iota
	FFTok_string_with_escapes FFTok = iota

	/* comment tokens are not currently returned to the parser, ever */
	FFTok_comment FFTok = iota
)

type FFErr int

const (
	FFErr_e_ok                           FFErr = iota
	FFErr_io                             FFErr = iota
	FFErr_string_invalid_utf8            FFErr = iota
	FFErr_string_invalid_escaped_char    FFErr = iota
	FFErr_string_invalid_json_char       FFErr = iota
	FFErr_string_invalid_hex_char        FFErr = iota
	FFErr_invalid_char                   FFErr = iota
	FFErr_invalid_string                 FFErr = iota
	FFErr_missing_integer_after_decimal  FFErr = iota
	FFErr_missing_integer_after_exponent FFErr = iota
	FFErr_missing_integer_after_minus    FFErr = iota
	FFErr_unallowed_comment              FFErr = iota
)

type FFLexer struct {
	reader        *bufio.Reader
	Token         FFTok
	Error         FFErr
	BigError      error
	CurrentOffset int
	CurrentLine   int
	CurrentChar   int
	Output        []byte
}

func NewFFLexer(r io.Reader) *FFLexer {
	return &FFLexer{
		Token:       FFTok_init,
		Error:       FFErr_e_ok,
		CurrentLine: 1,
		CurrentChar: 1,
		reader:      bufio.NewReader(r),
	}
}

func (ffl *FFLexer) readByte() (byte, error) {

	/*
		if ffl.empty() {
			ffl.gather()
			if ffl.Token != FFTok_init {
				goto lexed
			}
		}
	*/

	c, err := ffl.reader.ReadByte()
	if err != nil {
		ffl.Error = FFErr_io
		ffl.BigError = err
		return 0, err
	}

	ffl.CurrentOffset++

	return c, nil
}

func (ffl *FFLexer) unreadByte() error {
	err := ffl.reader.UnreadByte()
	if err != nil {
		ffl.Error = FFErr_io
		ffl.BigError = err
		return err
	}

	ffl.CurrentOffset--
	return nil
}

func (ffl *FFLexer) wantBytes(want []byte, iftrue FFTok) FFTok {
	for _, b := range want {
		c, err := ffl.readByte()

		if err != nil {
			return FFTok_error
		}

		if c != b {
			err = ffl.reader.UnreadByte()

			if err != nil {
				return FFTok_error
			}

			ffl.Error = FFErr_invalid_string
			return FFTok_error
		}

		// TODO: bytes.buffer? FIX THIS. rethink this.
		ffl.Output = append(ffl.Output, c)
	}

	return iftrue
}

func (ffl *FFLexer) lexString() FFTok {
	mask := IJC | NFP
	for {
		c, err := ffl.readByte()
		if err != nil {
			return FFTok_error
		}

		if charLookupTable[c]&mask == 0 {
			ffl.Output = append(ffl.Output, c)
			continue
		}

		if c == '"' {
			return FFTok_string
		}

		// TODO(pquerna): rest of string parsing.
		fmt.Printf("FFTok_error lexString char=%d\n", c)
		return FFTok_error
	}

	return FFTok_error
}

func (ffl *FFLexer) lexNumber() FFTok {
	tok := FFTok_integer

	c, err := ffl.readByte()
	if err != nil {
		return FFTok_error
	}

	/* optional leading minus */
	if c == '-' {
		ffl.Output = append(ffl.Output, c)
		c, err = ffl.readByte()
		if err != nil {
			return FFTok_error
		}
	}

	/* a single zero, or a series of integers */
	if c == '0' {
		ffl.Output = append(ffl.Output, c)
		c, err = ffl.readByte()
		if err != nil {
			return FFTok_error
		}
	} else if c >= '1' && c <= '9' {
		ffl.Output = append(ffl.Output, c)
		for {
			if c >= '0' && c <= '9' {
				c, err = ffl.readByte()
				if err != nil {
					return FFTok_error
				}
				ffl.Output = append(ffl.Output, c)
			} else {
				break
			}
		}
	} else {
		err = ffl.unreadByte()
		if err != nil {
			return FFTok_error
		}

		// yajl_lex_missing_integer_after_minus
		return FFTok_error
	}

	if c == '.' {
		// TODO(pquerna): handle floats
		var numRead int = 0
		ffl.Output = append(ffl.Output, c)
		c, err = ffl.readByte()
		if err != nil {
			return FFTok_error
		}

		for c >= '0' && c <= '9' {
			numRead++
			ffl.Output = append(ffl.Output, c)
			c, err = ffl.readByte()
			if err != nil {
				return FFTok_error
			}
		}

		if numRead == 0 {
			err = ffl.unreadByte()
			if err != nil {
				return FFTok_error
			}

			// yajl_lex_missing_integer_after_decimal
			return FFTok_error
		}

		tok = FFTok_double
	}

	err = ffl.unreadByte()
	if err != nil {
		return FFTok_error
	}

	return tok
}

var true_bytes = []byte{'r', 'u', 'e'}
var false_bytes = []byte{'a', 'l', 's', 'e'}
var null_bytes = []byte{'u', 'l', 'l'}

func (ffl *FFLexer) Scan() FFTok {
	tok := FFTok_error
	var startOffset = 0
	ffl.Output = make([]byte, 0, 0)
	ffl.Token = FFTok_init

	for {
		c, err := ffl.readByte()
		if err != nil {
			if err == io.EOF {
				return FFTok_eof
			} else {
				return FFTok_error
			}
		}

		switch c {
		case '{':
			tok = FFTok_left_bracket
			goto lexed
		case '}':
			tok = FFTok_right_bracket
			goto lexed
		case '[':
			tok = FFTok_left_brace
			goto lexed
		case ']':
			tok = FFTok_right_brace
			goto lexed
		case ',':
			tok = FFTok_comma
			goto lexed
		case ':':
			tok = FFTok_colon
			goto lexed
		case '\t', '\n', '\v', '\f', '\r', ' ':
			startOffset++
			break
		case 't':
			ffl.Output = append(ffl.Output, 't')
			tok = ffl.wantBytes(true_bytes, FFTok_bool)
			goto lexed
		case 'f':
			ffl.Output = append(ffl.Output, 'f')
			tok = ffl.wantBytes(false_bytes, FFTok_bool)
			goto lexed
		case 'n':
			ffl.Output = append(ffl.Output, 'n')
			tok = ffl.wantBytes(null_bytes, FFTok_null)
			goto lexed
		case '"':
			tok = ffl.lexString()
			goto lexed
		case '-', '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
			err = ffl.unreadByte()
			if err != nil {
				return FFTok_error
			}
			tok = ffl.lexNumber()
			goto lexed
		case '/':
			//tok = ffl.lexComent()
			goto lexed
		default:
			tok = FFTok_error
			ffl.Error = FFErr_invalid_char
		}
	}

lexed:
	ffl.Token = tok
	return tok
}

/* a lookup table which lets us quickly determine three things:
 * VEC - valid escaped control char
 * note.  the solidus '/' may be escaped or not.
 * IJC - invalid json char
 * VHC - valid hex char
 * NFP - needs further processing (from a string scanning perspective)
 * NUC - needs utf8 checking when enabled (from a string scanning perspective)
 */

const (
	VEC = 0x01
	IJC = 0x02
	VHC = 0x04
	NFP = 0x08
	NUC = 0x10
)

var charLookupTable map[byte]int = map[byte]int{
	0:   IJC,
	1:   IJC,
	2:   IJC,
	3:   IJC,
	4:   IJC,
	5:   IJC,
	6:   IJC,
	7:   IJC,
	8:   IJC,
	9:   IJC,
	10:  IJC,
	11:  IJC,
	12:  IJC,
	13:  IJC,
	14:  IJC,
	15:  IJC,
	16:  IJC,
	17:  IJC,
	18:  IJC,
	19:  IJC,
	20:  IJC,
	21:  IJC,
	22:  IJC,
	23:  IJC,
	24:  IJC,
	25:  IJC,
	26:  IJC,
	27:  IJC,
	28:  IJC,
	29:  IJC,
	30:  IJC,
	31:  IJC,
	32:  0,
	33:  0,
	34:  NFP | VEC | IJC,
	35:  0,
	36:  0,
	37:  0,
	38:  0,
	39:  0,
	40:  0,
	41:  0,
	42:  0,
	43:  0,
	44:  0,
	45:  0,
	46:  0,
	47:  VEC,
	48:  VHC,
	49:  VHC,
	50:  VHC,
	51:  VHC,
	52:  VHC,
	53:  VHC,
	54:  VHC,
	55:  VHC,
	56:  VHC,
	57:  VHC,
	58:  0,
	59:  0,
	60:  0,
	61:  0,
	62:  0,
	63:  0,
	64:  0,
	65:  VHC,
	66:  VHC,
	67:  VHC,
	68:  VHC,
	69:  VHC,
	70:  VHC,
	71:  0,
	72:  0,
	73:  0,
	74:  0,
	75:  0,
	76:  0,
	77:  0,
	78:  0,
	79:  0,
	80:  0,
	81:  0,
	82:  0,
	83:  0,
	84:  0,
	85:  0,
	86:  0,
	87:  0,
	88:  0,
	89:  0,
	90:  0,
	91:  0,
	92:  NFP | VEC | IJC,
	93:  0,
	94:  0,
	95:  0,
	96:  0,
	97:  VHC,
	98:  VEC | VHC,
	99:  VHC,
	100: VHC,
	101: VHC,
	102: VEC | VHC,
	103: 0,
	104: 0,
	105: 0,
	106: 0,
	107: 0,
	108: 0,
	109: 0,
	110: VEC,
	111: 0,
	112: 0,
	113: 0,
	114: VEC,
	115: 0,
	116: VEC,
	117: 0,
	118: 0,
	119: 0,
	120: 0,
	121: 0,
	122: 0,
	123: 0,
	124: 0,
	125: 0,
	126: 0,
	127: 0,
	128: NUC,
	129: NUC,
	130: NUC,
	131: NUC,
	132: NUC,
	133: NUC,
	134: NUC,
	135: NUC,
	136: NUC,
	137: NUC,
	138: NUC,
	139: NUC,
	140: NUC,
	141: NUC,
	142: NUC,
	143: NUC,
	144: NUC,
	145: NUC,
	146: NUC,
	147: NUC,
	148: NUC,
	149: NUC,
	150: NUC,
	151: NUC,
	152: NUC,
	153: NUC,
	154: NUC,
	155: NUC,
	156: NUC,
	157: NUC,
	158: NUC,
	159: NUC,
	160: NUC,
	161: NUC,
	162: NUC,
	163: NUC,
	164: NUC,
	165: NUC,
	166: NUC,
	167: NUC,
	168: NUC,
	169: NUC,
	170: NUC,
	171: NUC,
	172: NUC,
	173: NUC,
	174: NUC,
	175: NUC,
	176: NUC,
	177: NUC,
	178: NUC,
	179: NUC,
	180: NUC,
	181: NUC,
	182: NUC,
	183: NUC,
	184: NUC,
	185: NUC,
	186: NUC,
	187: NUC,
	188: NUC,
	189: NUC,
	190: NUC,
	191: NUC,
	192: NUC,
	193: NUC,
	194: NUC,
	195: NUC,
	196: NUC,
	197: NUC,
	198: NUC,
	199: NUC,
	200: NUC,
	201: NUC,
	202: NUC,
	203: NUC,
	204: NUC,
	205: NUC,
	206: NUC,
	207: NUC,
	208: NUC,
	209: NUC,
	210: NUC,
	211: NUC,
	212: NUC,
	213: NUC,
	214: NUC,
	215: NUC,
	216: NUC,
	217: NUC,
	218: NUC,
	219: NUC,
	220: NUC,
	221: NUC,
	222: NUC,
	223: NUC,
	224: NUC,
	225: NUC,
	226: NUC,
	227: NUC,
	228: NUC,
	229: NUC,
	230: NUC,
	231: NUC,
	232: NUC,
	233: NUC,
	234: NUC,
	235: NUC,
	236: NUC,
	237: NUC,
	238: NUC,
	239: NUC,
	240: NUC,
	241: NUC,
	242: NUC,
	243: NUC,
	244: NUC,
	245: NUC,
	246: NUC,
	247: NUC,
	248: NUC,
	249: NUC,
	250: NUC,
	251: NUC,
	252: NUC,
	253: NUC,
	254: NUC,
	255: NUC,
}
