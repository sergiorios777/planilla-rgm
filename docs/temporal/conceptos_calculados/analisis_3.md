Tu propuesta me parece **muy viable** y, más importante aún, apunta en la dirección correcta para un sistema de planillas SaaS destinado a municipalidades. Sin embargo, haría algunos ajustes antes de implementarla porque veo riesgos de acoplamiento entre la configuración de cálculo y los conceptos específicos del tenant.

## 1. El problema real que has identificado

Creo que tu diagnóstico es acertado.

Los enfoques anteriores tienen una limitación estructural:

### Enfoque 1: Conceptos maestros relacionados entre sí

```text
CTS
 ├─ Remuneración Principal
 ├─ Asignación Familiar
 └─ Bonificación X
```

Problema:

* No considera regímenes laborales.
* No considera personalizaciones del tenant.
* No considera excepciones municipales.

---

### Enfoque 2: Mapas en Go

```go
map[regimen]map[variable][]codigoSunat
```

Problema:

* El conocimiento del negocio vive en código.
* Cada cambio requiere despliegue.
* El usuario no puede adaptar la configuración.

---

Y aquí llegamos al punto clave:

> La composición de una base de cálculo es información de negocio, no lógica de programación.

Por tanto debería vivir en base de datos.

---

# 2. Lo que más me gusta de tu propuesta

La idea de introducir:

```sql
conceptos_calculados
```

y

```sql
conceptos_base_regimen
```

es básicamente convertir el motor de cálculo en un sistema "data-driven".

Eso es exactamente lo que hacen muchos ERP de planillas.

---

Ejemplo:

### Vacaciones truncas DL 276

```text
REMUNERACION_COMPUTABLE =
    Remuneración Básica
  + Asignación Familiar
  + Bonificación Diferencial
```

La calculadora ya no necesita saber qué conceptos intervienen.

Simplemente pregunta:

```sql
SELECT *
FROM conceptos_base_regimen
WHERE regimen_id = ?
AND concepto_calculado_id = ?
```

y obtiene:

```text
REMUNERACION_COMPUTABLE
 ├─ Básica
 ├─ Familiar
 └─ Diferencial
```

Eso elimina muchísimo hardcode.

---

# 3. El único punto que modificaría

Aquí veo un posible problema:

```sql
conceptos_base_regimen
(
    ...
    concepto_modelo_id
    concepto_tenant_id
)
```

No almacenaría ambos.

Porque introduces duplicidad.

---

## Escenario

Supongamos:

```text
Modelo:
Remuneración Básica
id=10
```

Tenant A:

```text
id=118
modelo_id=10
```

Tenant B:

```text
id=220
modelo_id=10
```

Entonces:

```sql
conceptos_base_regimen
```

tendría que almacenar distintas filas según el tenant.

Eso rompe la idea de configuración global.

---

Yo haría:

```sql
conceptos_base_regimen
(
    id,
    concepto_calculado_id,
    regimen_id,
    variable_calculo,
    concepto_modelo_id
)
```

Solamente.

---

Y cuando el motor calcule para un tenant:

```sql
SELECT ct.id
FROM conceptos_tenant ct
WHERE ct.modelo_id = ?
AND ct.tenant_id = ?
```

obtiene automáticamente el espejo.

---

Así:

```text
ADMIN SaaS
 ↓
Configura conceptos modelo

TENANT
 ↓
Hereda automáticamente
```

Mucho más limpio.

---

# 4. Haría una tabla adicional

En vez de:

```sql
variable_calculo VARCHAR(50)
```

consideraría:

```sql
variables_calculo
```

---

Ejemplo:

```sql
CREATE TABLE variables_calculo(
    id SERIAL PRIMARY KEY,
    codigo VARCHAR(50) UNIQUE,
    nombre VARCHAR(150)
);
```

Datos:

```text
REMUNERACION_COMPUTABLE
SEXTO_GRATI
PROMEDIO_COMISIONES
BASE_IMPONIBLE
```

---

Luego:

```sql
conceptos_base_regimen
(
    ...
    variable_calculo_id
)
```

Ventajas:

* Evitas errores tipográficos.
* Puedes documentar variables.
* Facilita construir UI.

---

# 5. Separaría "qué se calcula" de "cómo se calcula"

Aquí veo el siguiente desafío.

Tu tabla responde:

> ¿Qué conceptos participan?

Pero no responde:

> ¿Cómo se combinan?

---

Ejemplo CTS:

```text
CTS =
(
 Remuneración Computable
 +
 Sexto de Gratificación
)
/ 12
```

La tabla propuesta sólo resuelve:

```text
Remuneración Computable =
 conceptos X,Y,Z
```

pero no la fórmula final.

---

Yo agregaría:

```sql
conceptos_calculados
```

con algo como:

```sql
formula_codigo
```

Ejemplo:

```text
CTS
VAC_TRUNCA
GRATI
ESSALUD
ONP
AFP
```

---

En Go:

```go
switch formulaCodigo {

case "CTS":
    return CalcularCTS(...)

case "VAC_TRUNCA":
    return CalcularVacacionTrunca(...)

}
```

Observa que:

* La composición de la base es configurable.
* La matemática sigue siendo código.

Eso es importante.

Porque no recomendaría convertir fórmulas laborales peruanas completas en expresiones almacenadas en BD.

Sería demasiado complejo para el beneficio obtenido.

---

# 6. Sobre los flags actuales

Veo varios:

```text
es_pensionable
es_remunerativa
es_base_cts
es_base_beneficios_sociales
es_afecto_cargas_sociales
```

Aquí percibo un problema futuro.

Estás modelando afectaciones mediante columnas booleanas.

Funciona para:

```text
CTS
Sí/No
```

pero no escala.

---

Imagina que mañana agregas:

```text
Base Gratificación
Base Vacaciones
Base Escolaridad
Base Renta 5ta
Base SCTR
Base CAFAE
```

Terminarías con:

```text
es_base_cts
es_base_gratificacion
es_base_vacaciones
es_base_essalud
es_base_renta5
...
```

---

Tu nueva propuesta reemplaza elegantemente eso.

Yo incluso pensaría en descontinuar gradualmente:

```text
es_base_cts
es_base_beneficios_sociales
```

y migrar todo hacia:

```sql
conceptos_base_regimen
```

porque es infinitamente más flexible.

---

# 7. Lo que yo implementaría

Mantendría:

```sql
conceptos_calculados
```

```sql
variables_calculo
```

```sql
conceptos_base_regimen
```

Pero con esta estructura:

```sql
CREATE TABLE conceptos_calculados(
    id SERIAL PRIMARY KEY,
    nombre VARCHAR(150),
    tipo VARCHAR(50),
    formula_codigo VARCHAR(50) UNIQUE
);
```

```sql
CREATE TABLE variables_calculo(
    id SERIAL PRIMARY KEY,
    codigo VARCHAR(50) UNIQUE,
    nombre VARCHAR(150)
);
```

```sql
CREATE TABLE conceptos_base_regimen(
    id SERIAL PRIMARY KEY,

    concepto_calculado_id INT NOT NULL,
    regimen_id INT NOT NULL,

    variable_calculo_id INT NOT NULL,

    concepto_modelo_id INT NOT NULL
);
```

Y el flujo sería:

```text
Motor de cálculo
        ↓
Concepto calculado (CTS)
        ↓
Regimen laboral (728)
        ↓
Variables requeridas
        ↓
Conceptos modelo asociados
        ↓
Conceptos tenant espejo
        ↓
Montos
        ↓
Fórmula Go
        ↓
Resultado
```

## Conclusión

Considero que tu propuesta es una evolución correcta respecto a los dos enfoques anteriores. Te permitirá:

* Eliminar gran parte del hardcode.
* Soportar diferencias por régimen laboral.
* Dar capacidad de personalización a las municipalidades.
* Mantener la lógica matemática crítica en Go, donde es más segura y mantenible.

El único cambio importante que haría es **no almacenar simultáneamente `concepto_modelo_id` y `concepto_tenant_id` en la tabla de afectaciones**, sino usar únicamente `concepto_modelo_id` como fuente canónica y resolver el espejo del tenant durante el cálculo. Eso mantiene la consistencia del modelo SaaS y evita problemas de sincronización a largo plazo.

