# Deploying 

```sh
# Create main binary
make build 
rsync -avz -e ssh main user@SERVER_IP:/home/user/project_name

# Manually
sudo apt-get install screen
cd project_name
./main
```

## Add init.d startup script
```sh
cd /etc/init.d/
touch my-service.sh
vim my-service.sh
chmod +x my-service.sh
/etc/init.d/my-service.sh start
update-rc.d my-service.sh defaults
```
 
/etc/init.d/my-service.sh script
```bash
#!/bin/sh
### BEGIN INIT INFO
# Provides:          main_go
# Required-Start:    $network $syslog
# Required-Stop:     $network $syslog
# Default-Start:     2 3 4 5
# Default-Stop:      0 1 6
# Short-Description: Start go server at boot time
# Description:       your cool web app
### END INIT INFO

case "$1" in
  start)
    exec /var/www/project_name/main &
    ;;
  stop)
    kill $(lsof -t -i:4000)
    ;;
  *)
    echo $"Usage: $0 {start|stop}"
    exit 1
esac
exit 0
```

### or Systemd Script
```bash
sudo nano /etc/systemd/system/deals.service
```

In deals.service,
```
[Unit]
Description=instance to serve deals
After=network.target

[Service]
User=root
Group=www-data

ExecStart=/var/www/go/deals # or path to binary

[Install]
WantedBy=multi-user.target
```

Enable script
```bash
sudo systemctl start jobs
sudo systemctl enable jobs # run on startup
```

## Configuring Nginx to Proxy Requests
```bash
sudo nano /etc/nginx/sites-available/deals
```
Replace port and domain
```
listen 80 default_server;
server_name example.com www.example.com 
server_name dev ipv6only=on;
  
location / {
        proxy_pass http://localhost:8080;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection 'upgrade';
        proxy_set_header Host $host;
        proxy_cache_bypass $http_upgrade;
    }
}
```

Enable server config
```bash
sudo ln -s /etc/nginx/sites-available/deals /etc/nginx/sites-enabled
sudo nginx -t
sudo systemctl restart nginx
```