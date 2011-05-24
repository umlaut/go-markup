package markup

import (
	"bytes"
)

const (
	xhtmlClose = "/>\n"
	htmlClose = ">\n"
)

const (
	MKDA_NOT_AUTOLINK = iota	/* used internally when it is not an autolink*/
	MKDA_NORMAL					/* normal http/http/ftp/mailto/etc link */
	MKDA_EMAIL					/* e-mail link without explit mailto: */
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

type MarkdownOptions struct  {
	Xhtml bool
	SkipImages bool
	SkipLinks bool
	SkipHtml bool
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
	ExtTables bool
	ExtFencedCode bool
	ExtAutoLink bool
	ExtStrikeThrough bool
	ExtLaxHtmlBlocks bool
	ExtSpaceHeaders bool
}

type LinkRef struct {
	// make them all []byte ?
	Id string
	Link string
	Title string
}

type HtmlRenderer struct {
	out bytes.Buffer 
	options *MarkdownOptions
	closeTag string
	refs []LinkRef
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

	r.activeChar['<'] = MD_CHAR_LANGLE;
	r.activeChar['\\'] = MD_CHAR_ESCAPE;
	r.activeChar['&'] = MD_CHAR_ENTITITY;

	if options.ExtAutoLink {
		r.activeChar['h'] = MD_CHAR_AUTOLINK; // http, https
		r.activeChar['H'] = MD_CHAR_AUTOLINK;

		r.activeChar['f'] = MD_CHAR_AUTOLINK; // ftp
		r.activeChar['F'] = MD_CHAR_AUTOLINK;

		r.activeChar['m'] = MD_CHAR_AUTOLINK; // mailto
		r.activeChar['M'] = MD_CHAR_AUTOLINK;		
	}
	r.maxNesting = 16
	return r
}

func (d *HtmlRenderer) renderBlockCode(text, lang string) {
	
}

// Returns whether a line is a reference or not
// TODO: return slice of LinkRef
func isRef(data []byte, beg, end int) (ref bool, last int) {
	ref = false
	last = 0 // doesn't matter unless ref is true
	i := 0
	if beg + 3 >= end {
		return
	}
	if data[beg] == ' ' {
		i = 1; if data[beg+1] == ' ' {
			i = 2; if data[beg+2] == ' ' {
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
	i += 1;
	for i < end && (data[i] == ' ' || data[i] == '\t') {
		i += 1;
	}
	if i < end && (data[i] == '\n' || data[i] == '\r') {
		i += 1;
		if i < end && data[i] == '\r' && data[i - 1] == '\n' {
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
	if data[i - 1] == '>' {
		link_end = i - 1	
	}

	/* optional spacer: (space | tab)* (newline | '\'' | '"' | '(' ) */
	// TODO: and write the rest
	return
}

// TODO: a big change would be to use slices rather than pass indexes
func MarkdownToHtml(s string, options *MarkdownOptions) string {
	var i, end int
	renderer := newHtmlRenderer(options)
	ib := []byte(s)
	/* first pass: looking for references, copying everything else */
	beg := 0
	for beg < len(ib) {
		// TODO: also gather slice of LinkRef
		if ref, last := isRef(ib, beg, len(ib)); ref {
			
		}
	}
	return s
}
