# 📋 Plan de Implementación de Prorrateo y Renta 5ta
## Fase 1: El Prorrateo de Días (El cimiento)
Antes de arreglar los impuestos, debemos arreglar los ingresos.
   1. Agregaremos `FechaInicio` (y `FechaFin`) al modelo `ContratoPlanilla`.
   2. En la consulta SQL de la Fase B (`ObtenerContratosActivosPlanilla`), extraeremos esas fechas.
   3. En la Pasada 1 de `planilla_service.go`, crearemos una lógica que calcule los "Días Computables" del mes. Si el mes tiene 30 días, pero el trabajador inició el día 11, sus días computables son 20.
   4. Multiplicaremos su sueldo base por `(Días Computables / 30)`.

## Fase 2: El Historial de Ingresos (La memoria del motor)
Para que el mes de abril sepa cuánto cobró el trabajador en enero, febrero y marzo.
   1. Crearemos una nueva función masiva en el Repositorio: `ObtenerIngresosPreviosMasivo()`. Hará exactamente lo mismo que hicimos con las retenciones, pero sumará los ingresos (Sueldos, Bonos, etc.) de los meses anteriores del mismo año.
   2. Agregaremos este dato al "Maletín" (`ContextoCalculo`).

## Fase 3: La Corrección de la Calculadora de 5ta
Con los datos reales en el Maletín, ajustaremos tu archivo `renta5ta.go`.
   1. La proyección tomará el sueldo completo (sin prorratear) y lo multiplicará por los meses faltantes.
   2. A ese resultado, le sumará el "Historial de Ingresos" de la Fase 2.