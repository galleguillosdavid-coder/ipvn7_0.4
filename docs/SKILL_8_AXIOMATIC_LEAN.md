# SKILL 8.0: Axiomatic Lean Craftsmanship (ALC) — Ingeniería Frugal y Ejecución Continua

**Versión:** 8.0 (ipvn7 v0.4.0)  
**Clasificación:** Marco Operativo de Ingeniería de Redes Soberanas, Cero Desperdicio de Tokens y Concurrencia Robusta.  
**Estado:** Activo e Inviolable.

---

## 1. Misión y Filosofía Central

SKILL 8.0 redefine el modelo de trabajo para la construcción de **ipvn7 Network OS**. Preserva los axiomas matemáticos y la disciplina del modelado ontológico, pero **erradica la sobreingeniería, la fricción de intermediación y el gasto involuntario de tokens de Inteligencia Artificial**.

El principio rector es: **Toda verificación debe ser local, determinista e instantánea.**

```text
       ┌───────────────────────────────────────────────────────────┐
       │                   EL TRÍPTICO DE SKILL 8.0                │
       └─────────────────────────────┬─────────────────────────────┘
                                     ▼
      [1. ESPECIFICAR] ───► [2. IMPLEMENTAR] ───► [3. VERIFICAR]
      Invariante lógico      Código modular        Compilación + Tests
      en 3 líneas.           (Máx 400 líneas).     Go en < 1 segundo.
```

---

## 2. Los 5 Axiomas de Ingeniería de Código

### Axioma I: Frugalidad de Tokens & Autonomía Local (Zero Token Waste)
* **La IA es 100% opcional.** Ningún proceso crítico del sistema operativo, auto-curación de red, o rutina de desarrollo debe depender obligatoriamente de tokens de LLM.
* El formateo, detección de sintaxis, pruebas de regresión y benchmarks se resuelven localmente mediante herramientas nativas (`go test`, `go vet`, `gofmt`).

### Axioma II: La Regla de las 400 Líneas (Modularidad Atómica)
* Ningún archivo fuente nuevo o refactorizado debe exceder las 400 - 500 líneas de código.
* La responsabilidad única prevalece: si un componente adquiere dos razones para cambiar, se divide en submódulos especializados.

### Axioma III: Higiene de Concurrencia y Cero Fugas de Goroutines
* Toda goroutine debe nacer con un temporizador o un `context.Context` con cancelación garantizada:
  ```go
  go func(ctx context.Context) {
      defer cleanup()
      for {
          select {
          case <-ctx.Done():
              return
          case item := <-queue:
              handle(item)
          }
      }
  }(ctx)
  ```
* Todo commit y release debe superar `go test -race ./...`. Las condiciones de carrera se consideran fallos críticos inadmisibles.

### Axioma IV: Gestión de Errores Tipados y Cero Pánicos
* Queda estrictamente prohibido el uso de `panic()` en tiempo de ejecución. Los errores de red, datagramas corruptos o saturación de búferes deben gestionarse con errores centinela (`errors.Is()`).

### Axioma V: Frontend de Cero Dependencias Pesadas
* Interfaz construida sobre estándares web modernos (HTML5, Vanilla CSS con glassmorphism, Módulos ES6 nativos y Web Audio API).
* Prohibida la inclusión de gestores de paquetes con miles de dependencias en tiempo de ejecución (`node_modules`). El cliente web debe compilarse o servirse limpiamente y embeberse directamente en el binario autónomo de Go.

---

## 3. Ciclo de Ejecución de Tareas en 3 Pasos

1. **Paso 1 (Invariante):** Redactar en 3 líneas qué propiedades del sistema no pueden violarse (ej. Zero-PII, cuota de memoria de 1280B, apagado ordenado).
2. **Paso 2 (Cirugía Modular):** Escribir la implementación en su archivo específico, respetando el límite de tamaño y la tipificación estricta.
3. **Paso 3 (Verificación Empírica Inmediata):** Ejecutar `go test -v` localmente. Si pasa sin carreras y la interfaz responde en el puerto web, el hito está completado y certificado.
