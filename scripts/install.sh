#!/bin/bash
id -u leaps &>/dev/null || useradd -p leaps leaps

cp ./leaps /usr/sbin/leaps
cp ./config/leaps_all_fields.js /etc/leaps.js
cp ./config/init.d/leaps /etc/init.d/leaps

chmod 755 /etc/init.d/leaps
chown root:root /etc/init.d/leaps

sudo update-rc.d leaps defaults
sudo update-rc.d leaps enable
