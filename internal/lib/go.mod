module lib

go 1.20

require (
	github.com/rivo/tview v0.0.0-20231206124440-5f078138442e
	golang.org/x/exp v0.0.0-20231226003508-02704c960a9b
)

require (
	github.com/gdamore/tcell/v2 v2.6.1-0.20231203215052-2917c3801e73
	github.com/mum4k/termdash v0.18.0
	pkg/storage v0.0.0
)

require (
	github.com/gdamore/encoding v1.0.0 // indirect
	github.com/lucasb-eyer/go-colorful v1.2.0 // indirect
	github.com/mattn/go-runewidth v0.0.14 // indirect
	github.com/rivo/uniseg v0.4.3 // indirect
	golang.org/x/sys v0.11.0 // indirect
	golang.org/x/term v0.9.0 // indirect
	golang.org/x/text v0.12.0 // indirect
)

replace pkg/storage => ../../pkg/storage
