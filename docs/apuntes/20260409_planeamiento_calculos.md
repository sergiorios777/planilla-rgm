# Plan de implementación de cálculos

## La Visión General: ¿Cómo resolveremos el problema?
El problema principal es que el cálculo de un concepto (Ej. EsSalud) no depende solo del monto de los ingresos, sino del Contexto.

Para calcular, el motor necesita hacerle 3 preguntas al sistema:

1. **El Entorno:** ¿En cuánto está la UIT este año? (Viene de la BD).
2. **La Base:** ¿Qué ingresos suman para este concepto? (Viene de `conceptos_afectaciones`).
3. **El Trabajador:** ¿En qué régimen está? ¿Tiene suspensión de 4ta? (Viene del Contrato/Puesto).

Nuestra estrategia híbrida dividirá responsabilidades: la Base de Datos guardará los datos que cambian anualmente, y Go tendrá pequeños "archivos expertos" separados, donde cada uno sabrá hacer un solo cálculo matemático perfecto.

## Fases del plan

### Fase 1: Preparación de la Base de Datos (Datos y Contexto)
Antes de calcular, necesitamos guardar las variables.

* #### Paso 1.1: Tabla `parametros_anuales`.
  * _Qué haremos:_ Crear una tabla simple `(anio, nombre_parametro, valor)`.
  * _Por qué:_ Para guardar variables como `UIT_2026 = 5150`, `TASA_ESSALUD = 0.09`. Si en 2027 la UIT sube, solo insertamos una fila en la BD, ¡sin tocar el código Go!

* #### Paso 1.2: El Contexto del Trabajador.
  * _Qué haremos:_ Asegurarnos de que cuando hacemos la consulta de contratos en el Motor, traigamos información vital: El régimen laboral del Puesto (CAS, 276, 728) y si el trabajador tiene marca de "Suspensión de 4ta".

### Fase 2: El Motor de Reglas en Go (Patrón "Strategy")
Aquí aplicaremos ingeniería de software de alto nivel. No vamos a poner miles de `if / else` dentro del motor principal.

* #### Paso 2.1: Crear el Contexto (El "Maletín").
  * _Qué haremos:_ Crear una estructura en Go (`ContextoCalculo`) que agrupe toda la info de la persona: Sus ingresos de la Pasada 1, su régimen, y los parámetros del año.

* #### Paso 2.2: La Fábrica de Calculadoras.
  * _Qué haremos:_ Crearemos una carpeta nueva en Go llamada `calculadoras/`. Adentro, haremos archivos independientes: `essalud.go`, `renta4ta.go`, etc.
  * _Por qué:_ Si mañana cambia la ley de EsSalud, solo abres `essalud.go`, modificas la fórmula matemática, y no tienes miedo de romper la Renta de 5ta, porque están aislados.

### Fase 3: La Refactorización del Motor (El Algoritmo de 2 Pasadas)
Modificaremos la función `ProcesarPlanilla` que hicimos hoy para que funcione en dos tiempos:

* #### Paso 3.1: Pasada 1 (Dinero en Mano).
  * _Qué haremos:_ Leer todos los `INGRESOS` fijos del trabajador y sumarlos. Pero esta vez, los guardaremos en un "mapa en memoria" (Ej. `[0121] = 2500`, `[0914] = 300`).

* #### Paso 3.2: Pasada 2 (El Cruce de Afectaciones).
  * _Qué haremos:_ Por cada Retención/Aporte (Ej. EsSalud - 0804), el motor irá a tu tabla conceptos_afectaciones y preguntará: "¿De cuáles códigos maestros saco la plata para el 0804?".
  * _Con esa lista, sumará los ingresos de la Pasada 1 para hallar la Base Imponible exacta.

* #### Paso 3.3: La Llamada a la Calculadora.
  * _Qué haremos:_ Le entregaremos la Base Imponible y el Contexto (Régimen) al archivo `essalud.go`, este hará su magia y nos devolverá el monto final a descontar (Ej. S/ 202.50).

## Un ejemplo conceptual de cómo se verá el código final en Go
Para que tu cerebro humano visualice a dónde vamos, mira lo limpio que quedará nuestro código en Go una vez implementemos el "Patrón Strategy" (Fase 2):

```go
// En essalud.go (Aislado y limpio)
func CalcularEsSalud(baseImponible float64, regimen string, tasa float64) float64 {
    if regimen == "CAS" {
        // Regla CAS: 9% del 45%
        return baseImponible * 0.45 * tasa 
    }
    // Regla 276/728: 9% del total
    return baseImponible * tasa
}

// En el Motor Principal (Pasada 2)
if concepto.CodigoMaestro == "0804" {
    // Calculamos cruzando datos
    montoFinal = calculadoras.CalcularEsSalud(baseHallada, trabajador.Regimen, parametros.TasaEsSalud)
}
```