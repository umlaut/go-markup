package markup

import (
	"bytes"
	"fmt"
)

const (
	MKDEXT_NO_INTRA_EMPHASIS = 1 << 0
	MKDEXT_TABLES            = 1 << 1
	MKDEXT_FENCED_CODE       = 1 << 2
	MKDEXT_AUTOLINK          = 1 << 3
	MKDEXT_STRIKETHROUGH     = 1 << 4
	MKDEXT_LAX_HTML_BLOCKS   = 1 << 5
	MKDEXT_SPACE_HEADERS     = 1 << 6
)

const (
	HTML_SKIP_HTML        = 1 << 0
	HTML_SKIP_STYLE       = 1 << 1
	HTML_SKIP_IMAGES      = 1 << 2
	HTML_SKIP_LINKS       = 1 << 3
	HTML_EXPAND_TABS      = 1 << 5
	HTML_SAFELINK         = 1 << 7
	HTML_TOC              = 1 << 8
	HTML_HARD_WRAP        = 1 << 9
	HTML_GITHUB_BLOCKCODE = 1 << 10
	HTML_USE_XHTML        = 1 << 11
)

/* list/listitem flags */
const (
	MKD_LIST_ORDERED = 1
	MKD_LI_BLOCK     = 2 /* <li> containing block data */
)

const (
	MKD_TABLE_ALIGN_L      = 1 << 0
	MKD_TABLE_ALIGN_R      = 1 << 1
	MKD_TABLE_ALIGN_CENTER = MKD_TABLE_ALIGN_L | MKD_TABLE_ALIGN_R
)

const (
	xhtml_close = "/>\n"
	html_close  = ">\n"
)

type html_renderopt struct {
	toc_data struct {
		header_count  int
		current_level int
	}

	flags     uint
	close_tag string
}

type rndrFunc func(*bytes.Buffer, interface{})
type rndrBufFunc func(*bytes.Buffer, []byte, interface{})
type rndrBufBufFunc func(*bytes.Buffer, []byte, []byte, interface{})
type rndBufIntFunc func(*bytes.Buffer, []byte, int, interface{})

type rndrFunc_b func(*bytes.Buffer, interface{}) bool
type rndrBufFunc_b func(*bytes.Buffer, []byte, interface{}) bool
type rndBufIntFunc_b func(*bytes.Buffer, []byte, int, interface{}) bool
type rndrBufBufFunc_b func(*bytes.Buffer, []byte, []byte, interface{}) bool
type rndrBufBufBufFunc_b func(*bytes.Buffer, []byte, []byte, []byte, interface{}) bool

/* functions for rendering parsed data */
type mkd_renderer struct {
	/* block level callbacks - NULL skips the block */
	blockcode  rndrBufBufFunc
	blockquote rndrBufFunc
	blockhtml  rndrBufFunc
	header     rndBufIntFunc
	hrule      rndrFunc
	list       rndBufIntFunc
	listitem   rndBufIntFunc
	paragraph  rndrBufFunc
	table      rndrBufBufFunc
	table_row  rndrBufFunc
	table_cell rndBufIntFunc

	/* span level callbacks - NULL or return 0 prints the span verbatim */
	autolink        rndBufIntFunc_b
	codespan        rndrBufFunc_b
	double_emphasis rndrBufFunc_b
	emphasis        rndrBufFunc_b
	image           rndrBufBufBufFunc_b
	linebreak       rndrFunc_b
	link            rndrBufBufBufFunc_b
	raw_html_tag    rndrBufFunc_b
	triple_emphasis rndrBufFunc_b
	strikethrough   rndrBufFunc_b

	/* low level callbacks - NULL copies input directly into the output */
	entity      rndrBufFunc
	normal_text rndrBufFunc

	/* header and footer */
	doc_header rndrFunc
	doc_footer rndrFunc

	/* user data */
	opaque interface{}
}

/*
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
}*/


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

func (rndr *render) reachedNestingLimit() bool {
	//defer un(trace("reachedNestingLimit"))
	return len(rndr.work_bufs[0])+len(rndr.work_bufs[1]) > rndr.max_nesting
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

func is_html_tag(tag []byte, tagname string) bool {
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
	tagnameb := []byte(tagname) // TODO: use bytes.HasPrefix()?
	for i < size && j < len(tagnameb) {
		if tag[i] != tagnameb[j] {
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
	options, _ := opaque.(*html_renderopt)

	size := len(link)
	if 0 == size {
		return false
	}

	if (options.flags&HTML_SAFELINK != 0) && !is_safe_link(link) && typ != MKDA_EMAIL {
		return false
	}

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

func rndr_blockcode(ob *bytes.Buffer, text []byte, lang []byte, opaque interface{}) {
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
func rndr_blockcode_github(ob *bytes.Buffer, text []byte, lang []byte, opaque interface{}) {
	if ob.Len() > 0 {
		ob.WriteByte('\n')
	}

	if len(lang) > 0 {
		i := 0
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
	if len(text) > 0 {
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
	options, _ := opaque.(*html_renderopt)

	if ob.Len() > 0 {
		ob.WriteByte('\n')
	}

	if options.flags&HTML_TOC != 0 {
		ob.WriteString(fmt.Sprintf("<h%d id=\"toc_%d\">", level, options.toc_data.header_count))
		options.toc_data.header_count++
	} else {
		ob.WriteString(fmt.Sprintf("<h%d>", level))
	}
	ob.Write(text)
	ob.WriteString(fmt.Sprintf("</h%d>\n", level))
}

func rndr_link(ob *bytes.Buffer, link []byte, title []byte, content []byte, opaque interface{}) bool {
	options, _ := opaque.(*html_renderopt)

	if (options.flags&HTML_SAFELINK != 0) && !is_safe_link(link) {
		return false
	}

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

	if flags&MKD_LIST_ORDERED != 0 {
		ob.WriteString("<ol>\n")
	} else {
		ob.WriteString("<ul>\n")
	}
	ob.Write(text)
	if flags&MKD_LIST_ORDERED != 0 {
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

func rndr_paragraph(ob *bytes.Buffer, text []byte, opaque interface{}) {
	options, _ := opaque.(*html_renderopt)

	i := 0

	if ob.Len() > 0 {
		ob.WriteByte('\n')
	}

	if len(text) == 0 {
		return
	}

	size := len(text)
	for i < size && isspace(text[i]) {
		i++
	}

	if i == size {
		return
	}

	ob.WriteString("<p>")
	if options.flags&HTML_HARD_WRAP != 0 {
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
			ob.WriteString(options.close_tag)
			i++
		}
	} else {
		ob.Write(text[i:])
	}
	ob.WriteString("</p>\n")
}

func rndr_raw_block(ob *bytes.Buffer, text []byte, opaque interface{}) {
	if len(text) == 0 {
		return
	}
	sz := len(text)
	for sz > 0 && text[sz-1] == '\n' {
		sz -= 1
	}
	org := 0
	for org < sz && text[org] == '\n' {
		org += 1
	}
	if org >= sz {
		return
	}
	if ob.Len() > 0 {
		ob.WriteByte('\n')
	}
	ob.Write(text[org:sz])
	ob.WriteByte('\n')
}

func rndr_triple_emphasis(ob *bytes.Buffer, text []byte, opaque interface{}) bool {
	if len(text) == 0 {
		return false
	}
	ob.WriteString("<strong><em>")
	ob.Write(text)
	ob.WriteString("</em></strong>")
	return true
}

func rndr_hrule(ob *bytes.Buffer, opaque interface{}) {
	options, _ := opaque.(*html_renderopt)

	if ob.Len() > 0 {
		ob.WriteByte('\n')
	}

	ob.WriteString("<hr")
	ob.WriteString(options.close_tag)
}

func rndr_image(ob *bytes.Buffer, link []byte, title []byte, alt []byte, opaque interface{}) bool {
	options, _ := opaque.(*html_renderopt)
	if len(link) == 0 {
		return false
	}
	ob.WriteString("<img src=\"")
	attr_escape(ob, link)
	ob.WriteString("\" alt=\"")
	attr_escape(ob, alt)
	if len(title) > 0 {
		ob.WriteString("\" title=\"")
		attr_escape(ob, title)
	}

	ob.WriteByte('"')
	ob.WriteString(options.close_tag)
	return true
}

func rndr_linebreak(ob *bytes.Buffer, opaque interface{}) bool {
	options, _ := opaque.(*html_renderopt)
	ob.WriteString("<br")
	ob.WriteString(options.close_tag)
	return true
}

func rndr_raw_html(ob *bytes.Buffer, text []byte, opaque interface{}) bool {
	options, _ := opaque.(*html_renderopt)

	if options.flags&HTML_SKIP_HTML != 0 {
		return true
	}

	if (options.flags&HTML_SKIP_STYLE != 0) && is_html_tag(text, "style") {
		return true
	}

	if (options.flags&HTML_SKIP_LINKS != 0) && is_html_tag(text, "a") {
		return true
	}

	if (options.flags&HTML_SKIP_IMAGES != 0) && is_html_tag(text, "img") {
		return true
	}

	ob.Write(text)
	return true
}

func rndr_table(ob *bytes.Buffer, header []byte, body []byte, opaque interface{}) {
	if ob.Len() > 0 {
		ob.WriteByte('\n')
	}
	ob.WriteString("<table><thead>\n")
	ob.Write(header)
	ob.WriteString("\n</thead><tbody>\n")
	ob.Write(body)
	ob.WriteString("\n</tbody></table>")
}

func rndr_tablerow(ob *bytes.Buffer, text []byte, opaque interface{}) {
	if ob.Len() > 0 {
		ob.WriteByte('\n')
	}
	ob.WriteString("<tr>\n")
	ob.Write(text)
	ob.WriteString("\n</tr>")
}

func rndr_tablecell(ob *bytes.Buffer, text []byte, align int, opaque interface{}) {
	if ob.Len() > 0 {
		ob.WriteByte('\n')
	}
	switch {
	case align == MKD_TABLE_ALIGN_L:
		ob.WriteString("<td align=\"left\">")

	case align == MKD_TABLE_ALIGN_R:
		ob.WriteString("<td align=\"right\">")

	case align == MKD_TABLE_ALIGN_CENTER:
		ob.WriteString("<td align=\"center\">")

	case true:
		ob.WriteString("<td>")
	}

	ob.Write(text)
	ob.WriteString("</td>")
}

func rndr_normal_text(ob *bytes.Buffer, text []byte, opaque interface{}) {
	attr_escape(ob, text)
}

func toc_header(ob *bytes.Buffer, text []byte, level int, opaque interface{}) {
	options, _ := opaque.(*html_renderopt)

	for level > options.toc_data.current_level {
		if options.toc_data.current_level > 0 {
			ob.WriteString("<li>")
		}
		ob.WriteString("<ul>\n")
		options.toc_data.current_level++
	}

	for level < options.toc_data.current_level {
		ob.WriteString("</ul>")
		if options.toc_data.current_level > 1 {
			ob.WriteString("</li>\n")
		}
		options.toc_data.current_level--
	}

	ob.WriteString(fmt.Sprintf("<li><a href=\"#toc_%d\">", options.toc_data.header_count))
	options.toc_data.header_count++
	ob.Write(text)
	ob.WriteString("</a></li>\n")
}

func toc_finalize(ob *bytes.Buffer, opaque interface{}) {
	options, _ := opaque.(*html_renderopt)

	for options.toc_data.current_level > 1 {
		ob.WriteString("</ul></li>\n")
		options.toc_data.current_level--
	}

	if options.toc_data.current_level > 0 {
		ob.WriteString("</ul>\n")
	}
}

func upshtml_renderer(render_flags uint) *mkd_renderer {

	renderer := &mkd_renderer{
		rndr_blockcode,
		rndr_blockquote,
		rndr_raw_block,
		rndr_header,
		rndr_hrule,
		rndr_list,
		rndr_listitem,
		rndr_paragraph,
		rndr_table,
		rndr_tablerow,
		rndr_tablecell,

		rndr_autolink,
		rndr_codespan,
		rndr_double_emphasis,
		rndr_emphasis,
		rndr_image,
		rndr_linebreak,
		rndr_link,
		rndr_raw_html,
		rndr_triple_emphasis,
		rndr_strikethrough,

		nil,
		rndr_normal_text,

		nil,
		nil,
		nil}

	var opts html_renderopt
	opts.flags = render_flags
	opts.close_tag = html_close
	if render_flags&HTML_USE_XHTML != 0 {
		opts.close_tag = xhtml_close
	}
	renderer.opaque = &opts

	if render_flags&HTML_SKIP_IMAGES != 0 {
		renderer.image = nil
	}

	if render_flags&HTML_SKIP_LINKS != 0 {
		renderer.link = nil
		renderer.autolink = nil
	}

	if render_flags&HTML_SKIP_HTML != 0 {
		renderer.blockhtml = nil
	}

	if render_flags&HTML_GITHUB_BLOCKCODE != 0 {
		renderer.blockcode = rndr_blockcode_github
	}

	return renderer
}
