all: ui/handles.go piano-assistant

ui/handles.go: ui/ui.glade
	go generate ./ui

piano-assistant:
	go build ./cmd/piano-assistant

run: piano-assistant
	./start.sh
