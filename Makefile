all: ui/handles.go piano-assistant

ui/handles.go: ui/ui.glade
	go generate ./ui

piano-assistant:
	go build ./cmd/piano-assistant

run: piano-assistant
	./start.sh
clean:
	rm piano-assistant
dev: ui/handles.go
	GTK_DEBUG=interactive go run ./cmd/piano-assistant

deps:
	pkcon install rtmidi-devel gtk3-devel cairo-devel glib-devel gcc-c++
