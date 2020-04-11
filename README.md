# Build
`./build.sh`

# Install sm-reader
```
sudo mkdir -p /opt/sm-reader/bin
sudo mkdir /etc/opt/sm-reader
sudo cp sm-reader /opt/sm-reader/bin
```

Create `/etc/systemd/system/sm-reader.service`
```
[Unit]
Description=Smart meter reader
After=nats.service

[Service]
ExecStart=/opt/sm-reader/bin/sm-reader /dev/ttyUSB0
WorkingDirectory=/etc/opt/sm-reader
StandardOutput=inherit
StandardError=inherit
Restart=always

[Install]
WantedBy=multi-user.target
```

Start and enable it so that it is started on boot:
```
sudo systemctl start sm-reader
sudo systemctl enable sm-reader
```

# Install sm-postgres
```
sudo mkdir -p /opt/sm-postgres/bin
sudo mkdir /etc/opt/sm-postgres
sudo cp sm-reader /opt/sm-postgres/bin
```

Create `/etc/systemd/system/sm-postgres.service`
```
[Unit]
Description=Smart meter postgres update
After=nats.service

[Service]
Environment="DATABASE_URL=postgres://user:pass@host:5432/dbname"
ExecStart=/opt/sm-postgres/bin/sm-postgres
WorkingDirectory=/etc/opt/sm-postgres
StandardOutput=inherit
StandardError=inherit
Restart=always

[Install]
WantedBy=multi-user.target
```

Start and enable it so that it is started on boot:
```
sudo systemctl start sm-postgres
sudo systemctl enable sm-postgres
```
