## Pregunta: ¿Cuáles son los requisitos y condiciones para recibir estos nuevos beneficios laborales?

Para recibir los nuevos beneficios laborales bajo el régimen del Decreto Legislativo N° 1057 (CAS), los trabajadores deben cumplir con los siguientes requisitos y condiciones establecidos por la normativa:

**Gratificaciones por Fiestas Patrias y Navidad**

- **Requisitos de percepción:** El servidor debe encontrarse **prestando servicios efectivos**, estar en uso de descanso vacacional, tener licencia con goce de remuneraciones o estar percibiendo subsidios de salud (Ley N° 26790) en el momento en que corresponda el pago.
- **Oportunidad de pago:** Se abonan dos veces al año, en la misma fecha en la que se paga la remuneración de los meses de **julio y diciembre**.
- **Casos proporcionales o truncos:** Si el trabajador finaliza su vínculo laboral antes de la fecha de pago, tiene menos de seis meses de servicio o se encuentra con licencia sin goce de haber, recibirá una gratificación proporcional a los meses y días efectivamente laborados.
- **Restricciones:** Si el trabajador labora en más de una entidad pública, solo recibirá la gratificación en la entidad que le abone el mayor ingreso. Además, estas gratificaciones son **incompatibles con la recepción de aguinaldos** u otros beneficios similares, no son pensionables y no sirven como base para calcular otros beneficios.

**Compensación por Tiempo de Servicios (CTS)**

- **Requisitos:** Se tiene derecho a percibir la CTS por cada año trabajado o por una **fracción mayor a seis meses** (la cual se computará como un año completo de servicios). Solo aplica para los casos en los que la extinción del vínculo laboral ocurra **después de la entrada en vigencia de la Ley N° 32563**.
- **Oportunidad de pago:** La CTS tiene efecto cancelatorio y **se paga únicamente al término del vínculo laboral**. La entidad debe realizar el pago como máximo en la siguiente fecha en la que ordinariamente abona remuneraciones.
- **Cálculo:** Se calcula utilizando la última remuneración mensual percibida por el trabajador al terminar su vínculo y tomando en cuenta los días de servicio efectivo, vacaciones, licencias pagadas y subsidios de salud. Al igual que las gratificaciones, la CTS no es remunerativa ni pensionable.

**Condición especial: Implementación gradual** Tanto las gratificaciones como la CTS no se pagarán al 100% de manera inmediata, sino que están sujetas a una **implementación progresiva para cada año fiscal**. La base de cálculo sobre la remuneración mensual seguirá el siguiente cronograma:

- **Año 2026:** 10% de la remuneración mensual.
- **Año 2027:** 20% de la remuneración mensual.
- **Año 2028:** 30% de la remuneración mensual.
- **Año 2029:** 50% de la remuneración mensual.
- **Año 2030 en adelante:** 100% de la remuneración mensual.

## Pregunta: ¿Cómo se calculan las gratificaciones proporcionales para periodos cortos?
Las gratificaciones proporcionales (también llamadas truncas en ciertos casos) se calculan en función estricta a los **meses y días efectivamente laborados** por el trabajador en el semestre correspondiente (de enero a junio para Fiestas Patrias, o de julio a diciembre para Navidad).

Existen tres escenarios puntuales donde se aplica este cálculo proporcional:

- **Trabajadores con menos de seis meses de servicio:** Si al llegar la fecha de pago (julio o diciembre) el servidor tiene menos de medio año trabajando en la entidad, se le abonará una gratificación proporcional basada en los meses y días que haya laborado de manera efectiva.
- **Extinción del contrato (Gratificación trunca):** Si el vínculo laboral termina antes de la fecha en que se debe depositar el beneficio, el trabajador tiene derecho a recibir un monto proporcional por los meses y días que alcanzó a trabajar dentro de ese semestre.
- **Personal con licencia sin goce de remuneraciones:** Si al momento del pago el trabajador se encuentra con una licencia no remunerada, recibirá una gratificación trunca que será proporcional a los meses y días en los que sí prestó servicios efectivos durante el semestre, monto que se pagará en los mismos meses de julio o diciembre.

Al realizar este cálculo, se deben tomar en cuenta dos **condiciones adicionales importantes**:

1. **Regla de progresividad:** Los porcentajes de la implementación gradual explicados anteriormente (10% en 2026, 20% en 2027, etc.) **también se aplican al monto resultante de las gratificaciones proporcionales y truncas**.
2. **Excepción para julio de 2026:** De manera excepcional, para la gratificación por Fiestas Patrias de julio de 2026, el cálculo solo tomará en cuenta los **meses completos de servicios efectivamente prestados** desde que entró en vigencia la Ley Nº 32563, sin contabilizar días sueltos.

## Reglas propuestas para Gratificaciones y CTS
Para convertir los requisitos legales del Decreto Supremo Nº 142-2026-EF y la Ley Nº 32563 en una calculadora de planillas en Go, debemos estructurar las reglas de negocio en funciones lógicas y condicionales. A continuación, se detalla cómo modelar las **gratificaciones y la CTS** para los trabajadores CAS (Decreto Legislativo Nº 1057) basándonos en la normativa.

### 1. Definición del Estado del Trabajador (`Structs`)

Primero, necesitas definir un estado en Go que evalúe si el trabajador cumple con las condiciones para el cálculo de beneficios.

```go
type EstadoTrabajador string

const (
    ServicioEfectivo   EstadoTrabajador = "ServicioEfectivo"
    DescansoVacacional EstadoTrabajador = "DescansoVacacional"
    LicenciaConGoce    EstadoTrabajador = "LicenciaConGoce"
    SubsidioSalud      EstadoTrabajador = "SubsidioSalud"
    LicenciaSinGoce    EstadoTrabajador = "LicenciaSinGoce"
    Cese               EstadoTrabajador = "Cese"
)

type TrabajadorCAS struct {
    FechaInicioVinculo time.Time
    FechaCese          *time.Time
    RemuneracionMensual float64
    EstadoActual       EstadoTrabajador
    MesesLaborados     int
    DiasLaborados      int
}
```

### 2. Regla de Implementación Gradual (Ambos beneficios)

Tanto la CTS como las gratificaciones están sujetas a una progresión porcentual por año fiscal. Esto se puede traducir en una función reutilizable:

```go
func obtenerPorcentajeGradual(anio int) float64 {
    switch anio {
    case 2026:
        return 0.10 // 10%
    case 2027:
        return 0.20 // 20%
    case 2028:
        return 0.30 // 30%
    case 2029:
        return 0.50 // 50%
    default:
        if anio >= 2030 {
            return 1.00 // 100%
        }
        return 0.00
    }
}
```

### 3. Reglas para Gratificaciones (Fiestas Patrias y Navidad)

**Regla A: Validación de Elegibilidad** Para recibir la gratificación, el programa debe verificar que el estado del trabajador en julio o diciembre sea válido: debe estar prestando servicios efectivos, en vacaciones, con licencia pagada o recibiendo subsidios de EsSalud.

```go
func esElegibleParaGratificacion(estado EstadoTrabajador) bool {
    return estado == ServicioEfectivo ||
           estado == DescansoVacacional ||
           estado == LicenciaConGoce ||
           estado == SubsidioSalud
}
```

**Regla B: Determinación de la Base de Cálculo** La base de cálculo siempre será la remuneración mensual percibida al **30 de junio** (para Fiestas Patrias) y al **30 de noviembre** (para Navidad).

**Regla C: Cálculo Proporcional, Trunco y Excepción de Julio 2026** Si el trabajador cesa antes del pago o está con licencia sin goce, se le paga proporcionalmente a los meses y días efectivamente laborados. **Sin embargo, para julio 2026, existe una excepción estricta:** los días sueltos no se cuentan, solo meses completos.

```go
func calcularGratificacion(t TrabajadorCAS, mesPago int, anioPago int) float64 {
    if !esElegibleParaGratificacion(t.EstadoActual) && t.EstadoActual != Cese {
        return 0.0
    }

    // Regla excepcional para Fiestas Patrias (julio) 2026
    diasAComputar := t.DiasLaborados
    if mesPago == 7 && anioPago == 2026 {
        diasAComputar = 0 // Únicamente meses completos
    }

    porcentaje := obtenerPorcentajeGradual(anioPago)
    baseCalculo := t.RemuneracionMensual * porcentaje

    // Cálculo proporcional (fracciones de mes por sextos, días por treintavos de sexto)
    montoPorMeses := (baseCalculo / 6) * float64(t.MesesLaborados)
    montoPorDias := (baseCalculo / 6 / 30) * float64(diasAComputar)

    return montoPorMeses + montoPorDias
}
```

### 4. Reglas para Compensación por Tiempo de Servicios (CTS)

**Regla A: Desencadenante del Cálculo** La CTS **solo se calcula y se paga al momento de la extinción del vínculo laboral** (fecha de cese) a partir de la vigencia de la Ley Nº 32563.

**Regla B: Cómputo del Tiempo y Fracciones** El pago se realiza por cada año trabajado. La regla clave a programar es: **si la fracción sobrante de meses es mayor a seis (6) meses, el programa debe redondearlo computándolo como un año completo de servicios**. Los días efectivos incluyen trabajo, vacaciones, licencias con goce y subsidios.

```Go
func calcularCTS(t TrabajadorCAS, anioCese int, mesesRestantes int) float64 {
    // Solo se paga al cese
    if t.FechaCese == nil {
        return 0.0
    }

    // La base es la última remuneración mensual
    porcentaje := obtenerPorcentajeGradual(anioCese)
    baseCTS := t.RemuneracionMensual * porcentaje

    // Calcular años de servicio totales
    aniosCompletos := calcularAnios(t.FechaInicioVinculo, *t.FechaCese)

    // Regla: Fracción mayor a 6 meses se computa como 1 año completo
    if mesesRestantes > 6 {
        aniosCompletos += 1
    }

    return baseCTS * float64(aniosCompletos)
}
```

**Consideraciones Finales para tu Sistema:** Al diseñar el módulo financiero en Go, asegúrate de que estos conceptos **no se incluyan como base de cálculo para cargas sociales**, ya que la norma especifica que la CTS no está afecta a cargas sociales y las gratificaciones no constituyen base de cálculo para ningún reajuste u otro beneficio. Sin embargo, ten en cuenta que la entidad pública sí debe asumir internamente el aporte a EsSalud sobre las gratificaciones.