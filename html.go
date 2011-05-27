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
	for i := 0; i < size; i++ {
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
func rndr_autolink(ob *bytes.Buffer, link []byte, typ int, opaque interface{}) bool {
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
	if ob.Len() > 0 {
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

/*
 * GitHub style code block:
 *
 *		<pre lang="LANG"><code>
 *		...
 *		</pre></code>
 *
 * Unlike other parsers, we store the language identifier in the <pre>,
 * and don't let the user generate custom classes.
 *
 * The language identifier in the <pre> block gets postprocessed and all
 * the code inside gets syntax highlighted with Pygments. This is much safer
 * than letting the user specify a CSS class for highlighting.
 *
 * Note that we only generate HTML for the first specifier.
 * E.g.
 *		~~~~ {.python .numbered}	=>	<pre lang="python"><code>
 */
func rndr_blockcode_github(ob *bytes.Buffer, text []byte, lang []byte) {
	if ob.Len() > 0 {
		ob.WriteByte('\n')
	}

	if len(lang) > 0 {
		i := 0;
		ob.WriteString("<pre lang=\"")

		for i < len(lang) && !isspace(lang[i]) {
			i++
		}

		if lang[0] == '.' {
			attr_escape(ob, lang[1:i])
			// Note: hopefully correctly translated
			// attr_escape(ob, lang->data + 1, i - 1)
		} else {
			attr_escape(ob, lang[:i])
		}

		ob.WriteString("\"><code>")
	} else {
		ob.WriteString("<pre><code>")
	}
	if len(text > 0) {
		attr_escape(ob, text)
	}

	ob.WriteString("</code></pre>\n")
}

func rndr_blockquote(ob *bytes.Buffer, text []byte, opaque interface{}) {
	ob.WriteString("<blockquote>\n")
	ob.Write(text)
	ob.WriteString("</blockquote>")
}

func rndr_codespan(ob *bytes.Buffer, text []byte, opaque interface{}) bool {
	ob.WriteString("<code>")
	attr_escape(ob, text)
	ob.WriteString("</code>")
	return true
}

func rndr_strikethrough(ob *bytes.Buffer, text []byte, opaque interface{}) bool {
	if len(text) == 0 {
		return false
	}
	ob.WriteString("<del>")
	ob.Write(text)
	ob.WriteString("</del>")
	return true
}

func rndr_double_emphasis(ob *bytes.Buffer, text []byte, opaque interface{}) bool {
	if len(text) == 0 {
		return false
	}
	ob.WriteString("<strong>")
	ob.Write(text)
	ob.WriteString("</strong>")
	return true
}

func rndr_emphasis(ob *bytes.Buffer, text []byte, opaque interface{}) bool {
	if len(text) == 0 {
		return false
	}
	ob.WriteString("<em>")
	ob.Write(text)
	ob.WriteString("</em>")
	return true
}

func rndr_header(ob *bytes.Buffer, text []byte, level int, opaque interface{}) {
	//struct html_renderopt *options = opaque;
	
	if ob.Len() > 0 {
		ob.WriteByte('\n')
	}

	/*
	if (options->flags & HTML_TOC)
		bufprintf(ob, "<h%d id=\"toc_%d\">", level, options->toc_data.header_count++);
	else
		bufprintf(ob, "<h%d>", level);
	*/
	ob.Write(text)
	ob.WriteString(fmt.Sprintf("</h%d>\n", level))
}

func rndr_link(ob *bytes.Buffer, link []byte, title []byte, content []byte, opaque interface{}) bool {
	//struct html_renderopt *options = opaque;
	
	//if ((options->flags & HTML_SAFELINK) != 0 && !is_safe_link(link->data, link->size))
	//	return 0;

	ob.WriteString("<a href=\"")
	ob.Write(link)
	if len(title) > 0 {
		ob.WriteString("\" title=\"")
		attr_escape(ob, title)
	}
	ob.WriteString("\">")
	ob.Write(content)
	ob.WriteString("</a>")
	return true
}

func rndr_list(ob *bytes.Buffer, text []byte, flags int, opaque interface{}) {
	if ob.Len() > 0 {
		ob.WriteByte('\n')
	}

	if flags & MKD_LIST_ORDERED {
		ob.WriteString("<ol>\n")
	} else {
		ob.WriteString("<ul>\n")		
	}
	ob.Write(text)
	if flags & MKD_LIST_ORDERED {
		ob.WriteString("</ol>\n")
	} else {
		ob.WriteString("</ul>\n")		
	}
}

func rndr_listitem(ob *bytes.Buffer, text []byte, flags int, opaque interface{}) {
	ob.WriteString("<li>")
	i := len(text)
	for i > 0 && text[i-1] == '\n' {
		i--
	}
	ob.Write(text[:i])
	ob.WriteString("</li>\n")
}

static void
rndr_paragraph(ob *bytes.Buffer, text []byte, opaque interface{})
{
	struct html_renderopt *options = opaque;
	size_t i = 0;

	if ob.Len() > 0 {
		ob.WriteByte('\n')
	}

	if (!text || !text->size)
		return;

	while (i < text->size && isspace(text->data[i])) i++;

	if (i == text->size)
		return;

	BUFPUTSL(ob, "<p>");
	if (options->flags & HTML_HARD_WRAP) {
		size_t org;
		while (i < text->size) {
			org = i;
			while (i < text->size && text->data[i] != '\n')
				i++;

			if (i > org)
				bufput(ob, text->data + org, i - org);

			if (i >= text->size)
				break;

			BUFPUTSL(ob, "<br");
			bufputs(ob, options->close_tag);
			i++;
		}
	} else {
		bufput(ob, &text->data[i], text->size - i);
	}
	BUFPUTSL(ob, "</p>\n");
}

static void
rndr_raw_block(ob *bytes.Buffer, text []byte, opaque interface{})
{
	size_t org, sz;
	if (!text) return;
	sz = text->size;
	while (sz > 0 && text->data[sz - 1] == '\n') sz -= 1;
	org = 0;
	while (org < sz && text->data[org] == '\n') org += 1;
	if (org >= sz) return;
	if (ob->size) bufputc(ob, '\n');
	bufput(ob, text->data + org, sz - org);
	bufputc(ob, '\n');
}

static int
rndr_triple_emphasis(ob *bytes.Buffer, text []byte, opaque interface{})
{
	if (!text || !text->size) return 0;
	BUFPUTSL(ob, "<strong><em>");
	bufput(ob, text->data, text->size);
	BUFPUTSL(ob, "</em></strong>");
	return 1;
}

func rndr_hrule(ob *bytes.Buffer, opaque interface{}) {
	//struct html_renderopt *options = opaque;	

	if ob.Len() > 0 {
		ob.WriteByte('\n')
	}

	ob.WriteString("<hr")
	//bufputs(ob, options->close_tag);
}

func rndr_image(ob *bytes.Buffer, link []byte, title []byte, alt []byte, opaque interface{}) bool {
	//struct html_renderopt *options = opaque;
	if len(link) == 0 {
		return false
	}
	ob.WriteString("<img src=\"")
	attr_escape(ob, link)
	ob.WriteString("\" alt=\"")
	attr_escape(ob, alt)
	if (title && title->size) {
		BUFPUTSL(ob, "\" title=\"");
		attr_escape(ob, title->data, title->size); }

	bufputc(ob, '"');
	bufputs(ob, options->close_tag);
	return 1;
}

func rndr_linebreak(ob *bytes.Buffer, opaque interface{}) bool {
	//struct html_renderopt *options = opaque;
	ob.WriteString("<br")
	//bufputs(ob, options->close_tag);
	return true
}

static int
rndr_raw_html(ob *bytes.Buffer, text []byte, opaque interface{})
{
	struct html_renderopt *options = opaque;	

	if ((options->flags & HTML_SKIP_HTML) != 0)
		return 1;

	if ((options->flags & HTML_SKIP_STYLE) != 0 && is_html_tag(text, "style"))
		return 1;

	if ((options->flags & HTML_SKIP_LINKS) != 0 && is_html_tag(text, "a"))
		return 1;

	if ((options->flags & HTML_SKIP_IMAGES) != 0 && is_html_tag(text, "img"))
		return 1;

	bufput(ob, text->data, text->size);
	return 1;
}

static void
rndr_table(ob *bytes.Buffer, header []byte, body []byte, opaque interface{})
{
	if (ob->size) bufputc(ob, '\n');
	BUFPUTSL(ob, "<table><thead>\n");
	if (header)
		bufput(ob, header->data, header->size);
	BUFPUTSL(ob, "\n</thead><tbody>\n");
	if (body)
		bufput(ob, body->data, body->size);
	BUFPUTSL(ob, "\n</tbody></table>");
}

static void
rndr_tablerow(ob *bytes.Buffer, text []byte, opaque interface{})
{
	if (ob->size) bufputc(ob, '\n');
	BUFPUTSL(ob, "<tr>\n");
	if (text)
		bufput(ob, text->data, text->size);
	BUFPUTSL(ob, "\n</tr>");
}

static void
rndr_tablecell(ob *bytes.Buffer, text []byte, align int, opaque interface{})
{
	if (ob->size) bufputc(ob, '\n');
	switch (align) {
	case MKD_TABLE_ALIGN_L:
		BUFPUTSL(ob, "<td align=\"left\">");
		break;

	case MKD_TABLE_ALIGN_R:
		BUFPUTSL(ob, "<td align=\"right\">");
		break;

	case MKD_TABLE_ALIGN_CENTER:
		BUFPUTSL(ob, "<td align=\"center\">");
		break;

	default:
		BUFPUTSL(ob, "<td>");
		break;
	}

	if (text)
		bufput(ob, text->data, text->size);
	BUFPUTSL(ob, "</td>");
}

static void
rndr_normal_text(ob *bytes.Buffer, text []byte, opaque interface{})
{
	if (text)
		attr_escape(ob, text->data, text->size);
}

static void
toc_header(ob *bytes.Buffer, text []byte, level int, opaque interface{})
{
	struct html_renderopt *options = opaque;

	while (level > options->toc_data.current_level) {
		if (options->toc_data.current_level > 0)
			BUFPUTSL(ob, "<li>");
		BUFPUTSL(ob, "<ul>\n");
		options->toc_data.current_level++;
	}

	while (level < options->toc_data.current_level) {
		BUFPUTSL(ob, "</ul>");
		if (options->toc_data.current_level > 1)
			BUFPUTSL(ob, "</li>\n");
		options->toc_data.current_level--;
	}

	bufprintf(ob, "<li><a href=\"#toc_%d\">", options->toc_data.header_count++);
	if (text)
		bufput(ob, text->data, text->size);
	BUFPUTSL(ob, "</a></li>\n");
}

static void
toc_finalize(ob *bytes.Buffer, opaque interface{})
{
	struct html_renderopt *options = opaque;

	while (options->toc_data.current_level > 1) {
		BUFPUTSL(ob, "</ul></li>\n");
		options->toc_data.current_level--;
	}

	if (options->toc_data.current_level)
		BUFPUTSL(ob, "</ul>\n");
}

