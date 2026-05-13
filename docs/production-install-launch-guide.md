# Guía de puesta en producción del installer bootstrap

## Objetivo general

Poner en producción el flujo de instalación bootstrap de `nrcc` para que `https://get.nrcc.dev/install.sh` sirva el instalador publicado en GitHub Pages y permita completar el flujo:

```bash
curl -fsSL https://get.nrcc.dev/install.sh | sh
sudo nrcc install
```

## Prerrequisitos

- Tener permisos de administración sobre el repositorio `composedof2/nrcc`.
- Tener acceso a GitHub Pages del repositorio.
- Tener acceso al proveedor DNS de `nrcc.dev`.
- Confirmar que estos archivos ya están en la rama publicada:
  - `docs/index.html`
  - `docs/install.sh`
  - `docs/CNAME`
  - `.github/workflows/release.yml`
- Tener un host Linux o macOS para validar la instalación final.
- Tener `curl` disponible en la máquina de prueba.

## Checklist de GitHub Pages

1. Entrar en `GitHub -> composedof2/nrcc -> Settings -> Pages`.
2. En `Build and deployment`, seleccionar:
   - `Source`: `Deploy from a branch`
   - `Branch`: `main`
   - `Folder`: `/docs`
3. Guardar la configuración.
4. Confirmar que GitHub detecta el custom domain `get.nrcc.dev`.
5. Esperar a que Pages publique el sitio.
6. Verificar que `https://composedof2.github.io/nrcc/` o la URL de Pages asociada responde antes de validar el dominio final.

Notas:

- `docs/CNAME` ya debe contener `get.nrcc.dev`.
- Si Pages muestra advertencias de dominio, normalmente desaparecen cuando el DNS queda bien propagado.

## Checklist de DNS y CNAME

1. Ir al panel DNS de `nrcc.dev`.
2. Crear o actualizar el registro:

```dns
Tipo: CNAME
Nombre: get
Destino: composedof2.github.io
```

3. Eliminar registros conflictivos para `get.nrcc.dev` si existen, especialmente otros `A`, `AAAA` o `CNAME` duplicados.
4. Esperar propagación DNS.
5. Verificar resolución:

```bash
dig +short get.nrcc.dev
dig +short CNAME get.nrcc.dev
```

Resultado esperado:

- `dig +short CNAME get.nrcc.dev` debe devolver `composedof2.github.io.` o equivalente.
- La resolución final puede tardar varios minutos en estabilizarse según TTL y proveedor DNS.

## Checklist de release o tag de prueba

El workflow `.github/workflows/release.yml` se ejecuta al hacer push de tags `v*`.

1. Elegir una versión de prueba, por ejemplo `v0.1.0` o la siguiente versión disponible.
2. Crear el tag anotado localmente:

```bash
git tag -a v0.1.0 -m "Test release v0.1.0"
```

3. Publicar el tag:

```bash
git push origin v0.1.0
```

4. Ir a `GitHub -> Actions` y confirmar que el workflow `Release` termina en verde.
5. Ir a `GitHub -> Releases` y verificar que exista la release para ese tag.
6. Confirmar que la release adjunta al menos:
   - `nrcc-linux-amd64`
   - `nrcc-linux-arm64`
   - `nrcc-linux-armv7`
   - `nrcc-darwin-amd64`
   - `nrcc-darwin-arm64`
   - `nrcc-windows-amd64.exe`
   - `SHA256SUMS`
   - `*.sha256`

## Checklist de validación del installer y de `nrcc install`

### Validación del bootstrap remoto

1. Confirmar que el script publicado responde:

```bash
curl -fsSL https://get.nrcc.dev/install.sh | sed -n '1,20p'
```

2. Validar instalación en una máquina limpia o controlada:

```bash
curl -fsSL https://get.nrcc.dev/install.sh | sh
```

3. Confirmar que el binario quedó instalado:

```bash
which nrcc
nrcc --version
```

4. Validar versión fijada:

```bash
NRCC_VERSION=v0.1.0 curl -fsSL https://get.nrcc.dev/install.sh | sh
nrcc --version
```

### Validación de `nrcc install`

1. Ejecutar:

```bash
sudo nrcc install
```

2. Confirmar que el servicio queda instalado y arrancado.
3. Verificar estado:

```bash
systemctl status nrcc
```

4. Abrir `http://localhost:3001`.
5. Completar el setup inicial del usuario administrador.

Si no usas `systemd`, valida al menos que el binario descargado arranque y que el siguiente paso mostrado por el installer sea coherente para tu plataforma objetivo.

## Cómo verificar que `get.nrcc.dev` funciona

Usar estas comprobaciones en este orden:

1. DNS:

```bash
dig +short CNAME get.nrcc.dev
```

2. Respuesta HTTP del sitio:

```bash
curl -I https://get.nrcc.dev/
```

3. Respuesta del installer:

```bash
curl -I https://get.nrcc.dev/install.sh
```

4. Contenido esperado del installer:

```bash
curl -fsSL https://get.nrcc.dev/install.sh | grep 'REPO="composedof2/nrcc"'
```

5. Flujo end-to-end:

```bash
curl -fsSL https://get.nrcc.dev/install.sh | sh
sudo nrcc install
```

Resultado esperado:

- `https://get.nrcc.dev/` sirve la landing de instalación.
- `https://get.nrcc.dev/install.sh` devuelve el script shell correcto.
- El installer resuelve la última release desde GitHub y descarga el binario correspondiente.

## Troubleshooting básico

### GitHub Pages no publica

- Revisar `Settings -> Pages` y confirmar `main + /docs`.
- Confirmar que `docs/index.html` existe en la rama publicada.
- Confirmar que `docs/CNAME` contiene exactamente `get.nrcc.dev`.
- Esperar unos minutos y reintentar; Pages no siempre actualiza de inmediato.

### `get.nrcc.dev` no resuelve o resuelve mal

- Revisar que el registro sea `CNAME get -> composedof2.github.io`.
- Eliminar registros duplicados o conflictivos para el subdominio `get`.
- Comprobar propagación con `dig` desde más de una red si hace falta.

### `curl https://get.nrcc.dev/install.sh` devuelve 404

- Confirmar que GitHub Pages publica la carpeta `/docs`.
- Confirmar que el archivo se llama exactamente `docs/install.sh`.
- Confirmar que el despliegue de Pages ya terminó.

### El installer no encuentra la última versión

- Verificar que existe una GitHub Release publicada.
- Verificar que el tag sigue el patrón `v*` para disparar el workflow.
- Probar con `NRCC_VERSION=<tag>` para aislar si el problema está en `latest`.

### El installer falla al verificar checksum

- Revisar que la release incluya los archivos `*.sha256` y `SHA256SUMS`.
- Confirmar que los sidecars `.sha256` apuntan al nombre normalizado `nrcc` que espera el script.

### `sudo nrcc install` falla

- Revisar permisos, dependencias del host y si `systemd` está disponible.
- Ejecutar `nrcc --help` y `nrcc install --help` para revisar flags disponibles.
- Validar manualmente que el binario descargado corre antes de instalar el servicio.

## Próximos pasos después de validar

1. Crear la release real de lanzamiento si la prueba fue satisfactoria.
2. Actualizar la versión de ejemplo en documentación si hace falta.
3. Añadir la validación de `get.nrcc.dev/install.sh` al checklist operativo de releases.
4. Considerar una comprobación automatizada post-release para verificar:
   - `https://get.nrcc.dev/`
   - `https://get.nrcc.dev/install.sh`
   - instalación con `NRCC_VERSION=<tag>`
5. Anunciar oficialmente el flujo one-liner cuando el dominio y la release queden estables.
