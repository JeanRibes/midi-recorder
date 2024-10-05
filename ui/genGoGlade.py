import re

regex = r"class=([A-z]+).*id=([A-z]+)"

handles = []

with open('ui.glade') as f:
    for line in f.readlines():
        if 'class' in line and 'id' in line:
            line = line.replace('<object', '').replace('>','').replace('"','')
            matches = re.finditer(regex, line)
            for match in matches:
                classe = match.group(1)
                nom = match.group(2)
                classe = classe.replace('Gtk','gtk.')
                #print(f"_{nom}, _ := builder.GetObject(\"{nom}\")")
                #print(f"{nom} := _{nom}.(*{classe})")
                handles.append((nom,classe))

with open('handles.go', 'w+') as f:
    f.write("package ui\n\n")
    f.write('import "github.com/gotk3/gotk3/gtk"\n\n')
    for nom,classe in handles:
        f.write(f"var {nom} *{classe}\n")
    f.write("\nfunc loadUI(builder *gtk.Builder) {\n")
    for nom,classe in handles:
        f.write(f"\t_{nom}, _ := builder.GetObject(\"{nom}\")\n")
        f.write(f"\t{nom} = _{nom}.(*{classe})\n")
    f.write("}")

#	_playBtn, _ := builder.GetObject("playBtn")
#	playBtn := _playBtn.(*gtk.Button)