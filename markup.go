package markup

import (
	"bytes"
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

type MarkdownOptions struct {
	Xhtml           bool
	SkipImages      bool
	SkipLinks       bool
	SkipHtml        bool
	GitHubBlockCode bool

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
	out        bytes.Buffer
	options    *MarkdownOptions
	closeTag   string
	refs       []*LinkRef
	activeChar [256]byte
	maxNesting int
}

func newHtmlRenderer(options *MarkdownOptions) *HtmlRenderer {
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

func (d *HtmlRenderer) renderBlockCode(text, lang string) {

}

func (d *HtmlRenderer) renderDocHeader() {
	// do nothing
}

// Returns whether a line is a reference or not
func isRef(data []byte, beg, end int) (ref bool, last int, lr *LinkRef) {
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
	if data[0] != '#' {
		return false
	}

	if (rndr.options.ExtSpaceHeaders) {
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

func parse_atxheader(rndr *HtmlRenderer, data []byte) int {
	// TODO: implement me
	return 0
}

func is_empty(data []byte) int {
	var i int
	size := len(data)
	for i = 0; i < size && data[i] != '\n'; i++ {
		if data[i] != ' ' && data[i] != '\t' {
			return 0
		}
	}
	return i+1
}

func parse_block(rndr *HtmlRenderer, data []byte) {
	beg := 0

	/*
	if (rndr->work_bufs[BUFFER_SPAN].size +
		rndr->work_bufs[BUFFER_BLOCK].size > (int)rndr->max_nesting)
		return;
	*/
	size := len(data)

	for beg < size {
		txt_data := data[beg:]
		//end := len(txt_data)
		if is_atxheader(rndr, txt_data) {
			beg += parse_atxheader(rndr, txt_data)
		}
		/* 		else if (data[beg] == '<' && rndr->make.blockhtml &&
				(i = parse_htmlblock(ob, rndr, txt_data, end, 1)) != 0)
			beg += i; */
		if i := is_empty(txt_data); i != 0 {
			beg += i
		}

		beg++
	}
}

// TODO: a big change would be to use slices more directly rather than pass indexes
func MarkdownToHtml(s string, options *MarkdownOptions) string {
	//var i int
	renderer := newHtmlRenderer(options)
	ib := []byte(s)
	text := new(bytes.Buffer)

	/* first pass: looking for references, copying everything else */
	beg := 0
	for beg < len(ib) {
		if isRef, last, ref := isRef(ib, beg, len(ib)); isRef {
			beg = last
			if nil != ref {
				renderer.refs = append(renderer.refs, ref)
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
	renderer.renderDocHeader()

	if (text.Len() > 0) {
		/* adding a final newline if not already present */
		data := text.Bytes()
		if (data[len(data) - 1] != '\n' &&  data[len(data) - 1] != '\r') {
			text.WriteByte('\n')
		}

		parse_block(renderer, text.Bytes());
	}

	return s
}
