我是光年实验室高级招聘经理。
我在github上访问了你的开源项目，你的代码超赞。你最近有没有在看工作机会，我们在招软件开发工程师，拉钩和BOSS等招聘网站也发布了相关岗位，有公司和职位的详细信息。
我们公司在杭州，业务主要做流量增长，是很多大型互联网公司的流量顾问。公司弹性工作制，福利齐全，发展潜力大，良好的办公环境和学习氛围。
公司官网是http://www.gnlab.com,公司地址是杭州市西湖区古墩路紫金广场B座，若你感兴趣，欢迎与我联系，
电话是0571-88839161，手机号：18668131388，微信号：echo 'bGhsaGxoMTEyNAo='|base64 -D ,静待佳音。如有打扰，还请见谅，祝生活愉快工作顺利。

# mdserve

*mdserve* is a Markdown webserver. It's pointed to a directory; all
assets in that directory become available through HTTP, Markdown files
are converted to HTML on the fly. Nothing is stored on disk, so the
directory isn't cluttered with state files and other crap. Files are
updated as soon as they're changed on disk, it's enough to hit the
browsers refresh button to get the latest version.


## Usage

*mdserve* is as easy as it gets: If `mdserve` is called it's started
with the current working directory as web root directory and an URL
(which can be openend on the browser of choice) is printed. The URL
either points to the root directory (the server will return 403, the
user must enter the path to a Markdown file by hand) or to an index.md
file, if available.

**Command line options**:

* **-a**: Listen address. Must be given with port, e.g. `10.0.0.1:8080`.
  Defaults to `localhost:8080`.
* **-d**: Web root directory, defaults to `.`.
* **-l**: Language for typography and hyphenation, defaults to `de`.
  Currently only `de` and `en` are supported.


## Markdown dialect

*mdserve* is build around the Goldmark CommonMark parser. It implements
Github Flavoured Markdown: https://github.github.com/gfm/

Some other extensions are available:

* Definition lists: https://michelf.ca/projects/php-markdown/extra/#def-list
* Footnotes: https://michelf.ca/projects/php-markdown/extra/#footnotes
* Highlighting of fenced code blocks.


## Installation

You'll need the *go* tools in version 1.13 or higher. Additionally
*packr2* is required to crunch the static assets into the binary, it
can be installed with:

  `% go get -u github.com/gobuffalo/packr/v2/packr`

1. Clone the Github repo into a local directory and change into it.
2. Create the packr files: `% cd cmd/mdserve; packr2; cd -`
3. Compile the executable: `go build github.com/yamagi/mdserve/cmd/mdserve`
