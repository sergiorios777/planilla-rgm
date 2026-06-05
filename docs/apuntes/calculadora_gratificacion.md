# Calculadora de Gratificación DL 728
Para el régimen privado (D.L. 728), la gratificación legal equivale a un sueldo completo por cada semestre laborado (enero-junio y julio-diciembre).
El monto final a recibir incluye la gratificación base y una bonificación extraordinaria (9% si estás en EsSalud o 6.75% si tienes EPS).

## El cálculo

1. **Fórmula básica (semestre completo)**
* **Gratificación Base**: Sueldo básico + Asignación familiar (si aplica)
* **Bonificación Extraordinaria**: (Gratificación Base x 0.09) (o 6.75% según tu seguro)
* **Total a recibir**: Gratificación Base + Bonificación Extraordinaria

**Ejemplo práctico:**

Si ganas S/ 2,000 y tienes asignación familiar (S/ 102.50), tu sueldo computable es S/ 2,102.50.

* **Gratificación Base**: S/ 2,102.50
* **Bonificación Extraordinaria (EsSalud 9%)**: S/ 189.23
* **Total recibido**: S/ 2,291.732.

**Cálculo proporcional (menos de 6 meses)**

Si no laboraste el semestre completo, el cálculo se hace por cada mes calendario completo trabajado. 

**La fórmula es:**

Gratificación Base = ((Remuneración computable) / 6) x (Meses laborados)

*Por ejemplo*: Si ingresaste a trabajar el 1 de abril y calculamos la gratificación de julio, laboraste 3 meses. 

Si tu sueldo computable es S/ 2,000:
(2000/6) x 3 = S/ 1,000 de gratificación base.

## Consideraciones adicionales

* El valor de la gratificación se debe guardar en la planilla_detalles para los conceptos del tenant asociadoa los conceptos maestros de:
  - Gratificación de Julio (codigo_interno: GRATI_JUL_DL_728) o Gratificación de Diciembre (codigo_interno: GRATI_DIC_DL_728)
  - Bonificación extraordinaria de Julio (codigo_interno: BON_EXTR_JUL_DL_728) o Bonificación extraordinaria de Diciembre (codigo_interno: BON_EXTR_DIC_DL_728)
* La gratificación y bonificación extraordinaria se debe mostrar en la planilla y boleta del trabajador.
* La estructura de las tablas y sus relaciones los puedes revisar en 'docs\temporal\planilla_rgm.sql'.
* Dejemos abierta la posibilidad de que en el futuro podamos agregar calculadora para otro régimen laboral.
* Puedes observar los conceptos base relacionados en config.ConceptosBaseGratificaciones en 'internal\config\plantillas_conceptos.go'.