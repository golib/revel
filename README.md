# Revel

> This is 3d-party repo forked from [Revel](https://github.com/revel/revel). The goal of this is to easy revel usage. We'll try our best to submit the patch to official repo.

A high productivity, full-stack web framework for the [Go language](http://www.golang.org).

Current Version: 0.9.1 (Mar 1, 2014)

[![Build Status](https://travis-ci.org/golib/revel.svg?branch=master)](https://travis-ci.org/golib/revel)

## New Features
- Support app spec config file following run mode, such as *app.dev.conf*
- New template enginer interface, which makes custom template parser esaier
```go
type TemplateEnginer interface {
  Parse(s string) (*template.Template, error)
  SetOptions(options *config.Config)
  SetHelpers(helpers template.FuncMap)
  WatchDir(dir os.FileInfo) bool
  WatchFile(file string) bool
}
```
- Support template layout

## Learn More

[Manual, Samples, Godocs, etc](http://revel.github.io)

## Join The Community

* [Google Groups](https://groups.google.com/forum/#!forum/revel-framework) via [revel-framework@googlegroups.com](mailto:revel-framework@googlegroups.com)
* [GitHub Issues](https://github.com/golib/revel/issues)
* [IRC](http://webchat.freenode.net/?channels=%23revel&uio=d4) via #revel on Freenode

## Announcements

### New GitHub Repo

We have moved to the @revel organization. See the [v0.9.0 release notes](https://github.com/golib/revel/releases/tag/v0.9.0)
for how to update your app.

### v1.0 Goal

You'll notice that our next planned milestone release is v0.10 and you may wonder if Revel is
production-ready. The short answer is yes. However, the team feels that there are still some
features needed before we can make a whole-hearted endorsement of Revel as a true "batteries-included" web framework.

For those of you new to Go or Revel, we'd like to invite you to join us on our march to v1.0
and help make Revel a stable, secure and enjoyable web framework.

We'd like to take this opportunity to thank the entire community for their feedback and
regular contributions. Your support has been very encouraging and highly appreciated.
