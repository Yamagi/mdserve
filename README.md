# mdserve

*mdserve* is a Markdown webserver. It's pointed to a directory; all
assets in that directory become available through HTTP, Markdown files
are converted to HTML on the fly. Nothing is stored on disk, the
directory isn't cluttered with state files. Files are updated as soon as
they're changed on disk, it's enough to refresh the page to get the
latest version.


## Usage

*mdserve* is as easy as it gets: If `mdserve` is called without any
arguments, it's started with the current working directory as web root
directory and an URL (which can be openend on the browser of choice) is
printed. The URL either points to the root directory (the server will
return 403, the user must enter the path to a Markdown file by hand) or
to an index.md file, if available.

**Command line options**:

* **-a**: Listen address. Must be given with port, e.g. `10.0.0.1:8080`.
  Defaults to `localhost:8080`.
* **-d**: Web root directory, defaults to `.`.
* **-j**: Use an alternative CSS with centered text and auto
  hyphenation.
* **-l**: Language for typography and hyphenation, defaults to `de`.
  Currently `de` and `en` are supported.
* **-q**: Be quiet, suppress console output.


## Markdown dialect

*mdserve* is build around the Goldmark CommonMark parser. It implements
Github Flavoured Markdown: https://github.github.com/gfm/

Some other extensions are available:

* Definition lists: https://michelf.ca/projects/php-markdown/extra/#def-list
* Footnotes: https://michelf.ca/projects/php-markdown/extra/#footnotes
* Highlighting of fenced code blocks.
* KaTeX math: https://katex.org/
* WikiLinks: https://en.wikipedia.org/wiki/Hyperlink#Wikis


## Installation

You'll need the *go* tools in version 1.18 or higher.

1. Clone the Github repo into a local directory and change into it.
2. Compile the executable: `go build ./cmd/mdserve`
