## Descripción técnica del formato

**Resultado:** especificación reproducible de la hoja `'Liquidacion de beneficios'`, incluyendo estructura, estilos, dimensiones, cálculos y elementos gráficos para implementarla con Go/gofpdf.

### 1. Configuración general

- **Documento:** Liquidación de beneficios sociales.
- **Área principal:** `'Liquidacion de beneficios'!A1:K75`.
- **Papel:** A4 vertical.
- **Márgenes:** izquierdo/derecho 17.78 mm; superior/inferior 19.05 mm.
- **Lienzo original:** 713.25 × 1,119.75 pt.
- Para una sola página A4, aplicar escala uniforme aproximada de **65%**.
- Fuente predominante: **Calibri 11 pt**.
- Sin cuadrícula visible; los bordes se dibujan únicamente en encabezados y totales.

### 2. Paleta visual

|Uso|Color|
|---|---|
|Azul principal|`#0070C0`|
|Azul oscuro|`#002060`|
|Gris|`#808080`|
|Celeste auxiliar|`#DAEEF3`|
|Rojo|`#C00000`|
|Verde|`#00B050`|
|Blanco|`#FFFFFF`|
|Negro|`#000000`|

### 3. Anchos lógicos de columnas

Usar estos anchos en puntos y escalarlos al ancho imprimible:

```
[]float64{
    28.50, 60.00, 63.75, 66.00, 77.25,
    62.25, 75.75, 63.75, 66.75, 63.75, 84.75,
}
```

Corresponden a las columnas `A:K`.

Las columnas `Q:S` contienen cálculos auxiliares y tienen ancho cero; **no deben aparecer en el PDF**. `L:P` están vacías.

### 4. Alturas de filas

- Altura estándar: **14.25–15 pt**.
- Título, fila 1: **21 pt**.
- Filas descriptivas largas:  
    - fila 23: 19.5 pt
    - fila 50: 20.25 pt
- Filas de totales: aproximadamente **15.75 pt**.

### 5. Encabezado

|Rango|Contenido y formato|
|---|---|
|`A1:K1`|“LIQUIDACION DE BENEFICIOS SOCIALES”; fondo `#0070C0`, Arial Unicode MS 16, negrita, blanco, centrado|
|`A3:C8`|Etiquetas del trabajador; Calibri 11, negrita|
|`D3:F7`|Datos ingresados; fondo azul, texto blanco, negrita y centrado|
|`D8:F8`|Periodo trabajado; fondo gris, texto blanco, negrita cursiva|
|Zona superior derecha|Logotipo de 132.3 × 83.1 pt|
|Cuadro derecho|179.7 × 73.35 pt, centrado|

### 6. Bloques del documento

|Filas|Sección|
|---|---|
|`10:16`|1. Remuneración computable|
|`19:28`|2. Cálculo de CTS|
|`30:42`|3. Vacaciones truncas y descuento AFP/ONP|
|`46:55`|4. Gratificaciones truncas|
|`57:61`|5. Bonificación especial|
|`64:67`|Total general y monto en letras|
|`68:75`|Declaración, fecha, firma y DNI|

Los encabezados de sección usan fondo `#0070C0`, texto blanco, negrita y bordes superior/inferior azul oscuro de grosor medio.

### 7. Celdas combinadas principales

```text
A1:K1
A3:C3   D3:F3
A4:C4   D4:F4
A5:C5   D5:F5
A6:C6   D6:F6
A7:C7   D7:F7
A8:C8   D8:F8
C16:D16 G16:J16
C23:K23
I26:J26
C48:E48
C50:K50
G64:J64
C67:K67
C71:E71
```

### 8. Convenciones numéricas

- Importes: `#,##0.00`.
- Negativos: entre paréntesis, por ejemplo `(49,1)`.
- Porcentajes: `0.00%`, excepto bonificación especial: `0%`.
- Fechas operativas: `dd/mm/yyyy`.
- Total principal: dos decimales, blanco sobre azul oscuro.
- Moneda mostrada mediante texto contextual como “Nuevos Soles”; no utilizar el símbolo euro que aparece en algunos formatos internos de Excel.

### 9. Lógica de cálculo

```
Remuneración computable =
    sueldo básico
  + asignación familiar
  + promedio de gratificación
```

```text
CTS =
    remuneración computable / 12 × meses
  + remuneración computable / 12 / 30 × días
```

```text
Vacaciones brutas =
    remuneración vacacional / 12 × meses
  + remuneración vacacional / 12 / 30 × días

Vacaciones netas =
    vacaciones brutas - AFP/ONP
```

```
Gratificación trunca =
    remuneración / 6 × meses
  + remuneración / 6 / 30 × días
```

```text
Bonificación especial =
    gratificación trunca × 9%
```

```text
Total liquidación =
    CTS
  + vacaciones netas
  + gratificación trunca
  + bonificación especial
```

### 10. Elementos especiales

- En la selección previsional existen dos controles: **AFP** y **ONP**. En PDF deben sustituirse por círculos o casillas dibujadas.
- El cuadro AFP usa fondo `#DAEEF3`.
- `G64:K64` constituye el total general: fondo `#002060`, texto blanco, negrita cursiva.
- `C67:K67` muestra el monto en letras con fuente Consolas 11, negrita y rojo.
- `J74` muestra el nombre del trabajador sobre una línea de firma.
- `I75:J75` muestra “DNI:” y el número en rojo.

### 11. Estrategia sugerida para gofpdf

1. Definir una matriz con anchos de columnas y alturas de filas.
2. Convertir puntos a milímetros:

```go
mm := points * 0.352778
```

3. Multiplicar todas las coordenadas por una escala cercana a `0.65`.
4. Crear funciones reutilizables:
    - `drawMergedCell`
    - `drawSectionHeader`
    - `drawAmount`
    - `drawTotalRow`
    - `drawCalculationRow`
5. Dibujar primero fondos y bordes, luego texto y finalmente logotipo y controles.
6. Inyectar los datos del trabajador y los resultados calculados mediante una estructura, evitando posiciones codificadas dentro de la lógica de negocio.


---
## Contenido detallado de cada bloque

### 1. Título y datos del trabajador — `'Liquidacion de beneficios'!A1:K8`

#### Título

- `A1:K1`: **LIQUIDACIÓN DE BENEFICIOS SOCIALES**.
- Identifica el propósito legal y laboral del documento.

#### Datos personales y laborales

|Campo|Etiqueta|Valor actual|
|---|---|---|
|Trabajador|`A3:C3`|`D3:F3`: MIRANDA VALVERDE EDER|
|Cargo|`A4:C4`|`D4:F4`: CONTADOR|
|Fecha de ingreso|`A5:C5`|`D5:F5`: 01/01/2025|
|Fecha de cese|`A6:C6`|`D6:F6`: 01/04/2025|
|Motivo de cese|`A7:C7`|`D7:F7`: RENUNCIA VOLUNTARIA|
|Periodo trabajado|`A8:C8`|`D8:F8`: 0 años, 3 meses y 1 día|

El periodo trabajado se calcula mediante una fórmula basada en las fechas de ingreso y cese. Se descompone en años, meses y días utilizando el criterio financiero de meses de 30 días.

---

## 2. Remuneración computable — `'Liquidacion de beneficios'!A10:K16`

Este bloque determina la remuneración base que se utilizará para calcular la CTS.

### Encabezado

- `A10`: número de sección **1.-**
- `B10:J10`: **REMUNERACIÓN COMPUTABLE**
- `K10`: encabezado **MONTO**

### Conceptos remunerativos

|Concepto|Celda descriptiva|Monto|
|---|---|---|
|Sueldo básico|`C12`|`K12`: S/ 1,400.00|
|Asignación familiar|`C13`|`K13`: S/ 75.00|
|Promedio de gratificación percibida, equivalente a 1/6|`C15`|`K15`: S/ 300.00|
|**Total remuneración computable**|`G16:J16`|`K16`: **S/ 1,775.00**|

La fórmula principal es:

```excel
=SUMA(K12:K15)
```

El total de `K16` se usa posteriormente como base para la CTS.

---

## 3. Cálculo de la CTS — `'Liquidacion de beneficios'!A19:K28`

Este bloque calcula la **Compensación por Tiempo de Servicios trunca** pendiente al momento del cese.

### Periodo semestral de CTS

|Elemento|Ubicación|Contenido actual|
|---|---|---|
|Sección|`A19:B19`|2.- CÁLCULO DE LA CTS|
|Descripción del depósito|`C21`|CTS a depositar el 15 de mayo|
|Inicio del semestre|`C22`|01/11/2024|
|Fin del semestre|`D22`|30/04/2025|
|Periodo efectivo calculado|`C23:K23`|Desde 01/01/2025 hasta 01/04/2025|

El formato identifica automáticamente si la CTS corresponde al depósito de mayo o noviembre.

### Cálculo por meses

La fila `24` representa:

```text
S/ 1,775.00 ÷ 12 × 3 meses = S/ 443.75
```

|Componente|Celda|
|---|---|
|Remuneración computable|`C24`|
|Divisor anual|`E24`: 12|
|Meses computables|`G24`: 3|
|Resultado|`K24`: S/ 443.75|

### Cálculo por días

La fila `25` representa:

```text
S/ 1,775.00 ÷ 12 ÷ 30 × 1 día = S/ 4.93
```

|Componente|Celda|
|---|---|
|Remuneración computable|`C25`|
|Divisor anual|`E25`: 12|
|Divisor mensual|`G25`: 30|
|Días computables|`I25`: 1|
|Resultado|`K25`: S/ 4.93|

### Resultado de CTS

|Resultado|Ubicación|Importe|
|---|---|---|
|Total calculado|`I26:K26`|S/ 448.68|
|**CTS por pagar**|`H28:K28`|**S/ 448.68**|

```excel
=SUMA(K24:K25)
```

---

## 4. Vacaciones truncas — `'Liquidacion de beneficios'!A30:K42`

Calcula la compensación por vacaciones generadas y no gozadas hasta la fecha de cese.

### Base vacacional

La remuneración usada es:

```text
Sueldo básico + asignación familiar
S/ 1,400.00 + S/ 75.00 = S/ 1,475.00
```

No incluye el promedio de gratificación utilizado para la CTS.

### Cálculo por meses

La fila `32` representa:

```text
S/ 1,475.00 ÷ 12 × 3 meses = S/ 368.75
```

|Componente|Celda|
|---|---|
|Remuneración vacacional|`C32`|
|Divisor anual|`E32`: 12|
|Meses computables|`G32`: 3|
|Resultado|`K32`: S/ 368.75|

### Cálculo por días

La fila `33` representa:

```
S/ 1,475.00 ÷ 12 ÷ 30 × 1 día = S/ 4.10
```

|Componente|Celda|
|---|---|
|Remuneración vacacional|`C33`|
|Divisor anual|`E33`: 12|
|Divisor mensual|`G33`: 30|
|Días computables|`I33`: 1|
|Resultado|`K33`: S/ 4.10|

### Vacaciones brutas

```text
S/ 368.75 + S/ 4.10 = S/ 372.85
```

El importe bruto aparece en `I34`.

### Descuento previsional

El documento permite seleccionar entre:

- **AFP**, opción 1.
- **ONP**, opción 2.

La selección se controla mediante `A36`.

#### AFP

|Componente|Tasa|Descuento|
|---|---|---|
|Fondo de pensión|`D37`: 10.00%|`E37`: S/ 37.28|
|Comisión variable|`D38`: 1.47%|`E38`: S/ 5.48|
|Prima de seguro|`D39`: 1.70%|`E39`: S/ 6.34|
|**Total AFP**|`D40`: **13.17%**|`E40`: **S/ 49.10**|

En `F34:K34` se resume:

```text
AFP 13,17% × S/ 372,85 = (S/ 49,1)```

#### ONP

- `B42`: ONP.
- `D42`: 13.00%.
- `E42`: descuento calculado cuando se selecciona esta alternativa.

### Vacaciones netas

| Concepto | Importe |
|---|---:|
| Vacaciones brutas | S/ 372.85 |
| Descuento AFP | (S/ 49.10) |
| **Vacaciones por pagar** | **S/ 323.74** |

El resultado aparece en `K35` y se presenta formalmente en `H37:K37`.

---

## 5. Gratificaciones truncas — `'Liquidacion de beneficios'!A46:K55`

Calcula la parte proporcional de la gratificación correspondiente al semestre en el que ocurre el cese.

### Periodo de cálculo

| Elemento | Ubicación | Contenido actual |
|---|---|---|
| Sección | `A46:B46` | 4.- GRATIFICACIONES TRUNCAS |
| Tipo | `C48:E48` | Gratificación por Fiestas Patrias |
| Inicio del semestre | `C49` | 01/01/2025 |
| Fin del semestre | `D49` | 30/06/2025 |
| Periodo efectivo | `C50:K50` | Desde 01/01/2025 hasta 01/04/2025 |

El título cambia automáticamente entre:

- Gratificación por Fiestas Patrias.
- Gratificación por Navidad.

### Cálculo por meses

La fila `51` representa:
```

S/ 1,475.00 ÷ 6 × 3 meses = S/ 737.50

```
| Componente | Celda |
|---|---|
| Remuneración computable | `C51` |
| Divisor semestral | `E51`: 6 |
| Meses computables | `G51`: 3 |
| Resultado | `K51`: S/ 737.50 |

### Cálculo por días

La fila `52` representa:
```

text S/ 1,475.00 ÷ 6 ÷ 30 × 1 día = S/ 8.19

```
| Componente | Celda |
|---|---|
| Remuneración | `C52` |
| Divisor semestral | `E52`: 6 |
| Divisor mensual | `G52`: 30 |
| Días computables | `I52`: 1 |
| Resultado | `K52`: S/ 8.19 |

### Resultado
```

text S/ 737.50 + S/ 8.19 = S/ 745.69

```
- Subtotal: `K53`.
- **Gratificación por pagar:** `H55:K55`.
- Importe final: **S/ 745.69**.

---

## 6. Bonificación especial — `'Liquidacion de beneficios'!A57:K61`

Calcula la bonificación extraordinaria asociada a la gratificación trunca.

### Operación

La fila `59` representa:
```

S/ 745.69 × 9% = S/ 67.11

```
| Componente | Ubicación | Valor |
|---|---|---:|
| Gratificación base | `E59` | S/ 745.69 |
| Tasa | `G59` | 9% |
| Bonificación | `K59` | S/ 67.11 |

### Resultado

- `H61:J61`: **BONIFICACIÓN ESPECIAL A PAGAR**
- `K61`: **S/ 67.11**

La tasa del 9% sustituye el aporte que normalmente efectuaría el empleador a EsSalud sobre la gratificación.

---

## 7. Total de la liquidación — `'Liquidacion de beneficios'!G64:K67`

Consolida todos los beneficios calculados.

### Composición

| Concepto | Importe |
|---|---:|
| CTS por pagar | S/ 448.68 |
| Vacaciones netas | S/ 323.74 |
| Gratificación trunca | S/ 745.69 |
| Bonificación especial | S/ 67.11 |
| **Total por recibir** | **S/ 1,585.23** |

La fórmula de `K64` es:
```

excel =K28+K37+K55+K61 ```

### Importe en letras

`C67:K67` contiene:

> SON: UN MIL QUINIENTOS OCHENTA Y CINCO Y 23/100 NUEVOS SOLES

En Excel se genera mediante una función personalizada de conversión numérica a letras. En Go debe implementarse una función equivalente.

---

## 8. Declaración de conformidad — `'Liquidacion de beneficios'!C68:K69`

Contiene la manifestación del trabajador de haber recibido íntegramente la liquidación:

> FIRMO LA PRESENTE COMO CONSTANCIA DE HABER RECIBIDO LA INTEGRIDAD DE MI LIQUIDACIÓN DE BENEFICIOS SOCIALES DE CONFORMIDAD AL D.LEG. Nº 650 Y NO TENIENDO NADA QUE RECLAMAR.

Este texto debe mostrarse en dos líneas, alineado a la izquierda y sin fondo.

---

## 9. Fecha, firma e identificación — `'Liquidacion de beneficios'!C71:J75`

### Fecha de emisión

- `C71:E71`: fecha actual del documento.
- Presentación actual: **sábado, 15 de agosto de 2026**.

### Firma

- `J74`: nombre del trabajador, tomado de `D3`.
- Se muestra sobre una línea horizontal destinada a la firma.

### Documento de identidad

- `I75`: etiqueta **DNI:**
- `J75`: **45323233**

Este bloque cierra formalmente el documento y vincula la firma con la identidad del trabajador.