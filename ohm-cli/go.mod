module github.com/ohmjs/ohm-go/ohm-cli

go 1.26.4

tool (
	github.com/adl-lang/goadlc/cmd/goadlc
	github.com/jpillora/md-tmpl
)

require (
	github.com/adl-lang/goadl_rt/v3 v3.0.0-alpha.10
	github.com/google/go-cmp v0.6.0
	github.com/jpillora/opts v1.2.3
	github.com/millergarym/gotmpl v1.2.0
	github.com/ohmjs/ohm-go/ohm v0.0.1
	github.com/samber/lo v1.53.0
	golang.org/x/tools v0.49.0
)

// replace github.com/millergarym/gotmpl => ../../../golang/gotmpl

require (
	github.com/adl-lang/goadlc v1.0.0-alpha.13 // indirect
	github.com/hashicorp/errwrap v1.0.0 // indirect
	github.com/hashicorp/go-multierror v1.0.0 // indirect
	github.com/jpillora/md-tmpl v1.3.0 // indirect
	github.com/mattn/go-zglob v0.0.4 // indirect
	github.com/posener/complete v1.2.2-0.20190308074557-af07aa5181b3 // indirect
	github.com/tetratelabs/wazero v1.11.0 // indirect
	golang.org/x/mod v0.39.0 // indirect
	golang.org/x/sync v0.22.0 // indirect
	golang.org/x/sys v0.47.0 // indirect
	golang.org/x/text v0.22.0 // indirect
)
