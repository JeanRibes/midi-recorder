import re

regex = r"class=([A-z]+).*id=([A-z]+)"

with open('ui.glade') as f:
    for line in f.readlines():
        if 'class' in line and 'id' in line:
            line = line.replace('<object', '').replace('>','').replace('"','')
            matches = re.finditer(regex, line)
            for match in matches:
                classe = match.group(1)
                nom = match.group(2)
                classe = classe.replace('Gtk','gtk.')
                print(f"_{nom}, _ := builder.GetObject(\"{nom}\")")
                print(f"{nom} := _{nom}.(*{classe})")

 
#	_playBtn, _ := builder.GetObject("playBtn")
#	playBtn := _playBtn.(*gtk.Button)