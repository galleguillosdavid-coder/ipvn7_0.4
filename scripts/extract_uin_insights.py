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
print("INSPECCION DE ip7uin_MVP_Spec_v1.1.docx")
print("================================================================================")
p2 = read_docx_paragraphs(r'G:\Mi unidad\03_Programacion\IPv7_y_Redes\ip7uin_MVP_Spec_v1.1.docx')
for i, p in enumerate(p2):
    t = p.strip()
    # print section headers, tables or key phrases
    if t.startswith("SECCIÓN") or t.startswith("SECCION") or (len(t) < 60 and any(k in t.lower() for k in ["arquitectura", "protocolo", "uin", "wire", "smart", "handshake", "fallo", "gap", "riesgo", "routing", "reputacion", "contrato"])):
        print(f"[{i}] {t}")

print("\n================================================================================")
print("INSPECCION DE IPv7_UIN_Arquitectura_Unificada_v4.docx")
print("================================================================================")
p1 = read_docx_paragraphs(r'G:\Mi unidad\03_Programacion\IPv7_y_Redes\IPv7_UIN_Arquitectura_Unificada_v4.docx')
for i, p in enumerate(p1):
    t = p.strip()
    if t.startswith("PARTE") or t.startswith("1.") or t.startswith("2.") or t.startswith("3.") or t.startswith("4.") or t.startswith("5.") or t.startswith("6.") or t.startswith("7.") or t.startswith("8."):
        if len(t) < 80:
            print(f"[{i}] {t}")
