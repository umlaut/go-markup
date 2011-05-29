package markup

import (
	"bytes"
	"fmt"
)

const (
	MKDA_NOT_AUTOLINK = iota /* used internally when it is not an autolink*/
	MKDA_NORMAL              /* normal http/http/ftp/mailto/etc link */
	MKDA_EMAIL               /* e-mail link without explit mailto: */
)

const (
	BUFFER_BLOCK = iota
	BUFFER_SPAN
)

const (
	MKD_LI_END = 8
)

type LinkRef struct {
	id    []byte
	link  []byte
	title []byte
}

const (
	MD_CHAR_NONE = iota
	MD_CHAR_EMPHASIS
	MD_CHAR_CODESPAN
	MD_CHAR_LINEBREAK
	MD_CHAR_LINK
	MD_CHAR_LANGLE
	MD_CHAR_ESCAPE
	MD_CHAR_ENTITITY
	MD_CHAR_AUTOLINK
)

type TriggerFunc func(ob *bytes.Buffer, rndr *render, data []byte, offset int) int

var markdown_char_ptrs []TriggerFunc = []TriggerFunc{nil, nil, nil, nil, nil, nil, nil, nil, nil}

func init_markdown_char_ptrs() {
	markdown_char_ptrs[MD_CHAR_EMPHASIS] = char_emphasis
	markdown_char_ptrs[MD_CHAR_CODESPAN] = char_codespan
	markdown_char_ptrs[MD_CHAR_LINEBREAK] = char_linebreak
	markdown_char_ptrs[MD_CHAR_LINK] = char_link
	markdown_char_ptrs[MD_CHAR_LANGLE] = char_langle_tag
	markdown_char_ptrs[MD_CHAR_ESCAPE] = char_escape
	markdown_char_ptrs[MD_CHAR_ENTITITY] = char_entity
	markdown_char_ptrs[MD_CHAR_AUTOLINK] = char_autolink
}

type render struct {
	make        *mkd_renderer
	refs        []*LinkRef
	active_char [256]byte
	work_bufs   [2][]*bytes.Buffer // indexed by BUFFER_BLOCK or BUFFER_SPAN
	ext_flags   uint
	max_nesting int
}

var funcNestLevel int = 0

func spaces(n int) string {
	r := []byte("                                                                   ")[:n]
	return string(r)
}

var dolog bool = false

func trace(s string, args ...string) string {
	funcNestLevel++
	if !dolog {
		return s
	}
	sp := spaces(funcNestLevel * 2 - 2)
	if len(args) > 0 {
		fmt.Printf("%s%s(%s)\n", sp, s, args[0])
	} else {
		fmt.Printf("%s%s()\n", sp, s)
	}
	return s
}

func un(s string) {
	funcNestLevel--
	//sp := spaces(funcNestLevel)	
	//fmt.Printf("%s%s()\n", sp, s)
}

func (rndr *render) newbuf(bufType int) (buf *bytes.Buffer) {
	//defer un(trace("newbuf"))

	buf = new(bytes.Buffer)
	rndr.work_bufs[bufType] = append(rndr.work_bufs[bufType], buf)
	return buf
}

func (rndr *render) popbuf(bufType int) {
	//defer un(trace("popbuf"))
	rndr.work_bufs[bufType] = rndr.work_bufs[bufType][0 : len(rndr.work_bufs[bufType])-1]
}

var block_tags [][]byte = [][]byte{[]byte("p"), []byte("dl"), []byte("h1"), []byte("h2"), []byte("h3"), []byte("h4"), []byte("h5"), []byte("h6"), []byte("ol"), []byte("ul"), []byte("del"), []byte("div"), []byte("ins"), []byte("pre"), []byte("form"), []byte("math"), []byte("table"), []byte("iframe"), []byte("script"), []byte("fieldset"), []byte("noscript"), []byte("blockquote")}


var INS_TAG []byte = []byte("ins")
var DEL_TAG []byte = []byte("del")

/***************************
 * HELPER FUNCTIONS *
 ***************************/
func is_safe_link(link []byte) bool {
	valid_uris := [4]string{"http://", "https://", "ftp://", "mailto://"}

	for i := 0; i < 4; i++ {
		uri := []byte(valid_uris[i])
		if bytes.HasPrefix(link, uri) && isalnum(link[len(uri)]) {
			return true
		}
	}
	return false
}

func unscape_text(ob *bytes.Buffer, src []byte) {
	i := 0
	size := len(src)
	for i < size {
		org := i
		for i < size && src[i] != '\\' {
			i++
		}

		if i > org {
			ob.Write(src[org:i])
		}

		if i+1 >= size {
			break
		}
		ob.WriteByte(src[i+1])
		i += 2
	}
}

/* returns the current block tag */
/* TODO: speed it up by auto-generated optimized 
   comparison function that is a chain of ifs */
func find_block_tag(data []byte) (ret []byte) {
	defer un(trace("find_block_tag"))
	i := 0
	size := len(data)

	/* looking for the word end */
	for i < size && isalnum(data[i]) {
		i++
	}
	if i == 0 || i >= size {
		return
	}
	s := bytes.ToLower(data[:i])
	for _, tag := range block_tags {
		if bytes.Equal(s, tag) {
			return s
		}
	}
	return
}

/****************************
 * INLINE PARSING FUNCTIONS *
 ****************************/

/* looks for the address part of a mail autolink and '>' */
/* this is less strict than the original markdown e-mail address matching */
func is_mail_autolink(data []byte) int {
	size := len(data)
	nb := 0

	/* address is assumed to be: [-@._a-zA-Z0-9]+ with exactly one '@' */
	for i := 0; i < size; i++ {
		if isalnum(data[i]) {
			continue
		}
		c := data[i]
		if c == '@' {
			nb++
		} else if c == '-' || c == '.' || c == '_' {
			// do nothing
		} else if c == '>' {
			if nb == 1 {
				return i + 1
			}
			return 0
		} else {
			return 0
		}
	}
	return 0
}

/* returns the length of the given tag, or 0 is it's not valid */
func tag_length(data []byte, autolink *int) int {
	size := len(data)
	/* a valid tag can't be shorter than 3 chars */
	if size < 3 {
		return 0
	}

	/* begins with a '<' optionally followed by '/', followed by letter or number */
	if data[0] != '<' {
		return 0
	}
	i := 1
	if data[1] == '/' {
		i = 2
	}

	if !isalnum(data[i]) {
		return 0
	}

	/* scheme test */
	*autolink = MKDA_NOT_AUTOLINK

	/* try to find the beginning of an URI */
	for i < size && (isalnum(data[i]) || data[i] == '.' || data[i] == '+' || data[i] == '-') {
		i++
	}

	if i > 1 && data[i] == '@' {
		if j := is_mail_autolink(data[i:]); j != 0 {
			*autolink = MKDA_EMAIL
			return i + j
		}
	}

	if i > 2 && data[i] == ':' {
		*autolink = MKDA_NORMAL
		i++
	}

	/* completing autolink test: no whitespace or ' or " */
	if i >= size {
		*autolink = MKDA_NOT_AUTOLINK
	} else if *autolink == MKDA_NOT_AUTOLINK {
		j := i

		for i < size {
			if data[i] == '\\' {
				i += 2
			} else if data[i] == '>' || data[i] == '\'' || data[i] == '"' || isspace(data[i]) {
				break
			} else {
				i += 1
			}
		}

		if i >= size {
			return 0
		}
		if i > j && data[i] == '>' {
			return i + 1
		}
		/* one of the forbidden chars has been found */
		*autolink = MKDA_NOT_AUTOLINK
	}

	/* looking for sometinhg looking like a tag end */
	for i < size && data[i] != '>' {
		i += 1
	}
	if i >= size {
		return 0
	}
	return i + 1
}

/* parses inline markdown elements */
func parse_inline(ob *bytes.Buffer, rndr *render, data []byte) {
	defer un(trace("parse_inline"))
	size := len(data)
	if rndr.reachedNestingLimit() {
		return
	}

	var action byte = 0
	i := 0
	end := 0
	for i < size {
		/* copying inactive chars into the output */
		for end < size {
			action = rndr.active_char[data[end]]
			if action != 0 {
				break
			}
			end++
		}

		if rndr.make.normal_text != nil {
			work_data := data[i:end]
			rndr.make.normal_text(ob, work_data, rndr.make.opaque)
		} else {
			ob.Write(data[i:end])
		}

		if end >= size {
			break
		}
		i = end

		/* calling the trigger */
		actfunc := markdown_char_ptrs[action]
		end = actfunc(ob, rndr, data, i) // Note: unlike upskirt, we pass data, not data[i:]
		if 0 == end {
			/* no action from the callback */
			end = i + 1
		} else {
			i += end
			end = i
		}
	}
}

/* looks for the next emph char, skipping other constructs */
func find_emph_char(data []byte, c byte) int {
	defer un(trace("find_emph_char"))
	size := len(data)
	i := 1

	for i < size {
		for i < size && data[i] != c && data[i] != '`' && data[i] != '[' {
			i += 1
		}
		if i >= size {
			return 0
		}

		if data[i] == c {
			return i
		}

		/* not counting escaped chars */
		if i > 0 && data[i-1] == '\\' { // i > 0 probably not necessary
			i += 1
			continue
		}

		/* skipping a code span */
		if data[i] == '`' {
			tmp_i := 0
			i += 1
			for i < size && data[i] != '`' {
				if 0 == tmp_i && data[i] == c {
					tmp_i = i
				}
				i += 1
			}
			if i >= size {
				return tmp_i
			}
			i += 1
		} else if data[i] == '[' {
			/* skipping a link */
			tmp_i := 0
			i += 1
			for i < size && data[i] != ']' {
				if 0 == tmp_i && data[i] == c {
					tmp_i = i
				}
				i += 1
			}
			i += 1
			for i < size && (data[i] == ' ' || data[i] == '\t' || data[i] == '\n') {
				i += 1
			}
			if i >= size {
				return tmp_i
			}
			if data[i] != '[' && data[i] != '(' { /* not a link*/
				if tmp_i > 0 {
					return tmp_i
				} else {
					continue
				}
			}
			cc := data[i]
			i += 1
			for i < size && data[i] != cc {
				if 0 == tmp_i && data[i] == c {
					tmp_i = i
				}
				i += 1
			}
			if i >= size {
				return tmp_i
			}
			i += 1
		}
	}
	return 0
}

/* parsing single emphase */
/* closed by a symbol not preceded by whitespace and not followed by symbol */
func parse_emph1(ob *bytes.Buffer, rndr *render, data []byte, c byte) int {
	defer un(trace("parse_emph1"))
	if nil == rndr.make.emphasis {
		return 0
	}

	size := len(data)
	i := 0

	/* skipping one symbol if coming from emph3 */
	if size > 1 && data[0] == c && data[1] == c {
		i = 1
	}

	for i < size {
		len := find_emph_char(data[i:], c)
		if 0 == len {
			return 0
		}
		i += len
		if i >= size {
			return 0
		}

		if i+1 < size && data[i+1] == c {
			i += 1
			continue
		}

		if data[i] == c && !isspace(data[i-1]) {
			if rndr.ext_flags&MKDEXT_NO_INTRA_EMPHASIS != 0 {
				if !(i+1 == size || isspace(data[i+1]) || ispunct(data[i+1])) {
					continue
				}
			}

			work := rndr.newbuf(BUFFER_SPAN)
			parse_inline(work, rndr, data[:i])
			r := rndr.make.emphasis(ob, work.Bytes(), rndr.make.opaque)
			rndr.popbuf(BUFFER_SPAN)
			if r {
				return i + 1
			} else {
				return 0
			}
		}
	}

	return 0
}

/* parsing single emphase */
func parse_emph2(ob *bytes.Buffer, rndr *render, data []byte, c byte) int {
	defer un(trace("parse_emph2"))
	var render_method rndrBufFunc_b
	size := len(data)
	if c == '~' {
		render_method = rndr.make.strikethrough
	} else {
		render_method = rndr.make.double_emphasis
	}

	if nil == render_method {
		return 0
	}

	for i := 0; i < size; i++ {
		len := find_emph_char(data[i:], c)
		if 0 == len {
			return 0
		}
		i += len

		if i+1 < size && data[i] == c && data[i+1] == c && i > 0 && !isspace(data[i-1]) {
			work := rndr.newbuf(BUFFER_SPAN)
			parse_inline(work, rndr, data[:i])
			r := render_method(ob, work.Bytes(), rndr.make.opaque)
			rndr.popbuf(BUFFER_SPAN)
			if r {
				return i + 2
			} else {
				return 0
			}
		}
		i++
	}
	return 0
}

/* parsing single emphase */
/* finds the first closing tag, and delegates to the other emph */
func parse_emph3(ob *bytes.Buffer, rndr *render, dataorig []byte, iorig int, c byte) int {
	defer un(trace("parse_emph3"))
	data := dataorig[iorig:]
	size := len(data)
	i := 0
	for i < size {
		len := find_emph_char(data[i:], c)
		if 0 == len {
			return 0
		}
		i += len

		/* skip whitespace preceded symbols */
		if data[i] != c || isspace(data[i-1]) {
			continue
		}

		if i+2 < size && data[i+1] == c && data[i+2] == c && nil != rndr.make.triple_emphasis {
			/* triple symbol found */
			work := rndr.newbuf(BUFFER_SPAN)
			parse_inline(work, rndr, data[:i])
			r := rndr.make.triple_emphasis(ob, work.Bytes(), rndr.make.opaque)
			rndr.popbuf(BUFFER_SPAN)
			if r {
				return i + 3
			} else {
				return 0
			}
		} else if i+1 < size && data[i+1] == c {
			/* double symbol found, handing over to emph1 */
			len = parse_emph1(ob, rndr, dataorig[iorig-2:], c)
			if 0 == len {
				return 0
			} else {
				return len - 2
			}
		} else {
			/* single symbol found, handing over to emph2 */
			len = parse_emph2(ob, rndr, dataorig[iorig-1:], c)
			if 0 == len {
				return 0
			} else {
				return len - 1
			}
		}
	}
	return 0
}

func char_emphasis(ob *bytes.Buffer, rndr *render, data []byte, offset int) int {
	defer un(trace("char_emphasis"))

	c := data[0]
	size := len(data)
	var ret int

	if size > 2 && data[1] != c {
		/* whitespace cannot follow an opening emphasis;
		 * strikethrough only takes two characters '~~' */
		if c == '~' || isspace(data[1]) {
			return 0
		}

		if ret = parse_emph1(ob, rndr, data[1:], c); ret == 0 {
			return 0
		}

		return ret + 1
	}

	if size > 3 && data[1] == c && data[2] != c {
		if isspace(data[2]) {
			return 0
		}

		if ret = parse_emph2(ob, rndr, data[2:], c); ret == 0 {
			return 0
		}

		return ret + 2
	}

	if size > 4 && data[1] == c && data[2] == c && data[3] != c {
		if c == '~' || isspace(data[3]) {
			return 0
		}
		if ret = parse_emph3(ob, rndr, data, 3, c); ret == 0 {
			return 0
		}

		return ret + 3
	}
	return 0
}

func char_linebreak(ob *bytes.Buffer, rndr *render, data []byte, offset int) int {
	defer un(trace("char_linebreak"))
	if offset < 2 || data[offset-1] != ' ' || data[offset-2] != ' ' {
		return 0
	}

	len := ob.Len()
	newlen := len
	obd := ob.Bytes()
	/* removing the last space from ob and rendering */
	for newlen := 0; newlen >= 0 && obd[newlen-1] == ' '; newlen-- {
		// do nothing
	}
	if newlen != len {
		ob.Truncate(newlen)
	}

	if rndr.make.linebreak(ob, rndr.make.opaque) {
		return 1
	}
	return 0
}

func char_codespan(ob *bytes.Buffer, rndr *render, data []byte, offset int) int {
	defer un(trace("char_codespan"))

	data = data[offset:]
	size := len(data)
	nb := 0

	/* counting the number of backticks in the delimiter */
	for nb < size && data[nb] == '`' {
		nb++
	}

	/* finding the next delimiter */
	i := 0
	end := 0
	for end = nb; end < size && i < nb; end++ {
		if data[end] == '`' {
			i++
		} else {
			i = 0
		}
	}

	if i < nb && end >= size {
		return 0 /* no matching delimiter */
	}

	/* trimming outside whitespaces */
	f_begin := nb
	for f_begin < end && (data[f_begin] == ' ' || data[f_begin] == '\t') {
		f_begin++
	}

	f_end := end - nb
	for f_end > nb && (data[f_end-1] == ' ' || data[f_end-1] == '\t') {
		f_end--
	}

	/* real code span */
	if f_begin < f_end {
		if !rndr.make.codespan(ob, data[f_begin:f_end], rndr.make.opaque) {
			end = 0
		}
	} else {
		if !rndr.make.codespan(ob, nil, rndr.make.opaque) {
			end = 0
		}
	}

	return end
}

func char_escape(ob *bytes.Buffer, rndr *render, data []byte, offset int) int {
	defer un(trace("char_escape"))
	escape_chars := []byte("\\`*_{}[]()#+-.!:|&<>")

	data = data[offset:]
	size := len(data)

	if size > 1 {
		if -1 == bytes.IndexByte(escape_chars, data[1]) {
			return 0
		}

		if nil != rndr.make.normal_text {
			rndr.make.normal_text(ob, data[1:2], rndr.make.opaque)
		} else {
			ob.WriteByte(data[1])
		}
	}

	return 2
}

func char_entity(ob *bytes.Buffer, rndr *render, data []byte, offset int) int {
	defer un(trace("char_entity"))

	data = data[offset:]
	size := len(data)
	end := 1

	if end < size && data[end] == '#' {
		end++
	}

	for end < size && isalnum(data[end]) {
		end++
	}

	if end < size && data[end] == ';' {
		end += 1 /* real entity */
	} else {
		return 0 /* lone '&' */
	}

	if rndr.make.entity != nil {
		rndr.make.entity(ob, data[:end], rndr.make.opaque)
	} else {
		ob.Write(data[:end])
	}

	return end
}

func char_langle_tag(ob *bytes.Buffer, rndr *render, data []byte, offset int) int {
	defer un(trace("char_langle_tag"))

	altype := MKDA_NOT_AUTOLINK
	end := tag_length(data, &altype)

	data = data[offset:]

	work := data[:end]
	ret := false

	if end > 2 {
		if (rndr.make.autolink != nil) && (altype != MKDA_NOT_AUTOLINK) {
			u_link := rndr.newbuf(BUFFER_SPAN)
			work := data[1 : end-1]
			unscape_text(u_link, work)
			ret = rndr.make.autolink(ob, u_link.Bytes(), altype, rndr.make.opaque)
			rndr.popbuf(BUFFER_SPAN)
		} else if rndr.make.raw_html_tag != nil {
			ret = rndr.make.raw_html_tag(ob, work, rndr.make.opaque)
		}
	}

	if !ret {
		return 0
	}
	return end
}

func char_autolink(ob *bytes.Buffer, rndr *render, data []byte, offset int) int {
	defer un(trace("char_autolink"))
	var copen byte

	if offset > 0 {
		if !isspace(data[offset-1]) && !ispunct(data[offset-1]) {
			return 0
		}
	}

	dataorig := data
	data = data[offset:]
	size := len(data)
	if !is_safe_link(data) {
		return 0
	}

	link_end := 0
	for link_end < size && !isspace(data[link_end]) {
		link_end++
	}

	/* Skip punctuation at the end of the link */
	if (data[link_end-1] == '.' || data[link_end-1] == ',' || data[link_end-1] == ';') && data[link_end-2] != '\\' {
		link_end--
	}

	/* See if the link finishes with a punctuation sign that can be closed. */
	switch data[link_end-1] {
	case '"':
		copen = '"'
	case '\'':
		copen = '\''
	case ')':
		copen = '('
	case ']':
		copen = '['
	case '}':
		copen = '{'
	}

	if copen != 0 {
		buf_start_idx := 0
		buf_end_idx := offset + link_end - 2

		open_delim := 1

		/* Try to close the final punctuation sign in this same line;
		 * if we managed to close it outside of the URL, that means that it's
		 * not part of the URL. If it closes inside the URL, that means it
		 * is part of the URL.
		 *
		 * Examples:
		 *
		 *	foo http://www.pokemon.com/Pikachu_(Electric) bar
		 *		=> http://www.pokemon.com/Pikachu_(Electric)
		 *
		 *	foo (http://www.pokemon.com/Pikachu_(Electric)) bar
		 *		=> http://www.pokemon.com/Pikachu_(Electric)
		 *
		 *	foo http://www.pokemon.com/Pikachu_(Electric)) bar
		 *		=> http://www.pokemon.com/Pikachu_(Electric))
		 *
		 *	(foo http://www.pokemon.com/Pikachu_(Electric)) bar
		 *		=> foo http://www.pokemon.com/Pikachu_(Electric)
		 */

		for buf_end_idx >= buf_start_idx && dataorig[buf_end_idx] != '\n' && open_delim == 0 {
			if dataorig[buf_end_idx] == data[link_end-1] {
				open_delim++
			}

			if dataorig[buf_end_idx] == copen {
				open_delim--
			}

			buf_end_idx--
		}

		if open_delim == 0 {
			link_end--
		}
	}

	work := data[:link_end]

	if rndr.make.autolink != nil {
		u_link := rndr.newbuf(BUFFER_SPAN)
		unscape_text(u_link, work)

		rndr.make.autolink(ob, u_link.Bytes(), MKDA_NORMAL, rndr.make.opaque)
		rndr.popbuf(BUFFER_SPAN)
	}

	return len(work)
}

func (rndr *render) find_ref(id []byte) *LinkRef {
	//TODO: write me
	//arr_sorted_find(&rndr->refs, &id, cmp_link_ref)
	return nil
}

/* '[': parsing a link or an image */
func char_link(ob *bytes.Buffer, rndr *render, data []byte, offset int) int {
	defer un(trace("char_link"))
	is_img := false

	if offset > 0 && data[offset-1] == '!' {
		is_img = true
	}

	var content *bytes.Buffer
	var title *bytes.Buffer
	var u_link *bytes.Buffer

	org_work_size := len(rndr.work_bufs[BUFFER_SPAN])

	ret := false

	/* checking whether the correct renderer exists */
	if (is_img && nil != rndr.make.image) || (!is_img && nil != rndr.make.link) {
		goto cleanup
	}

	data = data[offset:]
	size := len(data)

	var link *bytes.Buffer
	i := 1
	text_has_nl := false
	/* looking for the matching closing bracket */
	for level := 1; i < size; i += 1 {
		if data[i] == '\n' {
			text_has_nl = true
		} else if data[i-1] == '\\' {
			continue
		} else if data[i] == '[' {
			level++
		} else if data[i] == ']' {
			level--
			if level <= 0 {
				break
			}
		}
	}

	if i >= size {
		goto cleanup
	}

	txt_e := i
	i += 1

	/* skip any amount of whitespace or newline */
	/* (this is much more laxist than original markdown syntax) */
	for i < size && isspace(data[i]) {
		i++
	}

	link_b := 0
	link_e := 0
	title_b := 0
	title_e := 0

	/* inline style link */
	if i < size && data[i] == '(' {
		/* skipping initial whitespace */
		i += 1

		for i < size && isspace(data[i]) {
			i++
		}

		link_b = i

		/* looking for link end: ' " ) */
		for i < size {
			if data[i] == '\\' {
				i += 2
			} else if data[i] == ')' || data[i] == '\'' || data[i] == '"' {
				break
			} else {
				i += 1
			}
		}

		if i >= size {
			goto cleanup
		}
		link_e = i

		/* looking for title end if present */
		if data[i] == '\'' || data[i] == '"' {
			i++
			title_b = i

			for i < size {
				if data[i] == '\\' {
					i += 2
				} else if data[i] == ')' {
					break
				} else {
					i += 1
				}
			}

			if i >= size {
				goto cleanup
			}

			/* skipping whitespaces after title */
			title_e = i - 1
			for title_e > title_b && isspace(data[title_e]) {
				title_e--
			}

			/* checking for closing quote presence */
			if data[title_e] != '\'' && data[title_e] != '"' {
				title_b = 0
				title_e = 0
				link_e = i
			}
		}

		/* remove whitespace at the end of the link */
		for link_e > link_b && isspace(data[link_e-1]) {
			link_e--
		}

		/* remove optional angle brackets around the link */
		if data[link_b] == '<' {
			link_b++
		}
		if data[link_e-1] == '>' {
			link_e--
		}

		/* building escaped link and title */
		if link_e > link_b {
			link = rndr.newbuf(BUFFER_SPAN)
			link.Write(data[link_b:link_e])
		}

		if title_e > title_b {
			title = rndr.newbuf(BUFFER_SPAN)
			title.Write(data[title_b:title_e])
		}

		i++
	} else if i < size && data[i] == '[' {
		/* reference style link */
		var id []byte
		var lr *LinkRef

		/* looking for the id */
		i += 1
		link_b = i
		for i < size && data[i] != ']' {
			i++
		}
		if i >= size {
			goto cleanup
		}
		link_e = i

		/* finding the link_ref */
		if link_b == link_e {
			if text_has_nl {
				b := rndr.newbuf(BUFFER_SPAN)

				for j := 1; j < txt_e; j++ {
					if data[j] != '\n' {
						b.WriteByte(data[j])
					} else if data[j-1] != ' ' {
						b.WriteByte(' ')
					}
				}

				id = b.Bytes()
			} else {
				id = data[1:txt_e]
			}
		} else {
			id = data[link_b:link_e]
		}

		lr = rndr.find_ref(id)
		if nil == lr {
			goto cleanup
		}

		/* keeping link and title from link_ref */
		// TODO: not sure if that's righ
		link = bytes.NewBuffer(lr.link)
		title = bytes.NewBuffer(lr.title)
		i += 1
	} else {
		/* shortcut reference style link */
		var id []byte
		var lr *LinkRef

		/* crafting the id */
		if text_has_nl {
			b := rndr.newbuf(BUFFER_SPAN)
			for j := 1; j < txt_e; j++ {
				if data[j] != '\n' {
					b.WriteByte(data[j])
				} else if data[j-1] != ' ' {
					b.WriteByte(' ')
				}
			}

			id = b.Bytes()
		} else {
			id = data[1:txt_e]
		}

		/* finding the link_ref */
		lr = rndr.find_ref(id)
		if nil == lr {
			goto cleanup
		}

		/* keeping link and title from link_ref */
		// TODO: not sure if bytes.NewBuffer() is righ
		link = bytes.NewBuffer(lr.link)
		title = bytes.NewBuffer(lr.title)

		/* rewinding the whitespace */
		i = txt_e + 1
	}

	/* building content: img alt is escaped, link content is parsed */
	if txt_e > 1 {
		content = rndr.newbuf(BUFFER_SPAN)
		if is_img {
			content.Write(data[1:txt_e])
		} else {
			parse_inline(content, rndr, data[1:txt_e])
		}
	}

	if nil != link {
		u_link = rndr.newbuf(BUFFER_SPAN)
		unscape_text(u_link, link.Bytes())
	}

	/* calling the relevant rendering function */
	if is_img {
		// TODO:
		//if ob.Len() && ob->data[ob->size - 1] == '!') {
		//	ob->size -= 1
		//}

		ret = rndr.make.image(ob, u_link.Bytes(), title.Bytes(), content.Bytes(), rndr.make.opaque)
	} else {
		ret = rndr.make.link(ob, u_link.Bytes(), title.Bytes(), content.Bytes(), rndr.make.opaque)
	}

	/* cleanup */
cleanup:
	for org_work_size > len(rndr.work_bufs[BUFFER_SPAN]) {
		rndr.popbuf(BUFFER_SPAN)
	}
	if ret {
		return i
	}
	return 0
}

/*********************************
 * BLOCK-LEVEL PARSING FUNCTIONS *
 *********************************/

/* returns the line length when it is empty, 0 otherwise */
func is_empty(data []byte) int {
	//defer un(trace("is_empty"))
	var i int
	size := len(data)
	for i = 0; i < size && data[i] != '\n'; i++ {
		if data[i] != ' ' && data[i] != '\t' {
			return 0
		}
	}
	return i + 1
}

/* returns whether a line is a horizontal rule */
func is_hrule(data []byte) bool {
	//defer un(trace("is_hrule"))
	size := len(data)
	if size < 3 {
		return false
	}
	i := 0
	/* skipping initial spaces */
	for i < 3 && data[i] == ' ' {
		i++
	}

	/* looking at the hrule char */
	if i+2 >= size || (data[i] != '*' && data[i] != '-' && data[i] != '_') {
		return false
	}
	c := data[i]

	/* the whole line must be the char or whitespace */
	n := 0
	for i < size && data[i] != '\n' {
		if data[i] == c {
			n += 1
		} else if data[i] != ' ' && data[i] != '\t' {
			return false
		}
		i += 1
	}

	return n >= 3
}

/* check if a line is a code fence; return its size if it is */
func is_codefence(data []byte, syntax *[]byte) int {
	//defer un(trace("is_codefence"))
	size := len(data)
	i := 0
	n := 0
	/* skipping initial spaces */
	if size < 3 {
		return 0
	}

	for i = 0; i < 3 && data[i] == ' '; i++ {
		// do nothing
	}

	/* looking at the hrule char */
	if i+2 >= size || !(data[i] == '~' || data[i] == '`') {
		return 0
	}

	c := data[i]

	/* the whole line must be the char or whitespace */
	for i < size && data[i] == c {
		n++
		i++
	}

	if n < 3 {
		return 0
	}

	if syntax != nil {
		syn := 0

		for i < size && (data[i] == ' ' || data[i] == '\t') {
			i++
		}

		*syntax = data[i:]

		if i < size && data[i] == '{' {
			i++
			*syntax = (*syntax)[1:]

			for i < size && data[i] != '}' && data[i] != '\n' {
				syn++
				i++
			}

			if i == size || data[i] != '}' {
				return 0
			}

			/* strip all whitespace at the beginning and the end
			 * of the {} block */
			for syn > 0 && isspace((*syntax)[0]) {
				*syntax = (*syntax)[1:]
				syn--
			}

			for syn > 0 && isspace((*syntax)[syn-1]) {
				syn--
			}

			i++
		} else {
			for i < size && !isspace(data[i]) {
				syn++
				i++
			}
		}

		*syntax = (*syntax)[:syn] // TODO: hopefully right
	}

	for i < size && data[i] != '\n' {
		if !isspace(data[i]) {
			return 0
		}

		i++
	}

	return i + 1
}

/* returns whether the line is a hash-prefixed header */
func is_atxheader(rndr *render, data []byte) bool {
	//defer un(trace("is_atxheader"))
	if data[0] != '#' {
		return false
	}

	if rndr.ext_flags&MKDEXT_SPACE_HEADERS != 0 {
		level := 0
		size := len(data)
		for level < size && level < 6 && data[level] == '#' {
			level++
		}

		if level < size && data[level] != ' ' && data[level] != '\t' {
			return false
		}
	}
	return true
}

/* returns whether the line is a setext-style hdr underline */
func is_headerline(data []byte) int {
	//defer un(trace("is_headerline"))
	i := 0
	size := len(data)

	/* test of level 1 header */
	if data[i] == '=' {
		for i = 1; i < size && data[i] == '='; i++ {
			// do nothing
		}
		for i < size && (data[i] == ' ' || data[i] == '\t') {
			i++
		}
		if i >= size || data[i] == '\n' {
			return 1
		}
		return 0
	}

	/* test of level 2 header */
	if data[i] == '-' {
		for i = 1; i < size && data[i] == '-'; i++ {
			// do nothing
		}
		for i < size && (data[i] == ' ' || data[i] == '\t') {
			i++
		}
		if i >= size || data[i] == '\n' {
			return 2
		}
	}
	return 0
}

func skip_spaces(data []byte, max int) int {
	n := 0
	for n < max && n < len(data) && data[n] == ' ' {
		n++
	}
	return n
}
/* returns blockquote prefix length */
func prefix_quote(data []byte) int {
	//defer un(trace("prefix_quote"))
	size := len(data)
	i := skip_spaces(data, 3)
	if i < size && data[i] == '>' {
		if i+1 < size && (data[i+1] == ' ' || data[i+1] == '\t') {
			return i + 2
		} else {
			return i + 1
		}
	}
	return 0
}

/* returns prefix length for block code*/
func prefix_code(data []byte) int {
	//defer un(trace("prefix_oli"))
	size := len(data)
	if size > 0 && data[0] == '\t' {
		return 1
	}
	if size > 3 && data[0] == ' ' && data[1] == ' ' && data[2] == ' ' && data[3] == ' ' {
		return 4
	}
	return 0
}

/* returns ordered list item prefix */
func prefix_oli(data []byte) int {
	//defer un(trace("prefix_oli"))
	size := len(data)
	i := 0
	if i < size && data[i] == ' ' {
		i += 1
	}
	if i < size && data[i] == ' ' {
		i += 1
	}
	if i < size && data[i] == ' ' {
		i += 1
	}
	if i >= size || data[i] < '0' || data[i] > '9' {
		return 0
	}
	for i < size && data[i] >= '0' && data[i] <= '9' {
		i += 1
	}
	if i+1 >= size || data[i] != '.' || (data[i+1] != ' ' && data[i+1] != '\t') {
		return 0
	}
	return i + 2
}

/* returns ordered list item prefix */
func prefix_uli(data []byte) int {
	size := len(data)
	i := 0
	if i < size && data[i] == ' ' {
		i += 1
	}
	if i < size && data[i] == ' ' {
		i += 1
	}
	if i < size && data[i] == ' ' {
		i += 1
	}
	if i+1 >= size || (data[i] != '*' && data[i] != '+' && data[i] != '-') || (data[i+1] != ' ' && data[i+1] != '\t') {
		return 0
	}
	return i + 2
}

/* handles parsing of a blockquote fragment */
func parse_blockquote(ob *bytes.Buffer, rndr *render, data []byte) int {
	defer un(trace("parse_blockquote"))
	size := len(data)
	work_data := make([]byte, 0, len(data))
	beg := 0
	end := 0
	for beg < size {
		for end = beg + 1; end < size && data[end-1] != '\n'; end++ {
		}

		pre := prefix_quote(data[beg:end])

		if pre > 0 {
			beg += pre /* skipping prefix */
		} else if is_empty(data[beg:end]) > 0 && (end >= size || (prefix_quote(data[end:]) == 0 && is_empty(data[end:]) == 0)) {
			/* empty line followed by non-quote line */
			break
		}
		if beg < end {
			work_data = append(work_data, data[beg:end]...)
		}
		beg = end
	}

	out := rndr.newbuf(BUFFER_BLOCK)
	parse_block(out, rndr, work_data)
	if nil != rndr.make.blockquote {
		rndr.make.blockquote(ob, out.Bytes(), rndr.make.opaque)
	}

	rndr.popbuf(BUFFER_BLOCK)
	return end
}

/* handles parsing of a regular paragraph */
func parse_paragraph(ob *bytes.Buffer, rndr *render, data []byte) int {
	defer un(trace("parse_paragraph"))
	size := len(data)
	i := 0
	end := 0
	level := 0
	for i < size {
		for end = i + 1; end < size && data[end-1] != '\n'; end++ {
			/* empty */
		}

		if is_empty(data[i:]) > 0 {
			break
		}
		if level = is_headerline(data[i:]); level != 0 {
			break
		}

		if rndr.ext_flags&MKDEXT_LAX_HTML_BLOCKS != 0 {
			if data[i] == '<' && rndr.make.blockhtml != nil && parse_htmlblock(ob, rndr, data[i:], false) > 0 {
				end = i
				break
			}
		}

		if is_atxheader(rndr, data[i:]) || is_hrule(data[i:]) {
			end = i
			break
		}

		i = end
	}

	work_size := i
	if work_size > 0 && data[work_size-1] == '\n' {
		work_size--
	}

	work := data

	if 0 == level {
		tmp := rndr.newbuf(BUFFER_BLOCK)
		parse_inline(tmp, rndr, data[:work_size])
		if nil != rndr.make.paragraph {
			rndr.make.paragraph(ob, tmp.Bytes(), rndr.make.opaque)
		}
		rndr.popbuf(BUFFER_BLOCK)
	} else {
		if work_size > 0 {
			i = work_size
			work_size -= 1

			for work_size > 0 && data[work_size] != '\n' {
				work_size -= 1
			}

			beg := work_size + 1
			for work_size > 0 && data[work_size-1] == '\n' {
				work_size -= 1
			}

			if work_size > 0 {
				tmp := rndr.newbuf(BUFFER_BLOCK)
				parse_inline(tmp, rndr, work[:size])

				if rndr.make.paragraph != nil {
					rndr.make.paragraph(ob, tmp.Bytes(), rndr.make.opaque)
				}

				rndr.popbuf(BUFFER_BLOCK)
				work = work[beg:i]
			} else {
				work = work[:i]
			}
		}

		header_work := rndr.newbuf(BUFFER_SPAN)
		parse_inline(header_work, rndr, work)

		if nil != rndr.make.header {
			rndr.make.header(ob, header_work.Bytes(), level, rndr.make.opaque)
		}

		rndr.popbuf(BUFFER_SPAN)
	}
	return end
}

/* handles parsing of a block-level code fragment */
func parse_fencedcode(ob *bytes.Buffer, rndr *render, data []byte) int {
	defer un(trace("parse_fencedcode"))
	size := len(data)
	var lang []byte
	beg := is_codefence(data, &lang)
	if beg == 0 {
		return 0
	}
	end := 0
	work := rndr.newbuf(BUFFER_BLOCK)
	for beg < size {
		fence_end := is_codefence(data[beg:], nil)
		if fence_end != 0 {
			beg += fence_end
			break
		}

		for end = beg + 1; end < size && data[end-1] != '\n'; end += 1 {
			// do nothing
		}

		if beg < end {
			/* verbatim copy to the working buffer,
			escaping entities */
			if is_empty(data[beg:end]) > 0 {
				work.WriteByte('\n')
			} else {
				work.Write(data[beg:end])
			}
		}
		beg = end
	}

	/* TODO:
	if work.Len() > 0 && work[len(work) - 1] != '\n' {
		work.WriteByte('\n')
	}
	*/

	if nil != rndr.make.blockcode {
		rndr.make.blockcode(ob, work.Bytes(), lang, rndr.make.opaque)
	}

	rndr.popbuf(BUFFER_BLOCK)
	return beg
}

func parse_blockcode(ob *bytes.Buffer, rndr *render, data []byte) int {
	defer un(trace("parse_blockcode"))
	size := len(data)
	work := rndr.newbuf(BUFFER_BLOCK)
	beg := 0
	end := 0
	for beg < size {
		for end = beg + 1; end < size && data[end-1] != '\n'; end++ {
			// do nothing
		}
		pre := prefix_code(data[beg:end])

		if pre > 0 {
			beg += pre /* skipping prefix */
		} else if 0 == is_empty(data[beg:end]) {
			/* non-empty non-prefixed line breaks the pre */
			break
		}

		if beg < end {
			/* verbatim copy to the working buffer,
			escaping entities */
			if is_empty(data[beg:end]) > 0 {
				work.WriteByte('\n')
			} else {
				work.Write(data[beg:end])
			}
		}
		beg = end
	}

	/* TODO: port me
	while (work->size && work->data[work->size - 1] == '\n')
		work->size -= 1;
	*/

	work.WriteByte('\n')

	if rndr.make.blockcode != nil {
		var emptySlice []byte
		rndr.make.blockcode(ob, work.Bytes(), emptySlice, rndr.make.opaque)
	}
	rndr.popbuf(BUFFER_BLOCK)
	return beg
}

/* parse_listitem • parsing of a single list item */
/*	assuming initial prefix is already removed */
func parse_listitem(ob *bytes.Buffer, rndr *render, data []byte, flags *int) int {
	defer un(trace("parse_listitem"))
	size := len(data)
	orgpre := 0
	/* keeping track of the first indentation prefix */
	for orgpre < 3 && orgpre < size && data[orgpre] == ' ' {
		orgpre++
	}

	beg := prefix_uli(data)
	if 0 == beg {
		beg = prefix_oli(data)
	}

	if 0 == beg {
		return 0
	}

	/* skipping to the beginning of the following line */
	end := beg
	for end < size && data[end-1] != '\n' {
		end++
	}

	/* getting working buffers */
	work := rndr.newbuf(BUFFER_SPAN)
	inter := rndr.newbuf(BUFFER_SPAN)

	/* putting the first line into the working buffer */
	work.Write(data[beg:end])
	beg = end

	in_empty := false
	has_inside_empty := false
	sublist := 0
	/* process the following lines */
	for beg < size {
		end++

		for end < size && data[end-1] != '\n' {
			end++
		}

		/* process an empty line */
		if 0 == is_empty(data[beg:end]) {
			in_empty = true
			beg = end
			continue
		}

		/* calculating the indentation */
		i := 0
		for i < 4 && beg+i < end && data[beg+i] == ' ' {
			i++
		}

		pre := i
		if data[beg] == '\t' {
			i = 1
			pre = 8
		}

		/* checking for a new item */
		if (prefix_uli(data[beg+i:end]) > 0 && !is_hrule(data[beg+i:end])) || prefix_oli(data[beg+i:]) > 0 {
			if in_empty {
				has_inside_empty = true
			}
			if pre == orgpre { /* the following item must have */
				break /* the same indentation */
			}

			if 0 == sublist {
				sublist = work.Len()
			}
		} else if in_empty && i < 4 && data[beg] != '\t' {
			/* joining only indented stuff after empty lines */
			*flags |= MKD_LI_END
			break
		} else if in_empty {
			work.WriteByte('\n')
			has_inside_empty = true
		}

		in_empty = false

		/* adding the line without prefix into the working buffer */
		work.Write(data[beg+i : end-beg-i])
		beg = end
	}

	/* render of li contents */
	if has_inside_empty {
		*flags |= MKD_LI_BLOCK
	}

	if *flags&MKD_LI_BLOCK != 0 {
		/* intermediate render of block li */
		if sublist > 0 && sublist < work.Len() {
			parse_block(inter, rndr, work.Bytes()[:sublist])
			parse_block(inter, rndr, work.Bytes()[sublist:])
		} else {
			parse_block(inter, rndr, work.Bytes())
		}
	} else {
		/* intermediate render of inline li */
		if sublist > 0 && sublist < work.Len() {
			parse_inline(inter, rndr, work.Bytes()[:sublist])
			parse_block(inter, rndr, work.Bytes()[sublist:])
		} else {
			parse_inline(inter, rndr, work.Bytes())
		}
	}

	/* render of li itself */
	if nil != rndr.make.listitem {
		rndr.make.listitem(ob, inter.Bytes(), *flags, rndr.make.opaque)
	}
	rndr.popbuf(BUFFER_SPAN)
	rndr.popbuf(BUFFER_SPAN)
	return beg
}

/* parsing ordered or unordered list block */
func parse_list(ob *bytes.Buffer, rndr *render, data []byte, flags int) int {
	defer un(trace("parse_list"))
	size := len(data)
	i := 0
	work := rndr.newbuf(BUFFER_BLOCK)

	for i < size {
		j := parse_listitem(work, rndr, data[i:], &flags)
		i += j

		if 0 == j || (flags&MKD_LI_END == 0) {
			break
		}
	}

	if nil != rndr.make.list {
		rndr.make.list(ob, work.Bytes(), flags, rndr.make.opaque)
	}
	rndr.popbuf(BUFFER_BLOCK)
	return i
}

/* parsing of atx-style headers */
func parse_atxheader(ob *bytes.Buffer, rndr *render, data []byte) int {
	defer un(trace("parse_atxheader"))
	size := len(data)
	level := 0

	for level < size && level < 6 && data[level] == '#' {
		level++
	}

	i := 0
	for i = level; i < size && (data[i] == ' ' || data[i] == '\t'); i++ {
		// do nothing
	}

	end := 0
	for end = i; end < size && data[end] != '\n'; end++ {
		// do nothing
	}
	skip := end

	for end > 0 && data[end-1] == '#' {
		end--
	}

	for end > 0 && (data[end-1] == ' ' || data[end-1] == '\t') {
		end--
	}

	if end > i {
		work := rndr.newbuf(BUFFER_SPAN)
		parse_inline(work, rndr, data[i:end])
		if nil != rndr.make.header {
			rndr.make.header(ob, work.Bytes(), level, rndr.make.opaque)
		}
		rndr.popbuf(BUFFER_SPAN)
	}

	return skip
}

// checking end of HTML block : </tag>[ \t]*\n[ \t*]\n
// returns the length on match, 0 otherwise
func htmlblock_end(tag []byte, rndr *render, data []byte) int {
	defer un(trace("htmlblock_end"))
	size := len(data)
	tag_size := len(tag)

	/* assuming data[0] == '<' && data[1] == '/' already tested */

	/* checking if tag is a match */
	if tag_size+3 >= size || !bytes.HasPrefix(data[2:], tag) || data[tag_size+2] != '>' {
		return 0
	}

	/* checking white lines */
	i := tag_size + 3
	w := 0
	if i < size {
		if w = is_empty(data[i:]); w == 0 {
			return 0 /* non-blank after tag */
		}
	}
	i += w
	w = 0

	if rndr.ext_flags&MKDEXT_LAX_HTML_BLOCKS != 0 {
		if i < size {
			w = is_empty(data[i:])
		}
	} else {
		if i < size {
			if w = is_empty(data[i:]); w == 0 {
				return 0 /* non-blank line after tag line */
			}
		}
	}
	return i + w
}

/* parsing of inline HTML block */
func parse_htmlblock(ob *bytes.Buffer, rndr *render, data []byte, do_render bool) int {
	defer un(trace("parse_htmlblock"))
	size := len(data)
	i := 0
	j := 0

	/* identification of the opening tag */
	if size < 2 || data[0] != '<' {
		return 0
	}
	curtag := find_block_tag(data[1:])

	/* handling of special cases */
	if 0 == len(curtag) {

		/* HTML comment, laxist form */
		if size > 5 && data[1] == '!' && data[2] == '-' && data[3] == '-' {
			i = 5

			for i < size && !(data[i-2] == '-' && data[i-1] == '-' && data[i] == '>') {
				i++
			}

			i++

			if i < size {
				j = is_empty(data[i:])
			}

			if j > 0 {
				work_size := i + j
				if do_render && nil != rndr.make.blockhtml {
					rndr.make.blockhtml(ob, data[:work_size], rndr.make.opaque)
				}
				return work_size
			}
		}

		/* HR, which is the only self-closing block tag considered */
		if size > 4 && (data[1] == 'h' || data[1] == 'H') && (data[2] == 'r' || data[2] == 'R') {
			i = 3
			for i < size && data[i] != '>' {
				i += 1
			}

			if i+1 < size {
				i += 1
				j = is_empty(data[i:])
				if j > 0 {
					work_size := i + j
					if do_render && nil != rndr.make.blockhtml {
						// TODO: use i + j directly instead of work_size
						rndr.make.blockhtml(ob, data[:work_size], rndr.make.opaque)
					}
					return work_size
				}
			}
		}

		/* no special case recognised */
		return 0
	}

	/* looking for an unindented matching closing tag */
	/*	followed by a blank line */
	i = 1
	found := false

	/* if not found, trying a second pass looking for indented match */
	/* but not if tag is "ins" or "del" (following original Markdown.pl) */
	if !bytes.Equal(curtag, INS_TAG) && !bytes.Equal(curtag, DEL_TAG) {
		i = 1
		for i < size {
			i++
			for i < size && !(data[i-1] == '<' && data[i] == '/') {
				i++
			}

			if i+2+len(curtag) >= size {
				break
			}

			j = htmlblock_end(curtag, rndr, data[i-1:])

			if j > 0 {
				i += j - 1
				found = true
				break
			}
		}
	}

	if !found {
		return 0
	}

	/* the end of the block has been found */
	if do_render && nil != rndr.make.blockhtml {
		work_size := i
		rndr.make.blockhtml(ob, data[:work_size], rndr.make.opaque) // TODO: just use i directly
	}

	return i
}

func parse_table_row(ob *bytes.Buffer, rndr *render, data []byte, col_data []int) {
	defer un(trace("parse_table_row"))
	size := len(data)
	columns := len(col_data)
	i := 0

	row_work := rndr.newbuf(BUFFER_SPAN)

	if i < size && data[i] == '|' {
		i++
	}

	col := 0
	for col = 0; col < columns && i < size; col++ {
		cell_work := rndr.newbuf(BUFFER_SPAN)
		for i < size && isspace(data[i]) {
			i++
		}

		cell_start := i
		for i < size && data[i] != '|' {
			i++
		}

		cell_end := i - 1
		for cell_end > cell_start && isspace(data[cell_end]) {
			cell_end--
		}

		parse_inline(cell_work, rndr, data[cell_start:cell_end+1])
		if nil != rndr.make.table_cell {
			tmp := 0
			if len(col_data) != 0 {
				tmp = col_data[col]
			}
			rndr.make.table_cell(row_work, cell_work.Bytes(), tmp, rndr.make.opaque)
		}

		rndr.popbuf(BUFFER_SPAN)
		i++
	}

	for ; col < columns; col++ {
		var empty_cell []byte // TODO: should this be non-nil?
		if nil != rndr.make.table_cell {
			tmp := 0
			if len(col_data) != 0 {
				tmp = col_data[col]
			}
			rndr.make.table_cell(row_work, empty_cell, tmp, rndr.make.opaque)
		}
	}

	if nil != rndr.make.table_row {
		rndr.make.table_row(ob, row_work.Bytes(), rndr.make.opaque)
	}
	rndr.popbuf(BUFFER_SPAN)
}

// return column_data_out as a second return arg
func parse_table_header(ob *bytes.Buffer, rndr *render, data []byte, column_data_out *[]int) int {
	defer un(trace("parse_table_header"))
	size := len(data)
	i := 0
	pipes := 0
	for i < size && data[i] != '\n' {
		if data[i] == '|' {
			pipes++
		}
		i++
	}

	if i == size || pipes == 0 {
		return 0
	}

	header_end := i

	if data[0] == '|' {
		pipes--
	}

	if i > 2 && data[i-1] == '|' {
		pipes--
	}

	columns := pipes + 1
	column_data := make([]int, columns, columns)
	*column_data_out = column_data
	/* Parse the header underline */
	i++
	if i < size && data[i] == '|' {
		i++
	}

	under_end := i
	for under_end < size && data[under_end] != '\n' {
		under_end++
	}

	col := 0
	for col = 0; col < columns && i < under_end; col++ {
		dashes := 0

		for i < under_end && (data[i] == ' ' || data[i] == '\t') {
			i++
		}

		if data[i] == ':' {
			i++
			column_data[col] |= MKD_TABLE_ALIGN_L
			dashes++
		}

		for i < under_end && data[i] == '-' {
			i++
			dashes++
		}

		if i < under_end && data[i] == ':' {
			i++
			column_data[col] |= MKD_TABLE_ALIGN_R
			dashes++
		}

		for i < under_end && (data[i] == ' ' || data[i] == '\t') {
			i++
		}

		if i < under_end && data[i] != '|' {
			break
		}

		if dashes < 3 {
			break
		}

		i++
	}

	if col < columns {
		return 0
	}

	parse_table_row(ob, rndr, data[:header_end], column_data)
	return under_end + 1
}

func parse_table(ob *bytes.Buffer, rndr *render, data []byte) int {
	defer un(trace("parse_table"))
	size := len(data)
	header_work := rndr.newbuf(BUFFER_SPAN)
	body_work := rndr.newbuf(BUFFER_BLOCK)
	var col_data []int
	i := parse_table_header(header_work, rndr, data, &col_data)
	if i > 0 {
		for i < size {
			pipes := 0
			row_start := i
			for i < size && data[i] != '\n' {
				if data[i] == '|' {
					pipes++
				}
				i++
			}

			if pipes == 0 || i == size {
				i = row_start
				break
			}

			parse_table_row(body_work, rndr, data[row_start:], col_data)
			i++
		}

		if nil != rndr.make.table {
			rndr.make.table(ob, header_work.Bytes(), body_work.Bytes(), rndr.make.opaque)
		}
	}

	rndr.popbuf(BUFFER_SPAN)
	rndr.popbuf(BUFFER_BLOCK)
	return i
}

func parse_block(ob *bytes.Buffer, rndr *render, data []byte) {
	defer un(trace("parse_block"))
	beg := 0

	if rndr.reachedNestingLimit() {
		return
	}

	size := len(data)

	for beg < size {
		txt_data := data[beg:]
		if is_atxheader(rndr, txt_data) {
			beg += parse_atxheader(ob, rndr, txt_data)
			continue
		}
		if data[beg] == '<' && rndr.make.blockhtml != nil {
			if i := parse_htmlblock(ob, rndr, txt_data, true); i != 0 {
				beg += i
				continue
			}
		}
		if i := is_empty(txt_data); i != 0 {
			beg += i
			continue
		}
		if is_hrule(txt_data) {
			if nil != rndr.make.hrule {
				rndr.make.hrule(ob, rndr.make.opaque)
				for beg < size && data[beg] != '\n' {
					beg++
				}
				beg++
			}
			continue
		}
		if rndr.ext_flags&MKDEXT_FENCED_CODE != 0 {
			if i := parse_fencedcode(ob, rndr, txt_data); i != 0 {
				beg += i
				continue
			}
		}
		if rndr.ext_flags&MKDEXT_TABLES != 0 {
			if i := parse_table(ob, rndr, txt_data); i != 0 {
				beg += i
				continue
			}
		}
		if prefix_quote(txt_data) > 0 {
			beg += parse_blockquote(ob, rndr, txt_data)
			continue
		}
		if prefix_code(txt_data) > 0 {
			beg += parse_blockcode(ob, rndr, txt_data)
			continue
		}
		if prefix_uli(txt_data) > 0 {
			beg += parse_list(ob, rndr, txt_data, 0)
			continue
		}
		if prefix_oli(txt_data) > 0 {
			beg += parse_list(ob, rndr, txt_data, MKD_LIST_ORDERED)
			continue
		}
		beg += parse_paragraph(ob, rndr, txt_data)
	}
}

/*********************
 * REFERENCE PARSING *
 *********************/

// Returns whether a line is a reference or not
func is_ref(data []byte, beg, end int) (ref bool, last int, lr *LinkRef) {
	defer un(trace("is_ref"))
	ref = false
	last = 0 // doesn't matter unless ref is true

	i := 0
	if beg+3 >= end {
		return
	}
	if data[beg] == ' ' {
		i = 1
		if data[beg+1] == ' ' {
			i = 2
			if data[beg+2] == ' ' {
				i = 3
				if data[beg+3] == ' ' {
					return
				}
			}
		}
	}
	i += beg

	/* id part: anything but a newline between brackets */
	if data[i] != '[' {
		return
	}
	i++
	id_offset := i
	for i < end && data[i] != '\n' && data[i] != '\r' && data[i] != ']' {
		i++
	}
	if i >= end || data[i] != ']' {
		return
	}
	id_end := i

	/* spacer: colon (space | tab)* newline? (space | tab)* */
	i++
	if i >= end || data[i] != ':' {
		return
	}
	i += 1
	for i < end && (data[i] == ' ' || data[i] == '\t') {
		i += 1
	}
	if i < end && (data[i] == '\n' || data[i] == '\r') {
		i += 1
		if i < end && data[i] == '\r' && data[i-1] == '\n' {
			i += 1
		}
	}
	for i < end && (data[i] == ' ' || data[i] == '\t') {
		i += 1
	}
	if i >= end {
		return
	}

	/* link: whitespace-free sequence, optionally between angle brackets */
	if data[i] == '<' {
		i += 1
	}

	link_offset := i
	for i < end && data[i] != ' ' && data[i] != '\t' && data[i] != '\n' && data[i] != '\r' {
		i += 1
	}

	link_end := i
	if data[i-1] == '>' {
		link_end = i - 1
	}

	/* optional spacer: (space | tab)* (newline | '\'' | '"' | '(' ) */
	for i < end && (data[i] == ' ' || data[i] == '\t') {
		i += 1
	}

	if i < end && data[i] != '\n' && data[i] != '\r' && data[i] != '\'' && data[i] != '"' && data[i] != '(' {
		return
	}
	line_end := 0
	/* computing end-of-line */
	if i >= end || data[i] == '\r' || data[i] == '\n' {
		line_end = i
	}
	if i+1 < end && data[i] == '\n' && data[i+1] == '\r' {
		line_end = i + 1
	}

	/* optional (space|tab)* spacer after a newline */
	if line_end > 0 {
		i = line_end + 1
		for i < end && (data[i] == ' ' || data[i] == '\t') {
			i++
		}
	}

	/* optional title: any non-newline sequence enclosed in '"()
	alone on its line */
	title_offset := 0
	title_end := 0
	if i+1 < end && (data[i] == '\'' || data[i] == '"' || data[i] == '(') {
		i += 1
		title_offset = i
		/* looking for EOL */
		for i < end && data[i] != '\n' && data[i] != '\r' {
			i += 1
		}
		if i+1 < end && data[i] == '\n' && data[i+1] == '\r' {
			title_end = i + 1
		} else {
			title_end = i
		}
		/* stepping back */
		i -= 1
		for i > title_offset && (data[i] == ' ' || data[i] == '\t') {
			i -= 1
		}
		if i > title_offset && (data[i] == '\'' || data[i] == '"' || data[i] == ')') {
			line_end = title_end
			title_end = i
		}
	}
	if line_end == 0 {
		return /* garbage after the link */
	}

	/* a valid ref has been found, filling-in return structures */
	last = line_end
	ref = true

	lr = new(LinkRef)
	lr.id = data[id_offset:id_end]
	lr.link = data[link_offset:link_end]
	if title_end > title_offset {
		lr.title = data[title_offset:title_end]
	}
	return
}

func expand_tabs(ob *bytes.Buffer, line []byte) {
	defer un(trace("expand_tabs"))
	tab := 0
	i := 0
	size := len(line)
	for i < size {
		org := i
		for i < size && line[i] != '\t' {
			i++
			tab++
		}
		if i > org {
			ob.Write(line[org:i])
		}
		if i >= size {
			break
		}
		for {
			ob.WriteByte(' ')
			tab++
			if tab%4 == 0 {
				break
			}
		}
		i++
	}
}

func ups_markdown_init(r *render, extensions uint) {
	defer un(trace("ups_markdown_init"))
	if nil != r.make.emphasis || nil != r.make.double_emphasis || nil != r.make.triple_emphasis {
		r.active_char['*'] = MD_CHAR_EMPHASIS
		r.active_char['_'] = MD_CHAR_EMPHASIS
		if extensions&MKDEXT_STRIKETHROUGH != 0 {
			r.active_char['~'] = MD_CHAR_EMPHASIS
		}
	}

	if r.make.codespan != nil {
		r.active_char['`'] = MD_CHAR_CODESPAN
	}

	if r.make.linebreak != nil {
		r.active_char['\n'] = MD_CHAR_LINEBREAK
	}

	if nil != r.make.image || nil != r.make.link {
		r.active_char['['] = MD_CHAR_LINK
	}

	r.active_char['<'] = MD_CHAR_LANGLE
	r.active_char['\\'] = MD_CHAR_ESCAPE
	r.active_char['&'] = MD_CHAR_ENTITITY

	if extensions&MKDEXT_AUTOLINK != 0 {
		r.active_char['h'] = MD_CHAR_AUTOLINK // http, https
		r.active_char['H'] = MD_CHAR_AUTOLINK

		r.active_char['f'] = MD_CHAR_AUTOLINK // ftp
		r.active_char['F'] = MD_CHAR_AUTOLINK

		r.active_char['m'] = MD_CHAR_AUTOLINK // mailto
		r.active_char['M'] = MD_CHAR_AUTOLINK
	}
	r.refs = make([]*LinkRef, 16)

	r.ext_flags = extensions
	r.max_nesting = 16
}

// TODO: a big change would be to use slices more directly rather than pass indexes
func MarkdownToHtml(s string, options uint) string {
	defer un(trace("MarkdownToHtml"))
	init_markdown_char_ptrs()

	var rndr render
	rndr.make = upshtml_renderer(options)
	ups_markdown_init(&rndr, 0)

	ib := []byte(s)
	ob := new(bytes.Buffer)
	text := new(bytes.Buffer)

	/* first pass: looking for references, copying everything else */
	beg := 0
	for beg < len(ib) {
		if isRef, last, ref := is_ref(ib, beg, len(ib)); isRef {
			beg = last
			if nil != ref {
				rndr.refs = append(rndr.refs, ref)
			}
		} else { /* skipping to the next line */
			end := beg
			for end < len(ib) && ib[end] != '\n' && ib[end] != '\r' {
				end += 1
			}

			/* adding the line body if present */
			if end > beg {
				expand_tabs(text, ib[beg:end])
			}

			for end < len(ib) && (ib[end] == '\n' || ib[end] == '\r') {
				/* add one \n per newline */
				if ib[end] == '\n' || (end+1 < len(ib) && ib[end+1] != '\n') {
					text.WriteByte('\n')
				}
				end += 1
			}

			beg = end
		}
	}

	/* sorting the reference array */
	// TODO: sort renderer.refs

	/* second pass: actual rendering */
	if rndr.make.doc_header != nil {
		rndr.make.doc_header(ob, rndr.make.opaque)
	}

	if text.Len() > 0 {
		/* adding a final newline if not already present */
		data := text.Bytes()
		if data[len(data)-1] != '\n' && data[len(data)-1] != '\r' {
			text.WriteByte('\n')
		}

		parse_block(ob, &rndr, text.Bytes())
	}

	if rndr.make.doc_footer != nil {
		rndr.make.doc_footer(ob, rndr.make.opaque)
	}

	return string(ob.Bytes())
}

func UnitTest() {
	find_emph_char([]byte("ca"), '*')
}
