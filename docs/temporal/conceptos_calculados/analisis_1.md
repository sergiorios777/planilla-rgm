Confirmo el diagnóstico: el límite real no está en la idea de tener una tabla puente explícita (eso ya es buena dirección), sino en que la propuesta mezcla dos responsabilidades distintas en las mismas tablas — "qué es la fórmula" y "quién la rellena para este tenant" — y eso es justo lo que más adelante genera ambigüedad al ejecutar el cálculo. Antes de los detalles, te muestro cómo quedaría un modelo que separa esas responsabilidades:## El problema percibido está bien identificado

Tus dos enfoques previos (tabla maestro-maestro + switch-case, y luego mapas régimen→clave→código SUNAT) tienen el mismo defecto de fondo: el "saber qué concepto alimenta a qué variable" vive en código o en una capa intermedia que ningún administrador de municipalidad puede tocar. Eso es exactamente lo que describes como la limitación: no hay manera de que el tenant *seleccione* su propia base de cálculo cuando su planilla no calza con el modelo estándar. Migrar eso a una tabla relacional explícita es el movimiento correcto.

## Lo que tu propuesta hace bien

- **La idea central — una tabla puente inequívoca** — es sólida. Evita búsquedas indirectas y permite que una variable de cálculo se alimente de *varios* conceptos sumados (tu ejemplo de `remuneracion_computable` con dos filas para el mismo `concepto_calculado_id` lo deja claro, y es importante conservar eso).
- **Mantener la fórmula matemática en Go** (vía `variable_interna`) en lugar de modelar toda la fórmula como datos es la decisión correcta. Construir un motor de reglas genérico sería sobre-ingeniería para algo que, como bien dices, cambia muy poco. El dato solo debe resolver *qué conceptos* alimentan cada variable — no *cómo* se combinan matemáticamente.
- **Ya corregiste algo importante entre tu boceto en markdown y tu DDL en SQL**: en el markdown, `conceptos_calculados` mezcla el beneficio con el régimen (una fila por combinación), mientras que en el SQL el régimen vive en `conceptos_base_regimen`, no en el catálogo. Esta segunda forma es la correcta — quédate con ella. Si dejaras el régimen en `conceptos_calculados`, tendrías que duplicar "CTS" una vez por cada régimen laboral, lo cual es justo la ambigüedad que quieres evitar.

## Los puntos que sí están "muy simples" todavía

**1. La doble FK (`concepto_modelo_id` + `concepto_tenant_id`) en la misma fila es el problema más serio.**
Si `concepto_tenant_id` apunta a una fila de `conceptos_tenant` (que ya tiene `tenant_id`), entonces esa fila de `conceptos_base_regimen` ya es específica de un tenant — no es un catálogo global. Pero al mismo tiempo guardas `concepto_modelo_id` ahí, sin un `tenant_id` propio en la tabla. Esto deja preguntas sin responder:
- ¿Cuál manda en tiempo de cálculo, el modelo o el tenant?
- ¿Cómo evitas que una fila tenga un `concepto_tenant_id` que pertenece a un tenant distinto del que estás calculando? Sin una columna `tenant_id` explícita no puedes filtrar ni indexar de forma segura por tenant.
- ¿Quién llena esta tabla cuando se incorpora un municipio nuevo?

**2. Te falta separar "plantilla global" de "configuración real del tenant".**
Esa separación es justamente la que resuelve tu problema percibido. Si todo vive en una sola tabla con dos FKs opcionales, terminas con lógica de "coalesce" (si no hay tenant, usa modelo) dispersa en el motor de cálculo cada vez que corres planilla — frágil y difícil de auditar.

**3. No hay catálogo de "qué variables necesita cada concepto calculado".**
Hoy solo defines la asignación (`variable_calculo` → concepto), pero no el contrato (CTS en DL 276 *requiere* `remuneracion_computable` y `sexto_grati`, por ejemplo). Sin ese contrato no puedes validar, antes de correr planilla, si a un tenant le falta configurar una variable. Es la diferencia entre descubrir el error en producción al momento de pagar, o detectarlo en una pantalla de validación antes.

**4. `variable_calculo` como `VARCHAR` libre es frágil.**
Un typo entre `"REMUNERACION_COMPUTABLE"` en una fila y `"remuneracion_computable"` en el switch de Go produce un silencio total — la variable simplemente nunca se llena y nadie se entera hasta que el monto calculado sale en cero o mal.

**5. `ON DELETE CASCADE` en `concepto_tenant_id` es peligroso.**
Si un administrador del municipio desactiva o elimina un concepto tenant que está siendo usado como base de cálculo de CTS, la fila de `conceptos_base_regimen` desaparece silenciosamente y CTS deja de calcular esa variable sin ningún aviso.

**6. Detalles menores:** `regimentes_laborales` (typo, debería ser `regimenes_laborales`), `conceto_tenant_id` (typo), y faltan constraints `UNIQUE` para impedir filas duplicadas accidentales por (concepto_calculado, régimen, variable, concepto base).

## Una forma de resolverlo sin perder simplicidad

Lo que muestra el diagrama de arriba son cuatro tablas, cada una con una sola responsabilidad:

1. **`conceptos_calculados`** — catálogo global del beneficio/retención/aporte (tal cual tu DDL, sin régimen).
2. **`variables_calculo_def`** — el *contrato*: qué variables exige cada concepto calculado (ej. CTS exige `remuneracion_computable`, `sexto_grati`). Esto casi no cambia y vive cerca del código Go.
3. **`base_regimen_default`** — la *plantilla global* del SaaS: para un régimen y una variable, qué `concepto_modelo_id`(s) recomienda el sistema por defecto. Sin `tenant_id`, sin ambigüedad — es 100% catálogo.
4. **`base_regimen_tenant`** — la *configuración real* de cada municipio, con `tenant_id` explícito y `concepto_tenant_id`. Es la única tabla que lee el motor de cálculo al correr planilla. Se siembra automáticamente desde `base_regimen_default` cuando el tenant activa un régimen (resolviendo `concepto_modelo_id → concepto_tenant_id` vía el `modelo_id` que ya tiene `conceptos_tenant`), y luego el administrador municipal la edita libremente desde la UI sin tocar código.

Con esto:
- El motor de cálculo no necesita lógica de "si no hay override, usa el default" en tiempo de ejecución — esa resolución ocurre una sola vez, al sembrar.
- Puedes validar completitud antes de correr planilla: comparas `variables_calculo_def` contra `base_regimen_tenant` por tenant+régimen+concepto y detectas huecos de configuración.
- Resuelves el problema percibido de forma literal: el municipio elige sus conceptos base sin que nadie escriba una línea de Go.
- Sigue siendo solo cuatro tablas — no es un motor de reglas genérico, es exactamente la cantidad de estructura que tu problema real necesita.

Para el `ON DELETE` de `concepto_tenant_id`, te sugiero `RESTRICT` (o una validación previa en el backend que avise "este concepto es base de cálculo de CTS, ¿seguro que quieres desactivarlo?") en vez de `CASCADE`, para que la pérdida de una fila de cálculo sea una decisión consciente, no un efecto colateral.
