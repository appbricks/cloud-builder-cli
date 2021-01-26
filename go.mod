module github.com/appbricks/cloud-builder-cli

go 1.13

replace github.com/appbricks/cloud-builder-cli => ./

replace github.com/appbricks/cloud-builder => ../cloud-builder

replace github.com/mevansam/gocloud => ../../mevansam/gocloud

replace github.com/mevansam/goforms => ../../mevansam/goforms

replace github.com/mevansam/goutils => ../../mevansam/goutils

replace github.com/mevansam/termtables => ../../mevansam/termtables

require (
	github.com/appbricks/cloud-builder v0.0.0-00010101000000-000000000000
	github.com/briandowns/spinner v1.12.0
	github.com/gobuffalo/packr/v2 v2.7.1
	github.com/gookit/color v1.2.1
	github.com/mevansam/gocloud v0.0.0-00010101000000-000000000000
	github.com/mevansam/goforms v0.0.0-00010101000000-000000000000
	github.com/mevansam/goutils v0.0.0-00010101000000-000000000000
	github.com/mevansam/termtables v0.0.0-00010101000000-000000000000
	github.com/mitchellh/go-homedir v1.1.0
	github.com/peterh/liner v1.1.0
	github.com/spf13/cobra v0.0.5
	golang.org/x/crypto v0.0.0-20200622213623-75b288015ac9
	golang.org/x/oauth2 v0.0.0-20210113205817-d3ed898aa8a3
)
