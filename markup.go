package markup

import (
	"bytes"
	"fmt"
	"strings"
)

const (
	xhtmlClose = "/>\n"
	htmlClose  = ">\n"
)

const (
	MKDA_NOT_AUTOLINK = iota /* used internally when it is not an autolink*/
	MKDA_NORMAL              /* normal http/http/ftp/mailto/etc link */
	MKDA_EMAIL               /* e-mail link without explit mailto: */
)

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

// bufType in newBuf() and popBuf()
const (
	BUFFER_BLOCK = iota
	BUFFER_SPAN
)

var block_tags []string = []string{"p", "dl", "h1", "h2", "h3", "h4", "h5", "h6", "ol", "ul", "del", "div", "ins", "pre", "form", "math", "table", "iframe", "script", "fieldset", "noscript", "blockquote"}

const (
	INS_TAG = "ins"
	DEL_TAG = "del"
)

type MarkdownOptions struct {
	/*	HTML_SKIP_HTML = (1 << 0),
		HTML_SKIP_STYLE = (1 << 1),
		HTML_SKIP_IMAGES = (1 << 2),
		HTML_SKIP_LINKS = (1 << 3),
		HTML_EXPAND_TABS = (1 << 5),
		HTML_SAFELINK = (1 << 7),
		HTML_TOC = (1 << 8),
		HTML_HARD_WRAP = (1 << 9),
		HTML_GITHUB_BLOCKCODE = (1 << 10),
		HTML_USE_XHTML = (1 << 11),
	*/
	SkipHtml        bool
	SkipStyle       bool
	SkipImages      bool
	SkipLinks       bool
	ExpandTabs      bool
	SafeLink        bool
	HardWrap        bool
	GitHubBlockCode bool
	Xhtml           bool

	/* bools below map:
	enum mkd_extensions {
		MKDEXT_NO_INTRA_EMPHASIS = (1 << 0),
		MKDEXT_TABLES = (1 << 1),
		MKDEXT_FENCED_CODE = (1 << 2),
		MKDEXT_AUTOLINK = (1 << 3),
		MKDEXT_STRIKETHROUGH = (1 << 4),
		MKDEXT_LAX_HTML_BLOCKS = (1 << 5),
		MKDEXT_SPACE_HEADERS = (1 << 6),
	};*/
	ExtNoIntraEmphasis bool
	ExtTables          bool
	ExtFencedCode      bool
	ExtAutoLink        bool
	ExtStrikeThrough   bool
	ExtLaxHtmlBlocks   bool
	ExtSpaceHeaders    bool
}

type LinkRef struct {
	id    []byte
	link  []byte
	title []byte
}

type HtmlRenderer struct {
	options    *MarkdownOptions
	closeTag   string
	refs       []*LinkRef
	activeChar [256]byte
	blockBufs  []*bytes.Buffer
	spanBufs   []*bytes.Buffer
	maxNesting int
}

var funcNestLevel int = 0
var spacesBytes []byte = []byte("                                                                   ")

func spaces(n int) string {
	r := spacesBytes[:n]
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

func newHtmlRenderer(options *MarkdownOptions) *HtmlRenderer {
	defer un(trace("newHtmlRenderer"))

	if options == nil {
		options = new(MarkdownOptions)
	}
	r := &HtmlRenderer{options: options}
	r.closeTag = htmlClose
	if options.Xhtml {
		r.closeTag = xhtmlClose
	}
	r.activeChar['*'] = MD_CHAR_EMPHASIS
	r.activeChar['_'] = MD_CHAR_EMPHASIS
	if options.ExtStrikeThrough {
		r.activeChar['~'] = MD_CHAR_EMPHASIS
	}
	r.activeChar['`'] = MD_CHAR_CODESPAN
	r.activeChar['\n'] = MD_CHAR_LINEBREAK
	r.activeChar['['] = MD_CHAR_LINK

	r.activeChar['<'] = MD_CHAR_LANGLE
	r.activeChar['\\'] = MD_CHAR_ESCAPE
	r.activeChar['&'] = MD_CHAR_ENTITITY

	if options.ExtAutoLink {
		r.activeChar['h'] = MD_CHAR_AUTOLINK // http, https
		r.activeChar['H'] = MD_CHAR_AUTOLINK

		r.activeChar['f'] = MD_CHAR_AUTOLINK // ftp
		r.activeChar['F'] = MD_CHAR_AUTOLINK

		r.activeChar['m'] = MD_CHAR_AUTOLINK // mailto
		r.activeChar['M'] = MD_CHAR_AUTOLINK
	}
	r.refs = make([]*LinkRef, 16)
	r.maxNesting = 16
	return r
}

func (rndr *HtmlRenderer) newBuf(bufType int) (buf *bytes.Buffer) {
	defer un(trace("newBuf"))

	buf = new(bytes.Buffer)
	if BUFFER_BLOCK == bufType {
		rndr.blockBufs = append(rndr.blockBufs, buf)
	} else {
		rndr.spanBufs = append(rndr.spanBufs, buf)
	}
	return
}

func char_emphasis(ob *bytes.Buffer, rndr *HtmlRenderer, data []byte) int {
	defer un(trace("char_emphasis"))
	// TODO: write me
	return 0
}

func char_linebreak(ob *bytes.Buffer, rndr *HtmlRenderer, data []byte) int {
	defer un(trace("char_linebreak"))
	// TODO: write me
	return 0
}

func char_codespan(ob *bytes.Buffer, rndr *HtmlRenderer, data []byte) int {
	defer un(trace("char_codespan"))
	// TODO: write me
	return 0
}

func char_escape(ob *bytes.Buffer, rndr *HtmlRenderer, data []byte) int {
	defer un(trace("char_escape"))
	// TODO: write me
	return 0
}

func char_entity(ob *bytes.Buffer, rndr *HtmlRenderer, data []byte) int {
	defer un(trace("char_entity"))
	// TODO: write me
	return 0
}

func char_langle_tag(ob *bytes.Buffer, rndr *HtmlRenderer, data []byte) int {
	defer un(trace("char_langle_tag"))
	// TODO: write me
	return 0
}

func char_autolink(ob *bytes.Buffer, rndr *HtmlRenderer, data []byte) int {
	defer un(trace("char_autolink"))
	// TODO: write me
	return 0
}

func char_link(ob *bytes.Buffer, rndr *HtmlRenderer, data []byte) int {
	defer un(trace("char_link"))
	// TODO: write me
	return 0
}

type TriggerFunc func(ob *bytes.Buffer, rndr *HtmlRenderer, data []byte) int

var markdown_char_ptrs []TriggerFunc = []TriggerFunc{nil, char_emphasis, char_codespan, char_linebreak, char_link, char_langle_tag, char_escape, char_entity, char_autolink}

func (rndr *HtmlRenderer) popBuf(bufType int) {
	defer un(trace("popBuf"))
	if BUFFER_BLOCK == bufType {
		rndr.blockBufs = rndr.blockBufs[0 : len(rndr.blockBufs)-1]
	} else {
		rndr.spanBufs = rndr.spanBufs[0 : len(rndr.spanBufs)-1]
	}
}

func (rndr *HtmlRenderer) reachedNestingLimit() bool {
	defer un(trace("reachedNestingLimit"))
	return len(rndr.blockBufs)+len(rndr.spanBufs) > rndr.maxNesting
}

// writes '<${tag}>\n${text}</${tag}\n' ot "ob"
func writeInTag(ob *bytes.Buffer, text *bytes.Buffer, tag string) {
	//defer un(trace("writeInTag"))
	ob.WriteString("<")
	ob.WriteString(tag)
	ob.WriteString(">\n")
	if text != nil {
		ob.Write(text.Bytes())
	}
	ob.WriteString("</")
	ob.WriteString(tag)
	ob.WriteString(">\n")
}

func (rndr *HtmlRenderer) blockquote(ob *bytes.Buffer, text *bytes.Buffer) {
	defer un(trace("blockquote"))
	writeInTag(ob, text, "blockquote")
	/*	ob.WriteString("<blockquote>\n")
		if text ! nil {
			ob.Write(text.Bytes())
		}
		ob.WriteString("</blockquote>\n")*/
}

// this is rndr_raw_block
func (rndr *HtmlRenderer) blockhtml(ob *bytes.Buffer, text *bytes.Buffer) {
	defer un(trace("blockhtml"))
	if nil == text {
		return
	}
	data := text.Bytes()
	sz := len(data)
	for sz > 0 && data[sz-1] == '\n' {
		sz -= 1
	}
	org := 0
	for org < sz && data[org] == '\n' {
		org += 1
	}
	if org >= sz {
		return
	}
	if ob.Len() > 0 {
		ob.WriteByte('\n')
	}
	ob.Write(data[org:])
	ob.WriteByte('\n')
}

func put_scaped_char(ob *bytes.Buffer, c byte) {
	switch {
	case c == '<':
		ob.WriteString("&lt;")
	case c == '>':
		ob.WriteString("&gt;")
	case c == '&':
		ob.WriteString("&amp;")
	case c == '"':
		ob.WriteString("&quot;")
	default:
		ob.WriteByte(c)
	}
}

/* copy the buffer entity-escaping '<', '>', '&' and '"' */
func attr_escape(ob *bytes.Buffer, src []byte) {
	defer un(trace("attr_escape"))
	size := len(src)
	i := 0
	for i < size {
		/* copying directly unescaped characters */
		org := i
		for i < size && src[i] != '<' && src[i] != '>' && src[i] != '&' && src[i] != '"' {
			i += 1
		}
		if i > org {
			ob.Write(src[org:])
		}

		/* escaping */
		if i >= size {
			break
		}

		put_scaped_char(ob, src[i])
		i++
	}
}

func (rndr *HtmlRenderer) normal_text(ob *bytes.Buffer, text *bytes.Buffer) {
	defer un(trace("normal_text"))
	if text != nil {
		attr_escape(ob, text.Bytes())
	}
}

func (rndr *HtmlRenderer) blockcode(text, lang string) {
	defer un(trace("blockcode"))

}

func (rndr *HtmlRenderer) docheader() {
	defer un(trace("docheader"))
	// do nothing
}

// TODO: what other chars are space?

func isspace(c byte) bool {
	return c == ' '
}

// rndr_paragraph
func (rndr *HtmlRenderer) paragraph(ob *bytes.Buffer, text []byte) {
	defer un(trace("paragraph", string(text)))
	//struct html_renderopt *options = opaque;
	i := 0
	size := len(text)

	if ob.Len() > 0 {
		ob.WriteByte('\n')
	}

	for i < size && isspace(text[i]) {
		i++
	}

	if i == size {
		return
	}

	ob.WriteString("<p>")

	if rndr.options.HardWrap {
		for i < size {
			org := i
			for i < size && text[i] != '\n' {
				i++
			}

			if i > org {
				ob.Write(text[org:i])
			}

			if i >= size {
				break
			}

			ob.WriteString("<br")
			ob.WriteString(rndr.closeTag)
			i++
		}
	} else {
		ob.Write(text[i:])
	}
	ob.WriteString("</p>\n")
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

func is_atxheader(rndr *HtmlRenderer, data []byte) bool {
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
func parse_atxheader(ob *bytes.Buffer, rndr *HtmlRenderer, data []byte) int {
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

func parse_fencedcode(ob *bytes.Buffer, rndr *HtmlRenderer, data []byte) int {
	defer un(trace("parse_fencedcode"))
	// TODO: write me
	return 0
}

func parse_table(ob *bytes.Buffer, rndr *HtmlRenderer, data []byte) int {
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
func htmlblock_end(tag string, rndr *HtmlRenderer, data []byte) int {
	defer un(trace("htmlblock_end"))
	// TODO: write me
	return 0
}

/* handles parsing of a blockquote fragment */
func parse_blockquote(ob *bytes.Buffer, rndr *HtmlRenderer, data []byte) int {
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

func isalnum(c byte) bool {
	if c >= '0' && c <= '9' {
		return true
	}
	if c >= 'A' && c <= 'Z' {
		return true
	}

	return c >= 'a' && c <= 'z'
}

/* returns the current block tag */
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

/* parses inline markdown elements */
func parse_inline(ob *bytes.Buffer, rndr *HtmlRenderer, data []byte) {
	defer un(trace("parse_inline"))
	i := 0
	end := 0
	size := len(data)

	if rndr.reachedNestingLimit() {
		return
	}

	var action byte = 0
	for i < size {
		/* copying inactive chars into the output */
		for end < size {
			action = rndr.activeChar[data[end]]
			if action != 0 {
				break
			}
			end++
		}

		work_data := data[i:]
		rndr.normal_text(ob, bytes.NewBuffer(work_data))

		if end >= size {
			break
		}
		i = end

		/* calling the trigger */
		f := markdown_char_ptrs[action]
		//end = f(ob, rndr, data + i, i, size - i)
		end = f(ob, rndr, data[i:])
		if 0 == end {
			/* no action from the callback */
			end = i + 1
		} else {
			i += end
			end = i
		}
	}
}
/* parsing of inline HTML block */
func parse_htmlblock(ob *bytes.Buffer, rndr *HtmlRenderer, data []byte, do_render bool) int {
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
func parse_paragraph(ob *bytes.Buffer, rndr *HtmlRenderer, data []byte) int {
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

func parse_block(ob *bytes.Buffer, rndr *HtmlRenderer, data []byte) {
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

// TODO: a big change would be to use slices more directly rather than pass indexes
func MarkdownToHtml(s string, options *MarkdownOptions) string {
	defer un(trace("MarkdownToHtml"))
	rndr := newHtmlRenderer(options)
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
