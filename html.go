package markup

import (
	"bytes"
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

const (
	xhtmlClose = "/>\n"
	htmlClose  = ">\n"
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

type HtmlRenderer struct {
	options    *MarkdownOptions
	closeTag   string
	refs       []*LinkRef
	activeChar [256]byte
	blockBufs  []*bytes.Buffer
	spanBufs   []*bytes.Buffer
	maxNesting int
}

// TODO: what other chars are space?
func isspace(c byte) bool {
	return c == ' '
}

// TODO: what other chars are punctutation
func ispunct(c byte) bool {
	if c == '.' {
		return true
	}
	return false
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

func is_html_tag(tag []byte, tagname []byte) bool {
	i := 0
	size := len(tag)
	if i < size && tag[0] != '<' {
		return false
	}

	i++

	for i < size && isspace(tag[i]) {
		i++
	}

	if i < size && tag[i] == '/' {
		i++
	}

	for i < size && isspace(tag[i]) {
		i++
	}
	j := 0
	for i < size && j < len(tagname) {
		if tag[i] != tagname[j] {
			return false
		}
		i++
		j++
	}

	if i == size {
		return false
	}

	return isspace(tag[i]) || tag[i] == '>'
}

/********************
 * GENERIC RENDERER *
 ********************/
func rndr_autolink(ob *bytes.Buffer, link []byte, typ int) bool {
	//struct html_renderopt *options = opaque;

	size := len(link)
	if 0 == size {
		return false
	}

	/*
	if ((options->flags & HTML_SAFELINK) != 0 &&
		!is_safe_link(link->data, link->size) &&
		type != MKDA_EMAIL)
		return 0;
	*/

	ob.WriteString("<a href=\"")
	if typ == MKDA_EMAIL {
		ob.WriteString("mailto:")
	}

	ob.Write(link)
	ob.WriteString("\">")

	/*
	 * Pretty printing: if we get an email address as
	 * an actual URI, e.g. `mailto:foo@bar.com`, we don't
	 * want to print the `mailto:` prefix
	 */
	if bytes.HasPrefix(link, []byte("mailto:")) {
		attr_escape(ob, link[7:])
	} else {
		attr_escape(ob, link)
	}

	ob.WriteString("</a>")
	return true
}

func rndr_blockcode(ob *bytes.Buffer, text []byte, lang []byte) {
	if ob.Len() == 0 {
		ob.WriteByte('\n')
	}

	if len(lang) > 0 {
		ob.WriteString("<pre><code class=\"")

		i := 0
		cls := 0
		for i < len(lang) {
			i++
			cls++
			for i < len(lang) && isspace(lang[i]) {
				i++
			}

			if i < len(lang) {
				org := i
				for i < len(lang) && !isspace(lang[i]) {
					i++
				}

				if lang[org] == '.' {
					org++
				}

				if cls > 0 {
					ob.WriteByte(' ')
				}
				attr_escape(ob, lang[org:])
			}
		}
		ob.WriteString("\">")
	} else {
		ob.WriteString("<pre><code>")
	}

	if len(text) > 0 {
		attr_escape(ob, text)
	}
	ob.WriteString("</code></pre>\n")
}
