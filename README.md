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

Create a user and the database:
```
sudo -u postgres psql
postgres=# create database timeseries;
postgres=# create user timeseries with encrypted password '<password>';
postgres=# grant all privileges on database timeseries to timeseries;
```

Run the ddl script as the new user against the new database: 
```
psql -h localhost -U timeseries -d timeseries -f measurement_ddl.sql
```

```
sudo mkdir -p /opt/sm-postgres/bin
sudo mkdir /etc/opt/sm-postgres
sudo cp sm-postgres /opt/sm-postgres/bin
```

Create `/etc/systemd/system/sm-postgres.service`
```
[Unit]
Description=Smart meter postgres update
After=nats.service

[Service]
# Url looks like: postgres://user:pass@host:5432/dbname"
Environment="DATABASE_URL=postgres://timeseries:<pass>@localhost:5432/timeseries"
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
