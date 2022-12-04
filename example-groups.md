Here are a list of all code blocks in the example group:

<!--eyecue-codemap-group:UuLzD7n96cD:{{ range . }}- {{ .MarkdownRangeLink }}{{ "\n" }}{{ end }}-->
- [example-groups.js:1](example-groups.js#L2-L4)
- [example-groups.js:7](example-groups.js#L8-L10)
<!--end-eyecue-codemap-group-->

And as a table:

<table>
<tr><th>Link</th></tr>
<!--eyecue-codemap-group:UuLzD7n96cD:{{ range . }}<tr><td><a href="{{ .RangeHref }}">{{ .FileLine }}</a></td></tr>{{ "\n" }}{{ end }}-->
<tr><td><a href="example-groups.js#L2-L4">example-groups.js:1</a></td></tr>
<tr><td><a href="example-groups.js#L8-L10">example-groups.js:7</a></td></tr>
<!--end-eyecue-codemap-group-->
</table>
