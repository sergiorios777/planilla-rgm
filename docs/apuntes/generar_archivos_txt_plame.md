# Generar archivos txt SUNAT PDT/Plame

Son dos los archivos principales que va a generar la aplicación:
1. Jornada laboral.
2. Remuneraciones.

## Precisiones sobre los nombres de archivo

Para que el PDT PLAME reconozca e importe tus archivos planos de forma masiva, debes nombrarlos siguiendo una estructura paramétrica obligatoria impuesta por la SUNAT. Si cometes un error en un solo carácter, el sistema emitirá un error de rechazo.

La estructura general del nombre es la siguiente:
$$\mathbf{0601 + A\tilde{N}O + MES + RUC + EXTENSI\acute{O}N}$$ 
------------------------------
### Componentes fijos del nombre

* 0601: Es el código fijo del PDT Planilla Electrónica. Siempre va al inicio.
* Año (YYYY): Cuatro dígitos del año de la declaración (Ejemplo: 2026).
* Mes (MM): Dos dígitos del mes de la declaración (Ejemplo: 05 para mayo).
* RUC: Los 11 dígitos del Registro Único de Contribuyentes de tu empresa.
* Extensión: Las tres letras finales que indican al sistema qué tipo de información contiene el archivo.

------------------------------
### Los 3 nombres exactos según el tipo de archivo
Dependiendo de los datos que vas a subir, renombra tus archivos .txt exactamente así:
#### 1. Archivo de Jornada Laboral

* Extensión: .jor
* Ejemplo real: Si declaras mayo de 2026 y tu RUC es 20123456789, el archivo debe llamarse:
060120260520123456789.jor

#### 2. Archivo de Remuneraciones, Ingresos y Descuentos

* Extensión: .rem
* Ejemplo real: Para el mismo periodo y RUC anterior, el archivo debe llamarse:
060120260520123456789.rem

#### 3. Archivo de Días No Laborados / Subsidiados (Opcional)

* Extensión: .snl
* Ejemplo real: Solo se usa si tienes descansos médicos, licencias o inasistencias:
060120260520123456789.snl
* No lo vamoa a crear todavía, solo lo mostraremos como un línea de texto simple en las opciones de descarga, será para una siguiente versión.

------------------------------
#### ⚠️ Regla crítica para Windows
Asegúrate de que tu sistema operativo no duplique la extensión. Si tu explorador de archivos tiene oculta la opción "Ver extensiones de nombre de archivo", podrías nombrar tu archivo sin querer como 060120260520123456789.jor.txt, lo cual causará un error inmediato en el PLAME. El formato final debe ser únicamente plano.
------------------------------

## Estructura de las columnas de los archicos Jornada Laboral y Remuneraciones

Para la importación masiva al PDT PLAME se necesitan principalmente dos archivos planos (.txt). La estructura exacta de columnas se define por la cantidad de palotes o caracteres de separación pipe (|).
Cada fila debe terminar con un palote (|) y un salto de línea. Los campos vacíos obligatorios se representan dejando el espacio entre palotes en blanco (||). [1] 
------------------------------
### 1. Estructura de Jornada Laboral (Nombre de archivo: 0601YYYYMMRUC.jor)
Este archivo detalla los días y horas trabajadas, así como las horas extras de cada empleado.

| N° Columna [2] | Nombre del Campo | Tipo de Dato / Longitud | Descripción / Valores |
|---|---|---|---|
| 1 | Tipo de Documento | Numérico (2) | 01 = DNI, 04 = Carné Extranjería, 07 = Pasaporte |
| 2 | Número de Documento | Alfanumérico (15) | Número de identidad del trabajador |
| 3 | Horas Ordinarias | Numérico (3) | Total de horas laboradas en el mes |
| 4 | Minutos Ordinarios | Numérico (2) | Minutos restantes de la jornada |
| 5 | Horas Sobretiempo | Numérico (3) | Total de horas extras acumuladas |
| 6 | Minutos Sobretiempo | Numérico (2) | Minutos restantes de sobretiempo |
| 7 | Días No Laborados | Numérico (2) | Días subsidiados o con inasistencia (Ej: 00) |

Ejemplo de línea de texto (.txt):

```txt
01|44556677|192|00|010|30|00|
```
------------------------------
### 2. Estructura de Remuneraciones (Nombre de archivo: 0601YYYYMMRUC.rem)
Este archivo asocia los montos calculados a los códigos de conceptos de la SUNAT (Ingresos, Descuentos y Aportes). Nota: Si un trabajador tiene 5 conceptos diferentes (Sueldo, Asignación Familiar, AFP, etc.), tendrá 5 filas independientes en este archivo.

| N° Columna [3] | Nombre del Campo | Tipo de Dato / Longitud | Descripción / Valores |
|---|---|---|---|
| 1 | Tipo de Documento | Numérico (2) | 01 = DNI, 04 = Carné Extranjería, etc. |
| 2 | Número de Documento | Alfanumérico (15) | Número de identidad del trabajador |
| 3 | Código de Concepto | Numérico (4) | Códigos SUNAT (Ej: 0121 Sueldo, 0201 Asig. Familiar) |
| 4 | Monto Devengado | Decimal (8.2) | Monto ganado en el mes (Ej: 1025.00) |
| 5 | Monto Pagado | Decimal (8.2) | Monto efectivamente pagado (Ej: 1025.00) |

Ejemplo de líneas de texto (.txt) para un solo trabajador:

```txt
01|44556677|0121|1500.00|1500.00|
01|44556677|0201|102.50|102.50|
01|44556677|0601|180.00|180.00|
```
------------------------------

## PRECISIONES ADICIONALES

1. Para la Jornada Laboral tendremos como suposición el cumplimiento de las 240 horas mensuales exigidas por ley en el Perú, a los cuáles descontaremos lo días de inasistencia que no esté en estado pendiente cuando sea oportuno, por cada trabajador.
2. Agregaremos un botón de acción en la lista de planillas de 'ui\templates\tenant\planillas_ui.html' que muestre un dialog modal (para no saturar la vista con botones de acción) con las opciones de descarga de los archivos en con sus extensiones correspondientes o en formato zip (cada uno).
3. Puedes revisar una versión actualizadad de estructura de las tablas de la BD y sus relaciones en 'docs\temporal\planilla_rgm.sql'.
4. En el formato Remuneraciones:
   * Los códigos de concepto se refieren a la columna 'código' de la tabla conceptos_maestros.
   * Los montos devengado y pagado serán los mismos por simplicidad en esta versión.