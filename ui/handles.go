package ui

import "github.com/gotk3/gotk3/gtk"

var eraseSessionImg *gtk.Image
var exportImg *gtk.Image
var openImg *gtk.Image
var pauseImg *gtk.Image
var playImg *gtk.Image
var previousImg *gtk.Image
var quantizeImg *gtk.Image
var reconnectImg *gtk.Image
var refreshImg *gtk.Image
var retourImt *gtk.Image
var saveImg *gtk.Image
var startrecordImg *gtk.Image
var stepsImg *gtk.Image
var stoprecordImg *gtk.Image
var undoImg *gtk.Image
var mainWin *gtk.Window
var mainHBox *gtk.Box
var reconnectMidi *gtk.Button
var reloadBtn *gtk.Button
var comboInPorts *gtk.ComboBox
var comboOutPorts *gtk.ComboBox
var bpmEntry *gtk.Entry
var ticksEntry *gtk.Entry
var quantizeBtn *gtk.Button
var fileButtons *gtk.Box
var saveStateBtn *gtk.Button
var loadStateBtn *gtk.Button
var eraseSessionBtn *gtk.Button
var playBtn *gtk.Button
var stepsChb *gtk.CheckButton
var stepReset *gtk.Button
var stepPrev *gtk.Button
var importBankBtn *gtk.FileChooserButton
var recordBtn *gtk.Button
var undoNote *gtk.Button
var exportMultiTrackBtn *gtk.Button
var dragNdropZones *gtk.Box
var importZone *gtk.EventBox
var exportZone *gtk.EventBox
var deleteZone *gtk.EventBox
var cutZone *gtk.EventBox
var trackTreeView *gtk.TreeView
var tracksTab *gtk.Label
var sessionsTreeView *gtk.TreeView
var sessionsTab *gtk.Label
var banksBox *gtk.Box

func loadUI(builder *gtk.Builder) {
	_eraseSessionImg, _ := builder.GetObject("eraseSessionImg")
	eraseSessionImg = _eraseSessionImg.(*gtk.Image)
	_exportImg, _ := builder.GetObject("exportImg")
	exportImg = _exportImg.(*gtk.Image)
	_openImg, _ := builder.GetObject("openImg")
	openImg = _openImg.(*gtk.Image)
	_pauseImg, _ := builder.GetObject("pauseImg")
	pauseImg = _pauseImg.(*gtk.Image)
	_playImg, _ := builder.GetObject("playImg")
	playImg = _playImg.(*gtk.Image)
	_previousImg, _ := builder.GetObject("previousImg")
	previousImg = _previousImg.(*gtk.Image)
	_quantizeImg, _ := builder.GetObject("quantizeImg")
	quantizeImg = _quantizeImg.(*gtk.Image)
	_reconnectImg, _ := builder.GetObject("reconnectImg")
	reconnectImg = _reconnectImg.(*gtk.Image)
	_refreshImg, _ := builder.GetObject("refreshImg")
	refreshImg = _refreshImg.(*gtk.Image)
	_retourImt, _ := builder.GetObject("retourImt")
	retourImt = _retourImt.(*gtk.Image)
	_saveImg, _ := builder.GetObject("saveImg")
	saveImg = _saveImg.(*gtk.Image)
	_startrecordImg, _ := builder.GetObject("startrecordImg")
	startrecordImg = _startrecordImg.(*gtk.Image)
	_stepsImg, _ := builder.GetObject("stepsImg")
	stepsImg = _stepsImg.(*gtk.Image)
	_stoprecordImg, _ := builder.GetObject("stoprecordImg")
	stoprecordImg = _stoprecordImg.(*gtk.Image)
	_undoImg, _ := builder.GetObject("undoImg")
	undoImg = _undoImg.(*gtk.Image)
	_mainWin, _ := builder.GetObject("mainWin")
	mainWin = _mainWin.(*gtk.Window)
	_mainHBox, _ := builder.GetObject("mainHBox")
	mainHBox = _mainHBox.(*gtk.Box)
	_reconnectMidi, _ := builder.GetObject("reconnectMidi")
	reconnectMidi = _reconnectMidi.(*gtk.Button)
	_reloadBtn, _ := builder.GetObject("reloadBtn")
	reloadBtn = _reloadBtn.(*gtk.Button)
	_comboInPorts, _ := builder.GetObject("comboInPorts")
	comboInPorts = _comboInPorts.(*gtk.ComboBox)
	_comboOutPorts, _ := builder.GetObject("comboOutPorts")
	comboOutPorts = _comboOutPorts.(*gtk.ComboBox)
	_bpmEntry, _ := builder.GetObject("bpmEntry")
	bpmEntry = _bpmEntry.(*gtk.Entry)
	_ticksEntry, _ := builder.GetObject("ticksEntry")
	ticksEntry = _ticksEntry.(*gtk.Entry)
	_quantizeBtn, _ := builder.GetObject("quantizeBtn")
	quantizeBtn = _quantizeBtn.(*gtk.Button)
	_fileButtons, _ := builder.GetObject("fileButtons")
	fileButtons = _fileButtons.(*gtk.Box)
	_saveStateBtn, _ := builder.GetObject("saveStateBtn")
	saveStateBtn = _saveStateBtn.(*gtk.Button)
	_loadStateBtn, _ := builder.GetObject("loadStateBtn")
	loadStateBtn = _loadStateBtn.(*gtk.Button)
	_eraseSessionBtn, _ := builder.GetObject("eraseSessionBtn")
	eraseSessionBtn = _eraseSessionBtn.(*gtk.Button)
	_playBtn, _ := builder.GetObject("playBtn")
	playBtn = _playBtn.(*gtk.Button)
	_stepsChb, _ := builder.GetObject("stepsChb")
	stepsChb = _stepsChb.(*gtk.CheckButton)
	_stepReset, _ := builder.GetObject("stepReset")
	stepReset = _stepReset.(*gtk.Button)
	_stepPrev, _ := builder.GetObject("stepPrev")
	stepPrev = _stepPrev.(*gtk.Button)
	_importBankBtn, _ := builder.GetObject("importBankBtn")
	importBankBtn = _importBankBtn.(*gtk.FileChooserButton)
	_recordBtn, _ := builder.GetObject("recordBtn")
	recordBtn = _recordBtn.(*gtk.Button)
	_undoNote, _ := builder.GetObject("undoNote")
	undoNote = _undoNote.(*gtk.Button)
	_exportMultiTrackBtn, _ := builder.GetObject("exportMultiTrackBtn")
	exportMultiTrackBtn = _exportMultiTrackBtn.(*gtk.Button)
	_dragNdropZones, _ := builder.GetObject("dragNdropZones")
	dragNdropZones = _dragNdropZones.(*gtk.Box)
	_importZone, _ := builder.GetObject("importZone")
	importZone = _importZone.(*gtk.EventBox)
	_exportZone, _ := builder.GetObject("exportZone")
	exportZone = _exportZone.(*gtk.EventBox)
	_deleteZone, _ := builder.GetObject("deleteZone")
	deleteZone = _deleteZone.(*gtk.EventBox)
	_cutZone, _ := builder.GetObject("cutZone")
	cutZone = _cutZone.(*gtk.EventBox)
	_trackTreeView, _ := builder.GetObject("trackTreeView")
	trackTreeView = _trackTreeView.(*gtk.TreeView)
	_tracksTab, _ := builder.GetObject("tracksTab")
	tracksTab = _tracksTab.(*gtk.Label)
	_sessionsTreeView, _ := builder.GetObject("sessionsTreeView")
	sessionsTreeView = _sessionsTreeView.(*gtk.TreeView)
	_sessionsTab, _ := builder.GetObject("sessionsTab")
	sessionsTab = _sessionsTab.(*gtk.Label)
	_banksBox, _ := builder.GetObject("banksBox")
	banksBox = _banksBox.(*gtk.Box)
}
