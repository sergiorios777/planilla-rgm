# Apuntes para calcular compensación vacacional por regímenes y por cese.

La decisión de que la liquidación se dispare a partir de la **baja del contrato** es excelente, ya que centraliza el evento de cese y garantiza que el trabajador no quede con saldos pendientes.

A continuación, traduzco los textos legales a **fórmulas matemáticas y reglas de negocio estrictas**, listas para ser implementadas mediante el patrón de diseño *Strategy* en Go.

## Variables Universales Necesarias para el Cálculo

Para que el motor en Go funcione para cualquier régimen, el módulo de liquidación (o el servicio de contratos al momento de dar la baja) deberá recolectar estas 4 variables:

1. **`RC`**: Remuneración Computable (varía según el régimen).
2. **`MesesT`**: Meses truncos laborados en el periodo actual que no llegó a cumplir un año.
3. **`DiasT`**: Días truncos laborados en el mes incompleto.
4. **`PeriodosPendientes`**: Cantidad de años (o periodos de 30 días) de vacaciones ganadas históricamente que el trabajador nunca gozó.

---

## 1. Régimen DL 276 (Carrera Administrativa)

Este régimen es muy estricto con el concepto de pago y tiene topes de acumulación definidos.

* **Remuneración Computable (`RC`):** MUC (Monto Único Consolidado) + BET (Beneficio Extraordinario Transitorio). *Nota: No ingresan otros conceptos.*
* **Regla de Vacaciones No Gozadas (VNG):** Se paga un MUC+BET por cada periodo ganado y no gozado, pero tiene un **límite estricto de 2 periodos**. Si el trabajador acumuló 3, el tercero se pierde financieramente.

$$VNG = \text{RC} \times \min(PeriodosPendientes, 2)$$


* **Regla de Vacaciones Truncas (VT):** Cálculo estándar proporcional.

$$VT = \left(\frac{\text{RC}}{12} \times MesesT\right) + \left(\frac{\text{RC}}{360} \times DiasT\right)$$



---

## 2. Régimen DL 1057 (CAS)

Es el régimen más sencillo de calcular, pero incluye una condición de "tiempo mínimo" que tu código en Go deberá validar antes de hacer la matemática.

* **Remuneración Computable (`RC`):** Retribución Mensual (Sueldo CAS).
* **Restricción de Tiempo Mínimo:** Si la diferencia entre la fecha de inicio del contrato y la fecha de cese es menor a **1 mes** (ej. trabajó 20 días y renunció), el pago de vacaciones truncas es **S/ 0.00**.
* **Regla de Vacaciones No Gozadas (VNG):** No hay límite legal explícito de acumulación en la norma citada.

$$VNG = \text{RC} \times PeriodosPendientes$$


* **Regla de Vacaciones Truncas (VT):**
Si el tiempo total en la entidad $\ge 1$ mes, se aplican los dozavos y treintavos.

$$VT = \left(\frac{\text{RC}}{12} \times MesesT\right) + \left(\frac{\text{RC}}{360} \times DiasT\right)$$



---

## 3. Régimen DL 728 (Actividad Privada)

El régimen más protector y, por ende, el más punitivo para la municipalidad si no gestiona bien los descansos (el famoso "triple vacacional").

* **Remuneración Computable (`RC`):** Remuneración Básica + Asignación Familiar + Promedio de comisiones/horas extras (si las hubiere).
* **Regla de Vacaciones Truncas (VT):**

$$VT = \left(\frac{\text{RC}}{12} \times MesesT\right) + \left(\frac{\text{RC}}{360} \times DiasT\right)$$


* **Regla de Vacaciones No Gozadas (VNG) e Indemnización:**
Aquí Go debe ser muy inteligente. Para los `PeriodosPendientes`, se debe evaluar **cuándo** se generó el derecho.
Si el trabajador cesa y tiene un periodo ganado que **ya venció** (es decir, pasó más de un año desde que ganó el derecho y no lo tomó), percibe la remuneración vacacional **más** la indemnización equivalente a una remuneración.

$$VNG_{simple} = \text{RC} \times \text{Periodos Ganados No Vencidos}$$


$$VNG_{indemnizado} = (\text{RC} \times 2) \times \text{Periodos Ganados Vencidos}$$



*Aclaración:* La ley habla de 3 pagos (trabajo realizado + vacación + indemnización), pero como en la liquidación el "trabajo realizado" ya se pagó en su momento en la planilla mensual regular, la deuda al cese es solo la vacación (1) + indemnización (1).

---

## 4. Régimen Ley 30057 (SERVIR)

Se alinea casi por completo con la lógica proporcional y elimina las penalidades complejas del 728.

* **Remuneración Computable (`RC`):** Valorización Principal + Valorización Ajustada.
* **Regla de Vacaciones No Gozadas (VNG):**
"Entrega económica vacacional" completa por el récord sin disfrutar.

$$VNG = \text{RC} \times PeriodosPendientes$$


* **Regla de Vacaciones Truncas (VT):**
Dozavos y treintavos estándar.

$$VT = \left(\frac{\text{RC}}{12} \times MesesT\right) + \left(\frac{\text{RC}}{360} \times DiasT\right)$$


## El Siguiente Paso Lógico

Estas reglas están perfectas para crear la interfaz `CalculadoraVacacional` en Go, donde cada régimen tendrá su propia estructura (`calculadora_cas`, `calculadora_276`, etc.) que implemente un método unificado `Calcular(rc, meses, dias, periodos)`.

Para que Go pueda realizar las operaciones de *Vacaciones No Gozadas* (VNG), el sistema necesita alimentar la variable `PeriodosPendientes` al momento de la liquidación. ¿La aplicación ya cuenta con un módulo de "Control de Asistencia/Permisos" que lleve la cuenta de cuántos días de vacaciones físicas ha tomado un trabajador en el año, o el Súper Admin ingresará este saldo pendiente manualmente en el modal al dar la baja del contrato?