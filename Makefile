all: ui/handles.go piano-assistant

ui/handles.go: ui/ui.glade
	go generate ./ui

piano-assistant:
	go build -buildmode pie -compiler gc '-tags=rpm_crashtraceback ' -a -v -x -ldflags ' -X github.com/JeanRibes/midi-recorder/version=0 -X github.com/JeanRibes/midi-recorder/version.commit=`git rev-parse HEAD` -compressdwarf=false' -o piano-assistant ./cmd/piano-assistant

run: piano-assistant
	./start.sh
clean:
	rm piano-assistant
dev: ui/handles.go
	GTK_DEBUG=interactive go run ./cmd/piano-assistant

deps:
	pkcon install rtmidi-devel gtk3-devel cairo-devel glib-devel gcc-c++ go2rpm

rpm:
	#go2rpm --no-dynamic-buildrequires github.com/JeanRibes/midi-recorder
	rpmdev-setuptree
	git archive --format=tar.gz --prefix=midi-recorder-`git rev-parse HEAD`/ -o midi-recorder-`git rev-parse HEAD`.tar.gz HEAD
	mv midi-recorder-`git rev-parse HEAD`.tar.gz ~/rpmbuild/SOURCES
	rpmbuild -bb golang-github-jeanribes-midi-recorder.spec
