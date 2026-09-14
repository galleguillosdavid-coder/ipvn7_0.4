import zipfile
import xml.etree.ElementTree as ET
import sys
import os

sys.stdout.reconfigure(encoding='utf-8', errors='replace')

def read_docx_paragraphs(path):
    with zipfile.ZipFile(path) as z:
        xml_content = z.read('word/document.xml')
    tree = ET.fromstring(xml_content)
    paragraphs = []
    for p in tree.iter('{http://schemas.openxmlformats.org/wordprocessingml/2006/main}p'):
        texts = [node.text for node in p.iter('{http://schemas.openxmlformats.org/wordprocessingml/2006/main}t') if node.text]
        if texts:
            paragraphs.append(''.join(texts))
    return paragraphs

print("================================================================================")
print("EXTRAYENDO DETALLES CLAVE DE ip7uin_MVP_Spec_v1.1.docx")
print("================================================================================")
doc_mvp = read_docx_paragraphs(r'G:\Mi unidad\03_Programacion\IPv7_y_Redes\ip7uin_MVP_Spec_v1.1.docx')

current_sec = "INTRO"
sec_content = {}
for p in doc_mvp:
    p_str = p.strip()
    if p_str.startswith("SECCIÓN") or p_str.startswith("SECCION"):
        current_sec = p_str
        sec_content[current_sec] = []
    elif current_sec in sec_content:
        if len(p_str) > 0:
            sec_content[current_sec].append(p_str)

for sec, lines in sec_content.items():
    print(f"\n--- {sec} --- (Lineas: {len(lines)})")
    # print up to 10 significant lines
    count = 0
    for l in lines:
        if any(w in l.lower() for w in ["decisión", "wire", "budget", "token", "anti-replay", "binding", "uin", "reputación", "identity", "routing", "handshake", "fallo", "riesgo", "prioridad"]):
            print(f"  > {l[:140]}")
            count += 1
            if count >= 8:
                break
