# Deploy en Oracle Cloud Free Tier

## 1. Crear cuenta en Oracle Cloud

1. Ve a https://cloud.oracle.com y crea una cuenta (necesitas tarjeta pero NO te cobran)
2. Selecciona una región (Phoenix o Ashburn tienen más disponibilidad)

## 2. Crear la VM

1. En la consola, ve a **Compute → Instances → Create Instance**
2. Configura:
   - Image: **Oracle Linux 8** o **Ubuntu 22.04**
   - Shape: **VM.Standard.A1.Flex** (ARM) — 1 OCPU, 1 GB RAM
   - Networking: crea una VCN con subnet pública
   - SSH key: sube tu clave pública (`~/.ssh/id_rsa.pub`)
3. Click **Create**

## 3. Abrir puertos

En **Networking → Virtual Cloud Networks → tu VCN → Security Lists**:
- Agrega regla de ingreso: **TCP puerto 8080** desde 0.0.0.0/0

## 4. Compilar y subir

Desde tu máquina local (la VM es ARM64):

```bash
# Compilar para Linux ARM64
cd CostHandler_bot
GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o costhandler ./cmd/main.go

# Subir el binario a la VM
scp costhandler ubuntu@<IP_DE_TU_VM>:~/

# Subir el .env
scp ../.env ubuntu@<IP_DE_TU_VM>:~/
```

## 5. Configurar en la VM

```bash
ssh ubuntu@<IP_DE_TU_VM>

# Hacer el binario ejecutable
chmod +x costhandler

# Probar que funcione
source .env
./costhandler
```

## 6. Crear servicio systemd (para que corra siempre)

```bash
sudo tee /etc/systemd/system/costhandler.service << 'EOF'
[Unit]
Description=CostHandler Bot
After=network.target

[Service]
User=ubuntu
WorkingDirectory=/home/ubuntu
EnvironmentFile=/home/ubuntu/.env
ExecStart=/home/ubuntu/costhandler
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
EOF

sudo systemctl daemon-reload
sudo systemctl enable costhandler
sudo systemctl start costhandler
```

## 7. Verificar

```bash
# Ver logs
sudo journalctl -u costhandler -f

# Ver status
sudo systemctl status costhandler
```

## 8. Actualizar (cuando hagas cambios)

```bash
# En tu máquina local
GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o costhandler ./cmd/main.go
scp costhandler ubuntu@<IP_DE_TU_VM>:~/

# En la VM
ssh ubuntu@<IP_DE_TU_VM>
sudo systemctl restart costhandler
```

## Alternativa: Deploy con Docker

Si prefieres Docker en la VM:

```bash
# En tu máquina local — build para ARM64
docker buildx build --platform linux/arm64 -t costhandler .

# O en la VM directamente
scp -r . ubuntu@<IP_DE_TU_VM>:~/costhandler/
ssh ubuntu@<IP_DE_TU_VM>
cd costhandler
docker build -t costhandler .
docker run -d --name costhandler --env-file .env -p 8080:8080 -v ~/data:/home/appuser/data costhandler
```

## Notas

- La DB SQLite se guarda en la VM. Haz backups periódicos: `scp ubuntu@<IP>:~/expenses.db ./backup/`
- Si la VM se reinicia, systemd levanta el bot automáticamente
- El free tier incluye 200 GB de almacenamiento — más que suficiente
