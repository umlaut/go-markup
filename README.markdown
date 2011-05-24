go-markup
=========

go-markup is an implementation of markdown to html converter
in Go language.

It's a port of C library upskirt (https://github.com/tanoku/upskirt).

Testing
-------

*.text files in testfiles directory are from MarkdownTest 1.0.3
(https://github.com/jgm/peg-markdown/tree/master/MarkdownTest_1.0.3/Tests)

I've used upskirt to generate corresponding _upskirt_ref.html files
(without using any extensions).

I test go-markup by converting .text files to html and comparing them
with _upskirt_ref.html files.

Todo
----

First, have to finish the port.

Second, would be nice to add textile support.

Credits
-------

* Krzysztof Kowalczyk - port of upskirt to go
* Natacha Porté - original author of upskirt
* Vicent Marti - upskirt fork on which go-markdown is based
* other people who contributed to upskirt

License
-------

Permission to use, copy, modify, and distribute this software for any
purpose with or without fee is hereby granted, provided that the above
copyright notice and this permission notice appear in all copies.

THE SOFTWARE IS PROVIDED "AS IS" AND THE AUTHOR DISCLAIMS ALL WARRANTIES
WITH REGARD TO THIS SOFTWARE INCLUDING ALL IMPLIED WARRANTIES OF
MERCHANTABILITY AND FITNESS. IN NO EVENT SHALL THE AUTHOR BE LIABLE FOR
ANY SPECIAL, DIRECT, INDIRECT, OR CONSEQUENTIAL DAMAGES OR ANY DAMAGES
WHATSOEVER RESULTING FROM LOSS OF USE, DATA OR PROFITS, WHETHER IN AN
ACTION OF CONTRACT, NEGLIGENCE OR OTHER TORTIOUS ACTION, ARISING OUT OF
OR IN CONNECTION WITH THE USE OR PERFORMANCE OF THIS SOFTWARE.
