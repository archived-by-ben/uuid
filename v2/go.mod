module github.com/Defacto2/uuid/v2

go 1.13

require (
	github.com/dustin/go-humanize v1.0.0
	github.com/go-sql-driver/mysql v1.4.1
	github.com/mitchellh/go-homedir v1.1.0
	github.com/spf13/cobra v0.0.5
	github.com/spf13/viper v1.6.1
//	github.com/Defacto2/uuid/v2/lib/data v0.0.0
)

// https://stackoverflow.com/questions/52026284/accessing-local-packages-within-a-go-module-go-1-11
//replace github.com/Defacto2/uuid/v2/lib/data v0.0.0 => ./lib/data
