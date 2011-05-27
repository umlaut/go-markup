package markup

import (
	"bytes"
	"fmt"
	"strings"
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
	MKD_LI_END = 8	/* internal list flag */
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

var markdown_char_ptrs []TriggerFunc = []TriggerFunc{nil, char_emphasis, char_codespan, char_linebreak, char_link, char_langle_tag, char_escape, char_entity, char_autolink}

type render struct {
	make 		mkd_renderer
	refs       	[]*LinkRef
	active_char [256]byte
	work_bufs 	[2][]*bytes.Buffer // indexed by BUFFER_BLOCK or BUFFER_SPAN
	ext_flags   uint
	max_nesting int
}

var funcNestLevel int = 0

func spaces(n int) string {
	r := []byte("                                                                   ")[:n]
	return string(r)
}

func trace(s string, args ...string) string {
	sp := spaces(funcNestLevel * 2)
	funcNestLevel++
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
	defer un(trace("newbuf"))

	buf = new(bytes.Buffer)
	rndr.work_bufs[bufType] = append(rndr.work_bufs[bufType], buf)
	return buf
}

func (rndr *render) popbuf(bufType int) {
	defer un(trace("popbuf"))
	rndr.work_bufs[bufType] = rndr.work_bufs[bufType][0 : len(rndr.work_bufs[bufType])-1]
}

var block_tags []string = []string{"p", "dl", "h1", "h2", "h3", "h4", "h5", "h6", "ol", "ul", "del", "div", "ins", "pre", "form", "math", "table", "iframe", "script", "fieldset", "noscript", "blockquote"}

const (
	INS_TAG = "ins"
	DEL_TAG = "del"
)

/***************************
 * HELPER FUNCTIONS *
 ***************************/
func is_safe_link(link []byte) bool {
	valid_uris := [4]string{"http://", "https://", "ftp://", "mailto://" }

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

		if i + 1 >= size {
			break
		}
		ob.WriteByte(src[i+1])
		i += 2
	}
}

/* returns the current block tag */
/* TODO: speed it up by auto-generated optimized 
   comparison function that is a chain of ifs */
func find_block_tag(data []byte) string {
	defer un(trace("find_block_tag"))
	i := 0
	size := len(data)

	/* looking for the word end */
	for i < size && isalnum(data[i]) {
		i++
	}
	if i == 0 || i >= size {
		return ""
	}
	s := strings.ToLower(string(data[:i]))
	for _, tag := range block_tags {
		if s == tag {
			return s
		}
	}
	return ""
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
		end = actfunc(ob, rndr, data[i:], i)
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
	size := len(data)
	i := 1

	for i < size {
		for i < size && data[i] != c && data[i] != '`' && data[i] != '[' {
			i += 1
		}

		if data[i] == c {
			return i
		}

		/* not counting escaped chars */
		if i > 0 && data[i - 1] == '\\' { // i > 0 probably not necessary
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
		} else if (data[i] == '[') {
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

		if i + 1 < size && data[i + 1] == c {
			i += 1
			continue
		}

		if data[i] == c && !isspace(data[i - 1]) {
			if rndr.ext_flags & MKDEXT_NO_INTRA_EMPHASIS != 0 {
				if !(i + 1 == size || isspace(data[i + 1]) || ispunct(data[i + 1])) {
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
	var render_method rndrBufFunc
	i := 0
	size := len(data)
	if c == '~' {
		render_method = rndr.make.strikethrough
	} else {
		render_method = rndr.make.double_emphasis
	}

	if nil == render_method {
		return 0
	}
	
	for i < size {
		len := find_emph_char(data[i:], c)
		if 0 == len {
			return 0
		}
		i += len

		if i + 1 < size && data[i] == c && data[i + 1] == c && i > 0 && !isspace(data[i - 1]) {
			work := rndr.newBuf(BUFFER_SPAN)
			parse_inline(work, rndr, data[:i])
			r := render_method(ob, work, rndr.make.opaque)
			rndr.popBuf(BUFFER_SPAN)
			if r > 0 {
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
func parse_emph3(ob *bytes.Buffer, rndr *render, data []byte, c byte) int {
	size := len(data)
	i := 0

	for i < size {
		len := find_emph_char(data[i:], c)
		if 0 == len {
			return 0
		}
		i += len

		/* skip whitespace preceded symbols */
		if data[i] != c || isspace(data[i - 1]) {
			continue
		}

		if i + 2 < size && data[i + 1] == c && data[i + 2] == c && nil != rndr.make.triple_emphasis{
			/* triple symbol found */
			work := rndr.newBuf(BUFFER_SPAN)

			parse_inline(work, rndr, data[:i])
			r := triple_emphasis(ob, work)
			rndr.popBuf(BUFFER_SPAN)
			if 0 == r {
				return i + 3
			} else {
				return 0
			}
		} else if i + 1 < size && data[i + 1] == c {
			/* double symbol found, handing over to emph1 */
			//TODO: len = parse_emph1(ob, rndr, data - 2, size + 2, c);
			if 0 == len {
				return 0
			} else {
				return len - 2
			}
		} else {
			/* single symbol found, handing over to emph2 */
			//TODO: len = parse_emph2(ob, rndr, data - 1, size + 1, c);
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

		return ret + 2;
	}

	if size > 4 && data[1] == c && data[2] == c && data[3] != c {
		if c == '~' || isspace(data[3]) {
			return 0
		}
		if ret = parse_emph3(ob, rndr, data[3:], c); ret == 0 {
			return 0
		}

		return ret + 3;
	}
	return 0
}

func char_linebreak(ob *bytes.Buffer, rndr *render, data []byte, offset int) int {
	defer un(trace("char_linebreak"))
	// TODO: write me
	return 0
}

func char_codespan(ob *bytes.Buffer, rndr *render, data []byte, offset int) int {
	defer un(trace("char_codespan"))
	// TODO: write me
	return 0
}

func char_escape(ob *bytes.Buffer, rndr *render, data []byte, offset int) int {
	defer un(trace("char_escape"))
	// TODO: write me
	return 0
}

func char_entity(ob *bytes.Buffer, rndr *render, data []byte, offset int) int {
	defer un(trace("char_entity"))
	// TODO: write me
	return 0
}

func char_langle_tag(ob *bytes.Buffer, rndr *render, data []byte, offset int) int {
	defer un(trace("char_langle_tag"))
	// TODO: write me
	return 0
}

func char_autolink(ob *bytes.Buffer, rndr *render, data []byte, offset int) int {
	defer un(trace("char_autolink"))
	// TODO: write me
	return 0
}

func char_link(ob *bytes.Buffer, rndr *render, data []byte, offset int) int {
	defer un(trace("char_link"))
	// TODO: write me
	return 0
}

// Returns whether a line is a reference or not
func isRef(data []byte, beg, end int) (ref bool, last int, lr *LinkRef) {
	defer un(trace("isRef"))
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
			ob.WriteByte('c')
			tab++
			if tab%4 == 0 {
				break
			}
		}
		i++
	}
}

func is_atxheader(rndr *render, data []byte) bool {
	defer un(trace("is_atxheader"))
	if data[0] != '#' {
		return false
	}

	if rndr.options.ExtSpaceHeaders {
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
	defer un(trace("is_headerline"))
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
func parse_atxheader(ob *bytes.Buffer, rndr *render, data []byte) int {
	defer un(trace("parse_atxheader"))
	// TODO: implement me
	return 0
}

func is_empty(data []byte) int {
	defer un(trace("is_empty"))
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
	defer un(trace("is_hrule"))
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

func parse_fencedcode(ob *bytes.Buffer, rndr *render, data []byte) int {
	defer un(trace("parse_fencedcode"))
	// TODO: write me
	return 0
}

func parse_table(ob *bytes.Buffer, rndr *render, data []byte) int {
	defer un(trace("parse_table"))
	// TODO: write me
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
	defer un(trace("prefix_quote"))
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

// checking end of HTML block : </tag>[ \t]*\n[ \t*]\n
//	returns the length on match, 0 otherwise
func htmlblock_end(tag string, rndr *render, data []byte) int {
	defer un(trace("htmlblock_end"))
	// TODO: write me
	return 0
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
		if beg < end { // copy into the in-place working buffer
			work_data = append(work_data, data[beg:end]...)
		}
		beg = end
	}

	out := rndr.newBuf(BUFFER_BLOCK)
	parse_block(out, rndr, work_data)
	rndr.blockquote(ob, out)
	rndr.popBuf(BUFFER_BLOCK)
	return end
}

/* parsing of inline HTML block */
func parse_htmlblock(ob *bytes.Buffer, rndr *render, data []byte, do_render bool) int {
	defer un(trace("parse_htmlblock"))
	i := 0
	j := 0
	curtag := ""
	size := len(data)

	/* identification of the opening tag */
	if size < 2 || data[0] != '<' {
		return 0
	}
	curtag = find_block_tag(data[1:])

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
				if do_render {
					work := bytes.NewBuffer(data[:work_size])
					rndr.blockhtml(ob, work)
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
					if do_render {
						work := bytes.NewBuffer(data[:work_size])
						rndr.blockhtml(ob, work)
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
	if curtag != INS_TAG && curtag != DEL_TAG {
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
	work_size := i
	if do_render {
		work := bytes.NewBuffer(data[:work_size])
		rndr.blockhtml(ob, work)
	}

	return i
}

/* handles parsing of a regular paragraph */
func parse_paragraph(ob *bytes.Buffer, rndr *render, data []byte) int {
	defer un(trace("parse_paragraph"))

	//struct buf work = { data, 0, 0, 0, 0 }; /* volatile working buffer */
	i := 0
	end := 0
	size := len(data)

	level := 0
	for i < size {
		for end = i + 1; end < size && data[end-1] != '\n'; end++ {
			/* empty */
		}

		if is_empty(data[i:]) > 0 {
			break
		}
		level = is_headerline(data[i:])
		if level != 0 {
			break
		}

		if rndr.options.ExtLaxHtmlBlocks {
			if data[i] == '<' && parse_htmlblock(ob, rndr, data[i:], false) > 0 {
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

	if 0 == level {
		tmp := rndr.newBuf(BUFFER_BLOCK)
		parse_inline(tmp, rndr, data[:work_size])
		rndr.paragraph(ob, tmp.Bytes())
		rndr.popBuf(BUFFER_BLOCK)
	} else {
		// TODO: write me
	}
	return end
}

func parse_block(ob *bytes.Buffer, rndr *render, data []byte) {
	defer un(trace("parse_block"))
	//fmt.Printf("parse_block:\n%s\n", string(data))
	beg := 0

	if rndr.reachedNestingLimit() {
		return
	}

	size := len(data)

	for beg < size {
		txt_data := data[beg:]
		//end := len(txt_data)
		if is_atxheader(rndr, txt_data) {
			beg += parse_atxheader(ob, rndr, txt_data)
			continue
		}
		/* if (data[beg] == '<' && rndr->make.blockhtml &&
			(i = parse_htmlblock(ob, rndr, txt_data, end, 1)) != 0)
		beg += i;
		continue
		*/
		if i := is_empty(txt_data); i != 0 {
			beg += i
			continue
		}
		if is_hrule(txt_data) {
			/*
							if (rndr->make.hrule)
					rndr->make.hrule(ob, rndr->make.opaque);

				while (beg < size && data[beg] != '\n')
					beg++;

				beg++
			*/
			continue
		}
		if rndr.options.ExtFencedCode {
			i := parse_fencedcode(ob, rndr, txt_data)
			if i > 0 {
				beg += i
				continue
			}
		}
		if rndr.options.ExtTables {
			i := parse_table(ob, rndr, txt_data)
			if i > 0 {
				beg += 1
				continue
			}
		}
		if prefix_quote(txt_data) > 0 {
			beg += parse_blockquote(ob, rndr, txt_data)
		}
		beg += parse_paragraph(ob, rndr, txt_data)
	}
}

func ups_markdown_init(r *render, extensions uint) {
	defer un(trace("ups_markdown_init"))

	if nil != r.make.emphasis || nil != rndrr.make.double_emphasis || nil != r.make.triple_emphasis {
		r.active_char['*'] = MD_CHAR_EMPHASIS
		r.active_char['_'] = MD_CHAR_EMPHASIS
		if extensions & MKDEXT_STRIKETHROUGH != 0 {
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

	if extensions & MKDEXT_AUTOLINK != 0 {
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
	return r
}

// TODO: a big change would be to use slices more directly rather than pass indexes
func MarkdownToHtml(s string, options uint) string {
	defer un(trace("MarkdownToHtml"))
	rndr := upshtml_renderer(options)
	ups_markdown_init(rndr, 0)

	ib := []byte(s)
	ob := new(bytes.Buffer)
	text := new(bytes.Buffer)

	/* first pass: looking for references, copying everything else */
	beg := 0
	for beg < len(ib) {
		if isRef, last, ref := isRef(ib, beg, len(ib)); isRef {
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
	rndr.docheader()

	if text.Len() > 0 {
		/* adding a final newline if not already present */
		data := text.Bytes()
		if data[len(data)-1] != '\n' && data[len(data)-1] != '\r' {
			text.WriteByte('\n')
		}

		parse_block(ob, rndr, text.Bytes())
	}

	return string(ob.Bytes())
}
