module main

go 1.22.2

toolchain go1.23.6

replace lukasolson.net/common => ../common

require lukasolson.net/common v0.0.0-00010101000000-000000000000

require github.com/maja42/ember v1.2.0

require github.com/stretchr/testify v1.9.0 // indirect

replace github.com/maja42/ember => github.com/lukasgolson/ember v0.0.0-20240222203012-16dfde8ef5de
