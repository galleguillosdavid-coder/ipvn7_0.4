# MAGNA SKILL 7.0: Ecosistema AFE-Kùzu — Motor Soberano de Ingeniería Axiomática y Ejecución Continua

**Versión:** 7.0 (Actualizada con la Arquitectura ipvn7 v0.4.0)  
**Clasificación:** Marco Operativo de Ingeniería Axiomática, Grafo Unificado y Ejecución Continua  
**Implementación Activa:** [`pkg/l3/kuzu_graph.go`](file:///c:/Users/Frondabrick/Desktop/dvd/ipvn7/pkg/l3/kuzu_graph.go), [`pkg/l3/ai_copilot.go`](file:///c:/Users/Frondabrick/Desktop/dvd/ipvn7/pkg/l3/ai_copilot.go), `web/`

---

## 1. Identidad y Filosofía Central

Este marco operativo define un modelo determinista, visual, autogestionado y basado en grafos para el desarrollo de sistemas complejos. Fusiona la rigurosidad lógica de los diagramas de flujo estructurados con una infraestructura moderna en WSL2, Kùzu DB como única fuente de verdad unificada (código, base de conocimiento y principios axiomáticos), un pipeline estricto de Ingesta PFO (Principios-Funciones-Observaciones), micro-workers acotados (DeepSeek), ejecutabilidad continua desde el día uno y resolución proactiva de contradicciones lógicas.

---

## 2. Infraestructura y Entorno Obligatorio

* **Entorno WSL2 (Ubuntu en Windows):** Control total del sistema de archivos, ejecución de bases de datos y compilación cruzada nativa.
* **Kùzu Graph DB (Modelo Unificado):** Repositorio central que modela nodos de código, el sistema de decisiones, la memoria persistente y los Principios Axiomáticos, consultable en tiempo de ejecución (`.kuzu_db/`, `pkg/l3/kuzu_graph.go`).
* **Control de Versiones Git Base:** Commits atados estrictamente a los hitos lógicos del grafo.

---

## 3. Pipeline de Ingesta PFO (Principio - Función - Observación)

* **Principios (Nivel Axiomático):** Leyes lógicas y directrices maestras (incluyendo normativas como el `Zero-PII`, `Strict Core Freeze L0` o `Flujo Sostenible`). Todo el código del sistema debe descender directamente de estos nodos.
* **Funciones (Nivel Operativo):** Módulos, adaptadores y scripts que materializan y ejecutan los principios (`pkg/l1/`, `pkg/l4/`, `cmd/`).
* **Observaciones (Nivel Empírico):** Métricas de telemetría (Ring Buffer lock-free <28 ns), notas de campo y evaluaciones externas que retroalimentan el grafo para refinar las funciones sin alterar los axiomas principales.

```text
┌──────────────────────────────────────────────┐
│  PRINCIPIOS AXIOMÁTICOS (Invariables / L0)   │
│  Zero-PII · Core Freeze · Flujo Sostenible   │
└──────────────────────┬───────────────────────┘
                       │ Materializa
                       ▼
┌──────────────────────────────────────────────┐
│  FUNCIONES OPERATIVAS (Código / Módulos L1)  │
│  ZTNA · QoS · EBRA · Corporate VPN · Sphinx  │
└──────────────────────┬───────────────────────┘
                       │ Produce
                       ▼
┌──────────────────────────────────────────────┐
│  OBSERVACIONES EMPÍRICAS (Telemetría L2/L3)  │
│  Ring Buffer <28ns · RTT EWMA · Jitter       │
└──────────────────────────────────────────────┘
```

---

## 4. Protocolo de Coherencia Lógica y Resolución de Conflictos

* **Supervisión Axiomática:** Kùzu audita cada nueva instrucción antes de tocar el código para evitar colisiones estructurales o violaciones de principios.
* **Resolución Dual:** Ante una contradicción, el sistema aplica deducción lógica pura basada en el grafo o interrumpe de inmediato la ejecución para exigir una validación directa del usuario, protegiendo la integridad de la arquitectura (`ValidateDirectiveAgainstAxioms`).

---

## 5. Ejecución Continua y Aislamiento por Comentarios

* **Ejecutabilidad Perpetua desde el Día 1:** El programa compila y corre de punta a punta desde la primera interacción. Los módulos futuros no bloquean el flujo: se integran mediante stubs, mocks o descripciones en pseudocódigo estrictamente acotadas por comentarios en su carril correspondiente.
* **Aislamiento de Componentes Críticos:** Respeto estricto por los componentes base protegidos (`pkg/l0/`), canalizando extensiones y adaptadores en las capas permitidas (`pkg/l1/`, `pkg/l4/`, `web/`) sin alterar el núcleo.

---

## 6. Integración de Micro-Workers (DeepSeek)

* **Aislamiento de Contexto:** DeepSeek se utiliza exclusivamente para microtareas atómicas y de baja complejidad.
* **Puente CLI/API Local:** Peticiones mediante scripts locales en WSL2 (`scripts/deepseek_worker.py`) que envían únicamente el fragmento de código afectado, sin contexto macro y excluyendo estrictamente cualquier módulo en fase de stub.

---

## 7. Distribución, Empaquetado y Actualización

* **Compilación Cruzada Dual:** Generación simultánea de binarios optimizados tanto para Windows (`.exe`) como para Linux desde la terminal de WSL2 (`scripts/build_dual.ps1`).
* **Auto-Actualización Nativa:** Mecanismo integrado de verificación y descarga transparente de versiones desde la arquitectura base.

---

## 8. Ciclo de Ejecución de Tareas (6 Fases)

1. **Fase de Sincronización Axiomática y de Grafo:** Consulta al grafo unificado en Kùzu para alinear los Principios, Funciones y Observaciones vigentes.
2. **Fase de Validación de Contradicciones:** Verificación lógica de la tarea actual frente al sistema axiomático para descartar conflictos.
3. **Fase de Acotación y Stubs Comentados:** Delimitación del código real frente a los componentes futuros simulados por mocks o pseudocódigo comentado.
4. **Fase de Implementación y Versionado:** Escritura del código ejecutable de punta a punta en WSL2 bajo el protocolo de linealidad y registro en Git.
5. **Fase de Verificación de Ejecución:** Comprobación de que el programa completo arranca y fluye de punta a punta sin interrupciones desde la primera línea (`go test ./...` -> PASS).
6. **Fase de Persistencia Estructural:** Actualización de los nodos, relaciones PFO y decisiones en Kùzu para preparar el siguiente ciclo de evolución.

---

## 9. Mapeo en el Código del Proyecto ipvn7 (v0.4.0)

| Eje del Skill 7.0 | Módulo en el Repositorio | Función en Tiempo de Ejecución |
| :--- | :--- | :--- |
| **Grafo PFO & Decisiones** | [`pkg/l3/kuzu_graph.go`](file:///c:/Users/Frondabrick/Desktop/dvd/ipvn7/pkg/l3/kuzu_graph.go) | Nodos `AxiomPrinciple`, `OperationalFunction`, `EmpiricalObservation`, `EngineeringDecision`. |
| **Consultas openCypher** | [`pkg/l3/kuzu_bridge.go`](file:///c:/Users/Frondabrick/Desktop/dvd/ipvn7/pkg/l3/kuzu_bridge.go) | Consultas topológicas ultrarrápidas y auditoría de directivas. |
| **Micro-Worker DeepSeek** | `scripts/deepseek_worker.py`, [`pkg/l3/ai_copilot.go`](file:///c:/Users/Frondabrick/Desktop/dvd/ipvn7/pkg/l3/ai_copilot.go) | Inferencia contextual aislada y diagnóstico de causas raíz. |
| **Interfaz Nivel 7 (Ingeniero)** | `web/index.html`, `web/app.js` | Consola openCypher en tiempo real, HUD de eBPF/XDP y anulación Override. |
| **Compilación Dual WSL2** | `scripts/build_dual.ps1`, `install.sh`, `install.ps1` | Binarios multiplataforma duales sin fricción de despliegue. |
